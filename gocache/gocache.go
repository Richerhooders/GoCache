package gocache

import (
	"fmt"
	// pb "gocache/gocachepb"
	"gocache/singleflight"
	"log"
	"sync"
	"time"
)
// gocache 模块提供比cache模块更高一层抽象的能力
// 换句话说，实现了填充缓存/命名划分缓存的能力


// A Getter loads data for a key
// 定义接口 Getter 和 回调函数 Get(key string)([]byte, error)，参数是 key，返回值是 []byte。
type Getter interface {
	retrieve(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function
// 定义函数类型 GetterFunc，并实现 Getter 接口的 Get 方法。任何具有相应签名的函数都可以被视为一个 GetterFunc。
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
// GetterFunc 还定义了 Get 方式，并在 Get 方法中调用自己
// 函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
func (f GetterFunc) retrieve(key string) ([]byte, error) {
	return f(key)
}

// A Group is a cache namespace and associated data loaded spread over
// Group 是 GoCache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程。
// 一个Group可以认为是一个缓存的命名空间，每个Group拥有一个唯一的名称name,
// 比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info，缓存学生课程的命名为 courses。
type Group struct {
	name      string
	getter    Getter //缓存未命中时获取源数据的回调(callback)。
	mainCache cache  //一开始实现的并发缓存。
	server    Picker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	flight    *singleflight.Flight
	Expire    time.Duration
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
// 构建函数 NewGroup 用来实例化 Group，并且将 group 存储在全局变量 groups 中。
func NewGroup(name string, cacheBytes int64, expire time.Duration, getter Getter,) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{capacity: cacheBytes},
		flight:    &singleflight.Flight{},
		Expire:    expire,
	}
	groups[name] = g
	return g
}

// 流程2 RegisterPeers registers a PeerPicker for choosing remote peer
// 实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中。// RegisterSvr 为 Group 注册 Server
func (g *Group) RegisterPeers(peers Picker) {
	if g.server != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.server = peers
}


// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
// 用来获取特定名称的group
func GetGroup(name string) *Group {
	mu.RLock()   //只读锁 RLock()，因为不涉及任何冲突变量的写操作。
	g := groups[name]
	mu.RUnlock()
	return g
}

func DestroyGroup(name string) {
	g := GetGroup(name)
	if g != nil {
		svr  := g.server.(*server)
		svr.Stop()
		delete(groups,name)
		log.Printf("Destroy cache [%s %s]", name, svr.addr)
	}
}

/*
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

// get value for a key from cache
// 实现了流程1、3
// 从 mainCache 中查找缓存，如果存在则返回缓存值。
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GoCache] hit")
		return v, nil
	}
	//缓存不存在，则调用 load 方法
	return g.load(key)
}


// load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取）
// 修改 load 方法，使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。若是本机节点或失败，则回退到 getLocally()。
func (g *Group) load(key string) (value ByteView, err error) {
	// return g.getLocally(key)

	//each key is only fetched oncce(either locally or remotely)
	//regardless of the number of concurrent callers
	//修改 load 函数，将原来的 load 的逻辑，使用 g.loader.Do 包裹起来即可，这样确保了并发场景下针对相同的 key，load 过程只会调用一次。
	viewi, err := g.flight.Fly(key, func() (interface{}, error) { //任何类型都满足空接口，确保func()函数只执行一次
		if g.server != nil {
			if fetcher, ok := g.server.Pick(key); ok {
				if bytes, err := fetcher.Fetch(g.name,key); err == nil {
					return ByteView{b: cloneBytes(bytes)}, nil
				}
				log.Println("[GoCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.retrieve(key) //调用get方法时，就已经用peer的*httpGetter的内容（存的ip地址）去访问数据了。
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value, g.Expire)
}
