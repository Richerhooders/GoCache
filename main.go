package main

import (
	"gocache"
	"fmt"
	"log"
	"net/http"
)

//我们使用 map 模拟了数据源 db。
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	//创建一个名为 scores 的 Group，若缓存为空，回调函数会从 db 中获取数据并返回。
	gocache.NewGroup("scores", 2<<10, gocache.GetterFunc(
		func(key string) ([]byte, error) { //从 db 中获取数据并返回。
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	addr := "localhost:8000"
	peers := gocache.NewHTTPPool(addr)
	log.Println("gocache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}