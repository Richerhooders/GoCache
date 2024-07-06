// Copyright 2021 Peanutzhen. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package registry

import (
	"context"
	"testing"
	"time"
	
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
)

func TestEtcdAdd(t *testing.T) {
	cli, _ := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	// 创建一个租约 配置5秒过期
	resp, err := cli.Grant(context.Background(), 5)
	if err != nil {
		t.Fatalf(err.Error())
	}
	err = etcdAdd(cli, resp.ID, "test", "127.0.0.1:6324")
	if err != nil {
		t.Fatalf(err.Error())
	}
}


func TestRegister(t *testing.T) {
	// Start an embedded etcd server for testing
	cfg := embed.NewConfig()
	cfg.Dir = "default.etcd"
	e, err := embed.StartEtcd(cfg)
	if err != nil {
		t.Fatalf("Failed to start embedded etcd: %v", err)
	}
	defer e.Close()

	// Wait until the etcd server is ready
	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(10 * time.Second):
		t.Fatal("Took too long to start etcd")
	}

	// Create an etcd client
	cli, err := clientv3.New( clientv3.Config{
		Endpoints:   []string{"localhost:2468"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create etcd client: %v", err)
	}
	defer cli.Close()

	// Define the service information
	serviceName := "testService"
	serviceAddr := "localhost:9999"
	stopCh := make(chan error, 1)

	// Call the Register function to register the service
	go func() {
		if err := Register(serviceName, serviceAddr, stopCh); err != nil {
			t.Errorf("Register failed: %v", err)
		}
	}()

	// Give some time for registration to process
	time.Sleep(1 * time.Second)

	// Check if the service is registered
	resp, err := cli.Get(context.Background(), serviceName)
	if err != nil {
		t.Fatalf("Failed to get service from etcd: %v", err)
	}
	if len(resp.Kvs) == 0 {
		t.Error("Service not registered")
	}

	// Clean up by sending a stop signal
	stopCh <- nil

	// Wait to ensure cleanup is processed
	time.Sleep(1 * time.Second)
}
