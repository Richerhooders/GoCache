# GoCache
```
├── gocache
│   ├── byteview.go    // 缓存值的抽象与封装 
│   ├── cache.go        // 并发控制 
│   ├── consistenthash
│   │   ├── consistenthash.go
│   │   └── consistenthash_test.go
│   ├── gocache.go       // 负责与外部交互，控制缓存存储和获取的主流程 
│   ├── gocachepb
│   │   ├── gocachepb.pb.go
│   │   └── gocachepb.proto
│   ├── gocache_test.go
│   ├── go.mod
│   ├── http.go
│   ├── lru
│   │   ├── lru.go    // lru 缓存淘汰策略 
│   │   └── lru_test.go
│   ├── peers.go
│   └── singleflight
│       ├── singleflight.go
│       └── singleflight_test.go
├── go.mod
├── go.sum
├── main.go
├── README.md
└── run.sh
```