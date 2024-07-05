package gocache

import (
	"testing"
	"time"
)


// 测试服务器的启动和停止
func TestServerStartAndStop(t *testing.T) {
	serverAddr := "localhost:9999"
	svr, err := NewServer(serverAddr)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// 异步启动服务器
	go func() {
		if err := svr.Start(); err != nil {
			t.Errorf("Failed to start server: %v", err)
		}
	}()

	// 给服务器一些时间来启动
	time.Sleep(time.Second * 1)

	// 停止服务器
	svr.Stop()

	// 检查服务器是否正确停止
	if svr.status {
		t.Errorf("Server did not stop correctly")
	}
}

// 测试服务器的对等点选择功能
func TestServerPeerSelection(t *testing.T) {
	serverAddr := "localhost:9999"
	peerAddr1 := "localhost:9998"
	peerAddr2 := "localhost:9997"

	svr, err := NewServer(serverAddr)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// 设置对等点
	svr.SetPeers(peerAddr1, peerAddr2)

	// 选择对等点
	key := "somekey"
	fetcher, ok := svr.Pick(key)
	if !ok {
		t.Errorf("Failed to pick a peer for key: %s", key)
	}

	// 验证返回的Fetcher是否为期望的客户端
	if _, ok := fetcher.(*client); !ok {
		t.Errorf("Picked peer is not a client type")
	}
}

// 测试完整的启动到选择对等点再到停止的流程
func TestServerFullCycle(t *testing.T) {
	serverAddr := "localhost:9999"
	peerAddr1 := "localhost:9998"
	peerAddr2 := "localhost:9997"

	svr, err := NewServer(serverAddr)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// 启动服务器
	go func() {
		if err := svr.Start(); err != nil {
			t.Errorf("Failed to start server: %v", err)
		}
	}()

	// 设置对等点
	svr.SetPeers(peerAddr1, peerAddr2)

	// 选择对等点
	key := "somekey"
	_, ok := svr.Pick(key)
	if !ok {
		t.Errorf("Failed to pick a peer for key: %s", key)
	}

	// 停止服务器
	svr.Stop()

	// 检查服务器是否正确停止
	if svr.status {
		t.Errorf("Server did not stop correctly")
	}
}
