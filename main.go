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

	// 多个节点的地址
	addrs := []string{"localhost:9999", "localhost:9998", "localhost:9997"}
	groupname := []string{"9999", "9998", "9997"}
	var Group []*gocache.Group
	// 创建并启动每个服务实例
	for i, addr := range addrs {
		svr, err := gocache.NewServer(addr)
		if err != nil {
			log.Fatalf("Failed to create server on %s: %v", addr, err)
		}
		svr.SetPeers(addrs...)
		// 创建每个server的专属Group
		group := gocache.NewGroup(groupname[i], 2<<10, time.Second, gocache.GetterFunc(
			func(key string) ([]byte, error) {
				log.Println("[Mysql] search key", key)
				if v, ok := mysql[key]; ok {
					return []byte(v), nil
				}
				return nil, fmt.Errorf("%s not exist", key)
			}))

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

	time.Sleep(5 * time.Second) // 等待服务器启动

	// 发出几个Get请求，分开发送，保证第二次Get缓存命中
	// 可以向任意服务器发起请求
	var wg sync.WaitGroup
	wg.Add(2)
	go GetTomScore(Group[0], &wg)
	go GetJackScore(Group[0], &wg)
	wg.Wait()

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
