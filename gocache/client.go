package gocache

import (
	"context"
	"fmt"
	pb "gocache/gocachepb"
	"gocache/registry"
	"time"
	"google.golang.org/grpc"


	clientv3 "go.etcd.io/etcd/client/v3"
)

// client 模块实现gocache访问其他远程节点 从而获取缓存的能力
type client struct {
	name string  // 服务名称 pcache/ip:addr
	
	etcdClient *clientv3.Client
	grpcConn   *grpc.ClientConn
	cacheClient pb.GroupCacheClient
}

// 使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。 getFromPeer 从remote peer获取对应缓存值
func (c *client) Fetch(group string, key string) ([]byte, error) {
	//创建一个etcd client
	cli,err := clientv3.New(defaultEtcdConfig) 
	if err != nil {
		return nil,err
	}
	defer cli.Close()
	// 发现服务 取得与服务的连接 通过etcd获取gRPC服务的地址，并建立连接。
	conn,err := registry.EtcdDial(cli,c.name);
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	//如果连接成功，会使用这个连接创建一个新的gRPC客户端
	grpcClient := pb.NewGroupCacheClient(conn)
	//设置一个十秒超时时间，
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	//发送一个gPRC请求到远程服务，请求包括组名和键名，
	resp, err := grpcClient.Get(ctx, &pb.Request{
		Group: group,
		Key:   key,
	})
	if err != nil {
		return nil, fmt.Errorf("could not get %s/%s from peer %s", group, key, c.name)
	}

	return resp.GetValue(), nil
}

//用于创建新的client实例，接收一个服务名作为参数，这个服务名是etcd中注册的服务名，用于在 Fetch 方法中与远程服务通信。
func NewClient(service string) *client {
	return &client{name: service}
}

// 测试Client是否实现了Fetcher接口，验证 client 类型是否实现了 Fetcher 接口。
//这是 Go 语言的一种常见模式，确保类型正确地实现了接口。这里的 _ Fetcher = (*client)(nil) 是一个编译时的断言，如果 client 没有实现 Fetcher 接口，程序会编译失败。
var _ Fetcher = (*client)(nil)
