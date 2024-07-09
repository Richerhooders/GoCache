package gocache

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	pb "gocache/gocachepb"
	"gocache/registry"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func printConnState(conn *grpc.ClientConn) {
	val := reflect.ValueOf(conn).Elem()
	state := val.FieldByName("state")
	log.Println("Connection state:", state)
}

// client 模块实现gocache访问其他远程节点 从而获取缓存的能力
type client struct {
	name       string // 服务名称 pcache/ip:addr
	etcdClient *clientv3.Client
	conn       *grpc.ClientConn
	mu         sync.Mutex
}

func (c *client) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.etcdClient == nil {
		var err error
		c.etcdClient, err = clientv3.New(defaultEtcdConfig)
		if err != nil {
			return fmt.Errorf("failed to create etcd client: %v", err)
		}
	}

	if c.conn == nil {
		conn, err := registry.EtcdDial(c.etcdClient, c.name)
		if err != nil {
			return fmt.Errorf("failed to dial gRPC server: %v", err)
		}
		c.conn = conn
	}

	return nil
}

// 使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。 getFromPeer 从remote peer获取对应缓存值
func (c *client) Fetch(group string, key string) ([]byte, error) {

	if err := c.initialize(); err != nil {
		log.Printf("Initialization failed: %v", err)
		return nil, err
	}
	log.Println("Initialization successful")
	//如果连接成功，会使用这个连接创建一个新的gRPC客户端
	grpcClient := pb.NewGroupCacheClient(c.conn)
	if grpcClient == nil {
		log.Println("Failed to create gRPC client")
		return nil, fmt.Errorf("failed to create gRPC client")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	//发送一个gPRC请求到远程服务，请求包括组名和键名，
	resp, err := grpcClient.Get(ctx, &pb.Request{Group: group, Key: key})
	if err != nil {
		log.Printf("gRPC call failed: %v", err)
		return nil, fmt.Errorf("could not get %s/%s from peer %s: %v", group, key, c.name, err)
	}
	log.Println("Successfully sent gRPC request")
	return resp.GetValue(), nil
}

// 用于创建新的client实例，接收一个服务名作为参数，这个服务名是etcd中注册的服务名，用于在 Fetch 方法中与远程服务通信。
func NewClient(service string) *client {
	return &client{name: service}
}

// 测试Client是否实现了Fetcher接口，验证 client 类型是否实现了 Fetcher 接口。
// 这是 Go 语言的一种常见模式，确保类型正确地实现了接口。这里的 _ Fetcher = (*client)(nil) 是一个编译时的断言，如果 client 没有实现 Fetcher 接口，程序会编译失败。
var _ Fetcher = (*client)(nil)
