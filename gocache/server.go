// Copyright 2021 Peanutzhen. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gocache

import (
	"context"
	"fmt"
	"gocache/consistenthash"
	pb "gocache/gocachepb"
	"gocache/registry"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// server 模块为gocache之间提供通信能力
// 这样部署在其他机器上的cache可以通过访问server获取缓存
//
//	至于找哪台主机 那是一致性哈希的工作了
//
// 服务器的默认地址
const (
	defaultAddr     = "127.0.0.1:6324"
	defaultReplicas = 50
)

// 配置了 etcd 客户端的默认设置，包括 etcd 服务的端点地址和拨号超时时间。这是用于服务发现和注册的配置，确保服务器可以与 etcd 集群正确通信。
var (
	defaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
)

// server 和 Group 是解耦合的 所以server要自己实现并发控制
type server struct {
	pb.UnimplementedGroupCacheServer //protobuf生成的接口，确保server结构体实现了必须的grpc方法

	addr       string     // format: ip:port
	status     bool       // true: running false: stop
	stopSignal chan error // 通知registry revoke服务
	mu         sync.Mutex
	consHash   *consistenthash.Consistency
	clients    map[string]*client //每个客户端地址对应一个客户端实例
}

// NewServer 创建cache的svr 若addr为空 则使用defaultAddr
func NewServer(addr string) (*server, error) {
	if addr == "" {
		addr = defaultAddr
	}
	if !validPeerAddr(addr) {
		return nil, fmt.Errorf("invalid addr %s, it should be x.x.x.x:port", addr)
	}
	return &server{addr: addr}, nil
}

// Get 实现 GoCache service 的 Get 接口
func (s *server) Get(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	group, key := req.GetGroup(), req.GetKey()
	resp := &pb.Response{}

	log.Printf("[gocache_svr %s] Received RPC Request - Group: %s, Key: %s", s.addr, group, key)
	if key == "" {
		return resp, fmt.Errorf("key is required")
	}

	// 获取缓存组
	g := GetGroup(group)
	if g == nil {
		return resp, fmt.Errorf("group %s not found", group)
	}

	// 尝试从缓存获取数据
	value, err := g.Get(key)
	if err == nil {
		resp.Value = value.ByteSlice()
		return resp, nil
	}

	// 数据不在缓存中，从数据库加载
	view, err := g.getLocally(key)
	if err != nil {
		return nil, fmt.Errorf("failed to load data for key %s: %v", key, err)
	}

	resp.Value = view.ByteSlice()
	return resp, nil
}

// Start 启动cache服务
func (s *server) Start() error {
	s.mu.Lock()
	if s.status == true {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	// -----------------启动服务----------------------
	// 1. 设置status为true 表示服务器已在运行
	// 2. 初始化stop channal,这用于通知registry stop keep alive
	// 3. 初始化tcp socket并开始监听
	// 4. 注册rpc服务至grpc 这样grpc收到request可以分发给server处理
	// 5. 将自己的服务名/Host地址注册至etcd 这样client可以通过etcd
	//    获取服务Host地址 从而进行通信。这样的好处是client只需知道服务名
	//    以及etcd的Host即可获取对应服务IP 无需写死至client代码中
	// ----------------------------------------------
	s.status = true
	s.stopSignal = make(chan error) // 创建一个接收停止信号的通道，这个通道用于从注册服务接收停止或错误信号

	port := strings.Split(s.addr, ":")[1]
	lis, err := net.Listen("tcp", ":"+port) // 启动TCP服务器，监听指定端口
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()             // 创建新的服务器实例
	pb.RegisterGroupCacheServer(grpcServer, s) // 这个服务器实例与 gRPC 服务相关联，允许 gRPC 处理到来的请求。

	// 注册服务至etcd，异步运行服务注册逻辑，避免阻塞主线程
	go func() {
		// Register never return unless stop singnal received
		err := registry.Register("gocache", s.addr, s.stopSignal) //注册服务器的地址到etcd，这样客户端可以通过 etcd 发现并连接到这个服务器。
		if err != nil {
			log.Fatalf(err.Error())
		}
		// Close channel
		close(s.stopSignal)
		// Close tcp listen
		err = lis.Close()
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.Printf("[%s] Revoke service and close tcp socket ok.", s.addr)
	}()

	//log.Printf("[%s] register service ok\n", s.addr)
	s.mu.Unlock()

	// 在之前创建的监听器上服务gRPC请求，这是一个阻塞调用，会持续监听直到服务器关闭
	if err := grpcServer.Serve(lis); s.status && err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}

// SetPeers 将各个远端主机IP配置到Server里
// 这样Server就可以Pick他们了
// 注意: 此操作是*覆写*操作！
// 注意: peersIP必须满足 x.x.x.x:port的格式
func (s *server) SetPeers(peersAddr ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//初始化一个一致性哈希环
	s.consHash = consistenthash.New(defaultReplicas, nil)
	//供的远程节点地址注册到一致性哈希环中
	s.consHash.Register(peersAddr...)
	s.clients = make(map[string]*client)
	for _, peerAddr := range peersAddr {
		if !validPeerAddr(peerAddr) {
			panic(fmt.Sprintf("[peer %s] invalid address format, it should be x.x.x.x:port", peerAddr))
		}
		//对于每一个有效的节点地址，创建并注册新的客户端实例
		service := fmt.Sprintf("gocache/%s", peerAddr)
		s.clients[peerAddr] = NewClient(service) // peerAddr -> gocache/peerAddr
		//registry.Register(service,peerAddr,make(chan error, 1))
	}
}

// Pick 根据一致性哈希选举出key应存放在的cache
// return false 代表从本地获取cache
func (s *server) Pick(key string) (Fetcher, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	peerAddr := s.consHash.GetPeer(key) //节点地址
	// Pick itself
	if peerAddr == s.addr {
		log.Printf("ooh! pick myself, I am %s\n", s.addr)
		return nil, false
	}
	log.Printf("[cache %s] pick remote peer: %s\n", s.addr, peerAddr)
	return s.clients[peerAddr], true

}

// Stop 停止server运行 如果server没有运行 这将是一个no-op
func (s *server) Stop() {
	s.mu.Lock()
	if s.status == false {
		s.mu.Unlock()
		return
	}
	s.stopSignal <- nil // 发送停止keepalive信号
	s.status = false    // 设置server运行状态为stop
	s.clients = nil     // 清空一致性哈希信息 有助于垃圾回收
	s.consHash = nil
	s.mu.Unlock()
}

// 测试Server是否实现了Picker接口
var _ Picker = (*server)(nil)
