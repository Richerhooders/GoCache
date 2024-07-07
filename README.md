# GoCache
The gRPC implementation of groupcache: A high performance, open source, using RPC framework that communicated with each cache node. Cache service can register to etcd, and each cache client can dicovery the service list by etcd.For more information see the groupcache, or goache.

groupcache的gRPC实现：一个高性能、开源、使用RPC框架与每个缓存节点进行通信。

缓存服务可以注册到etcd，每个缓存客户端都可以通过etcd发现服务列表。

改进LRU cache，使其具备TTL的能力，以及改进锁的粒度，提高并发度。

将单独 lru 算法实现改成多种算法可选（lru、lfu、arc、hashlru(lru-k)、hashlfu(lfu-k)）

根据需要的不同缓存淘汰算法,使用对应的调用方式(尚未实现)

## Prerequisites
- Golang 1.16 or later

- Etcd v3.4.0 or later

- gRPC-go v1.38.0 or later

- protobuf v1.26.0 or later


## Usage
```
// example.go file
// 运行前，你需要在本地启动Etcd实例，作为服务中心。

package main

import (
	"fmt"
	"gocache"
	"log"
	"sync"
	"time"
)

func main() {
	// 模拟MySQL数据库，用于从数据源获取值
	var mysql = map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}

	// 服务实例的地址
	addrs := []string{"localhost:9999", "localhost:9998", "localhost:9997"}
	var Group []*gocache.Group
	// 创建并启动每个服务实例
	for _, addr := range addrs {
		svr, err := gocache.NewServer(addr)
		if err != nil {
			log.Fatalf("Failed to create server on %s: %v", addr, err)
		}
		svr.SetPeers(addrs...)
		// 创建每个server的专属Group
		group := gocache.NewGroup("scores", 2<<10, time.Second, gocache.GetterFunc(
			func(key string) ([]byte, error) {
				log.Println("[Mysql] search key", key)
				if v, ok := mysql[key]; ok {
					return []byte(v), nil
				}
				return nil, fmt.Errorf("%s not exist", key)
			})) // 这里假设NewGroup的构造函数可以接受server作为参数
		
		// 将服务与group绑定
		group.RegisterPeers(svr)
		Group = append(Group, group)
		// 启动服务
		go func() {
			// Start将不会return 除非服务stop或者抛出error
			err = svr.Start()
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	log.Println("gocache is running at", addrs)

	time.Sleep(3 * time.Second) // 等待服务器启动

	// 发出几个Get请求
	var wg sync.WaitGroup
	wg.Add(2)
	go GetTomScore(Group[0], &wg)
	go GetJackScore(Group[0], &wg)
	wg.Wait()

	wg.Add(2)
	go GetTomScore(Group[0], &wg)
	go GetJackScore(Group[0], &wg)
	wg.Wait()
}

func GetTomScore(group *gocache.Group, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("get Tom...")
	view, err := group.Get("Tom")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(view.String())
}
func GetJackScore(group *gocache.Group, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("get Jack...")
	view, err := group.Get("Jack")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(view.String())
}
```
## 性能对比
hashlru 与 lru 性能对比

| 算法            | 耗时     |
|-----------------|----------|
| lru             | 220.2s   |
| hashlru-2分区   | 267.75s  |
| hashlru-4分区   | 137.36s  |
| hashlru-8分区   | 22.4s    |
| hashlru-16分区  | 23.57s   |
| hashlru-32分区  | 16.84s   |
| hashlru-64分区  | 15.29s   |

 hashlfu 与 lfu 性能对比

| 算法           | 耗时      |
|----------------|-----------|
| lru            | 220.92s   |
| hashlfu-2分区  | 231.28s   |
| hashlfu-4分区  | 72.74s    |
| hashlfu-8分区  | 20.33s    |
| hashlfu-16分区 | 17.76s    |
| hashlfu-32分区 | 16.93s    |
| hashlfu-64分区 | 16.03s    |

## hash算法减少耗时原因:

LruCache在高QPS下的耗时增加原因分析：

线程安全的LruCache中有锁的存在。每次读写操作之前都有加锁操作，完成读写操作之后还有解锁操作。 在低QPS下，锁竞争的耗时基本可以忽略；但是在高QPS下，大量的时间消耗在了等待锁的操作上，导致耗时增长。

HashLruCache适应高QPS场景：

针对大量的同步等待操作导致耗时增加的情况，解决方案就是尽量减小临界区。引入Hash机制，对全量数据做分片处理，在原有LruCache的基础上形成HashLruCache，以降低查询耗时。

HashLruCache引入哈希算法，将缓存数据分散到N个LruCache上。查询时也按照相同的哈希算法，先获取数据可能存在的分片，然后再去对应的分片上查询数据。这样可以增加LruCache的读写操作的并行度，减小同步等待的耗时。
