// Copyright 2021 Peanutzhen. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package registry

// register模块提供服务Service注册至etcd的能力

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"log"
	"time"
)

var (
	defaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}
)

// etcdAdd 在租赁模式添加一对kv至etcd
func etcdAdd(c *clientv3.Client, lid clientv3.LeaseID, service string, addr string) error {
	em, err := endpoints.NewManager(c, service)
	if err != nil {
		return err
	}
	//return em.AddEndpoint(c.Ctx(), service+"/"+addr, endpoints.Endpoint{Addr: addr})
	return em.AddEndpoint(c.Ctx(), service+"/"+addr, endpoints.Endpoint{Addr: addr}, clientv3.WithLease(lid))
}

// Register 注册一个服务至etcd
// 注意 Register将不会return 如果没有error的话
func Register(service string, addr string, stop chan error) error {
	// 创建一个etcd client
	cli, err := clientv3.New(defaultEtcdConfig)
	if err != nil {
		return fmt.Errorf("create etcd client failed: %v", err)
	}
	defer cli.Close()
	// 创建一个租约 配置5秒过期，
	resp, err := cli.Grant(context.Background(), 5)
	if err != nil {
		return fmt.Errorf("create lease failed: %v", err)
	}
	leaseId := resp.ID //返回的响应包含租约ID
	// 注册服务,将服务注册到etcd中，绑定到刚才创建的租约上。
	err = etcdAdd(cli, leaseId, service, addr)
	if err != nil {
		return fmt.Errorf("add etcd record failed: %v", err)
	}
	// 设置服务心跳检测，保证租约的活跃状态，KeepAlive 方法返回一个通道，用于接收心跳检测的响应。
	ch, err := cli.KeepAlive(context.Background(), leaseId)
	if err != nil {
		return fmt.Errorf("set keepalive failed: %v", err)
	}

	log.Printf("[%s] register service ok\n", addr)
	for { // 进入一个无限循环
		select {
		case err := <-stop: //stop通道：接收到错误信号时，记录日志并返回错误
			if err != nil {
				log.Println(err)
			}
			return err
		case <-cli.Ctx().Done()://cli.Ctx().Done()：监听etcd客户端的上下文，如果客户端关闭，记录日志并返回
			log.Println("service closed")
			return nil
		case _, ok := <-ch://监听心跳检测的响应，如果通道关闭，记录日志，
			// 监听租约
			if !ok {
				log.Println("keep alive channel closed")
				_, err := cli.Revoke(context.Background(), leaseId)
				return err
			}
			//log.Printf("Recv reply from service: %s/%s, ttl:%d", service, addr, resp.TTL)
		}
	}
}