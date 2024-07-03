# GoCache
```
├── gocache
│   ├── byteview.go    // 缓存值的抽象与封装 封装一个只读的数据结构，防止修改
│   ├── cache.go        // 使用 sync.Mutex 封装 LRU 的几个方法，使之支持并发的读写。
│   ├── consistenthash
│   │   ├── consistenthash.go  //一致性哈希的实现，防止缓存雪崩
│   │   └── consistenthash_test.go
│   ├── gocache.go       // 负责与外部交互，控制缓存存储和获取的主流程 
│   ├── gocachepb
│   │   ├── gocachepb.pb.go
│   │   └── gocachepb.proto // protobuf序列化反序列化实现
│   ├── gocache_test.go
│   ├── go.mod  
│   ├── http.go         //通信模块
│   ├── lru
│   │   ├── lru.go    // lru 缓存淘汰策略 
│   │   └── lru_test.go
│   ├── peers.go        //分布式节点远程访问接口
│   └── singleflight
│       ├── singleflight.go    //singleflight算法防止缓存击穿
│       └── singleflight_test.go
├── go.mod
├── go.sum
├── main.go
├── README.md
└── run.sh
```

安装命令： 
go get github.com/golang/protobuf/proto 
go get google.golang.org/protobuf/reflect/protoreflect 
go get google.golang.org/protobuf/runtime/protoimpl 