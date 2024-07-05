# GoCache
```
第一版
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

第二版：实现使用rpc进行节点间获取缓存，使用etcd
├── gocache
│   ├── byteview.go     / 缓存值的抽象与封装 封装一个只读的数据结构，防止修改
│   ├── byteview_test.go
│   ├── cache.go
│   ├── client.go
│   ├── consistenthash
│   │   ├── consistenthash.go
│   │   └── consistenthash_test.go
│   ├── gocache.go
│   ├── gocachepb
│   │   ├── gocachepb_grpc.pb.go
│   │   ├── gocachepb.pb.go
│   │   └── gocachepb.proto
│   ├── gocache_test.go
│   ├── go.mod
│   ├── go.sum
│   ├── lru
│   │   ├── lru.go
│   │   └── lru_test.go
│   ├── peers.go
│   ├── registry
│   │   ├── discover.go
│   │   ├── mocks_test.go
│   │   ├── register.go
│   │   └── register_test.go
│   ├── server.go
│   ├── server_test.go
│   ├── singleflight
│   │   ├── singleflight.go
│   │   └── singleflight_test.go
│   └── utils.go
├── go.mod
├── main.go
├── README.md
└── run.sh
```
第三版：
增加了缓存过期时间,增加lfu算法,增加了arc算法，lru-K算法、
根据过期时间懒汉式删除过期数据,也可主动刷新过期缓存
现已支持内存算法:
lru、lfu、arc、hashlru、hashlfu

##性能对比
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

##hash算法减少耗时原因:
LruCache在高QPS下的耗时增加原因分析：

线程安全的LruCache中有锁的存在。每次读写操作之前都有加锁操作，完成读写操作之后还有解锁操作。 在低QPS下，锁竞争的耗时基本可以忽略；但是在高QPS下，大量的时间消耗在了等待锁的操作上，导致耗时增长。

HashLruCache适应高QPS场景：

针对大量的同步等待操作导致耗时增加的情况，解决方案就是尽量减小临界区。引入Hash机制，对全量数据做分片处理，在原有LruCache的基础上形成HashLruCache，以降低查询耗时。

HashLruCache引入哈希算法，将缓存数据分散到N个LruCache上。查询时也按照相同的哈希算法，先获取数据可能存在的分片，然后再去对应的分片上查询数据。这样可以增加LruCache的读写操作的并行度，减小同步等待的耗时。

go mod tidy
