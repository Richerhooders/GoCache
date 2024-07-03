package main

import (
	"gocache"
	"fmt"
	"log"
	"net/http"
	"flag"
)

//我们使用 map 模拟了数据源 db。
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

//创建一个名为 scores 的 Group，若缓存为空，回调函数会从 db 中获取数据并返回。
func createGroup() *gocache.Group {
	return gocache.NewGroup("scores", 2<<10, gocache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}
//startCacheServer() 用来启动缓存服务器：创建 HTTPPool，添加节点信息，注册到 gee 中，启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知。
func startCacheServer(addr string,addrs []string,gee *gocache.Group) {
	peers := gocache.NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("gocache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

//startAPIServer() 用来启动一个 API 服务（端口 9999），与用户进行交互，用户感知。
func startAPIServer(apiAddr string,gee* gocache.Group) {
	http.Handle("/api",http.HandlerFunc(
		func(w http.ResponseWriter,r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

//需要命令行传入 port 和 api 2 个参数，用来在指定端口启动 HTTP 服务。
func main() {
	//创建一个名为 scores 的 Group，若缓存为空，回调函数会从 db 中获取数据并返回。
	// gocache.NewGroup("scores", 2<<10, gocache.GetterFunc(
	// 	func(key string) ([]byte, error) { //从 db 中获取数据并返回。
	// 		log.Println("[SlowDB] search key", key)
	// 		if v, ok := db[key]; ok {
	// 			return []byte(v), nil
	// 		}
	// 		return nil, fmt.Errorf("%s not exist", key)
	// 	}))

	// addr := "localhost:8000"
	// peers := gocache.NewHTTPPool(addr)
	// log.Println("gocache is running at", addr)
	// log.Fatal(http.ListenAndServe(addr, peers))
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Gocache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	/*指定了启动 API 服务器，则在一个新的 goroutine 中运行 startAPIServer 函数，确保 API 服务器与缓存服务器并行运行。
	为终端用户提供了一个访问缓存数据的接口，API服务接收到请求后，调用gocache.Group.Get()方法查询这个键。
	gocache.Group实例利用其注册的:"HTTPPool"（缓存节点池）来找到负责该键的节点，并从哪里获取数据，数据被返回给API服务，
	然后API服务将数据返回给请求的用户
	*/
	if api { 
		go startAPIServer(apiAddr, gee)
	}
	//调用 startCacheServer 启动缓存服务器。
	startCacheServer(addrMap[port], []string(addrs), gee)
}