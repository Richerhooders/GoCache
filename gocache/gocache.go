package gocache

import (
	"fmt"
	"log"
	"sync"
)

//A Getter loads data for a key
//定义接口 Getter 和 回调函数 Get(key string)([]byte, error)，参数是 key，返回值是 []byte。
type Getter interface {
	Get(key string)([]byte, error)
}

//A GetterFunc implements Getter with a function
//定义函数类型 GetterFunc，并实现 Getter 接口的 Get 方法。
type GetterFunc func(key string)([]byte ,error) 


//Get implements Getter interface function
//函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
func (f GetterFunc) Get(key string)([]byte,error) {
	return f(key)
}

//A Group is a cache namespace and associated data loaded spread over
//Group 是 GoCache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程。
//一个Group可以认为是一个缓存的命名空间，每个Group拥有一个唯一的名称name,
// 比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info，缓存学生课程的命名为 courses。
type Group struct {
	name    string 
    getter  Getter    //缓存未命中时获取源数据的回调(callback)。
	mainCache  cache   //一开始实现的并发缓存。
	peers 	PeerPicker
}

var (
	mu	sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
// 构建函数 NewGroup 用来实例化 Group，并且将 group 存储在全局变量 groups 中。
func NewGroup(name string,cacheBytes int64,getter Getter) *Group {
	if getter == nil {
		panic("nil Getter");
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: 	name,
		getter: getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
// 用来获取特定名称的group，使用了只读锁，不涉及写操作
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

/*
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

//get value for a key from cache
//实现了流程1、3
//从 mainCache 中查找缓存，如果存在则返回缓存值。
func(g *Group) Get(key string)(ByteView,error) {
	if key == "" {
		return ByteView{},fmt.Errorf("key is required")
	}
	if v,ok := g.mainCache.get(key);ok{
		log.Println("[GoCache] hit")
		return v,nil
	}
	//缓存不存在，则调用 load 方法
	return g.load(key)
}

//流程2 RegisterPeers registers a PeerPicker for choosing remote peer
//实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中。
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

//load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取）
//修改 load 方法，使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。若是本机节点或失败，则回退到 getLocally()。
func (g* Group) load(key string) (value ByteView,err error) {
	// return g.getLocally(key)
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key);ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[GoCache] Failed to get from peer", err)
		}
	}
	return g.getLocally(key)
}

//使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

//getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
func (g* Group) getLocally(key string)(ByteView,error) {
	bytes,err := g.getter.Get(key)
	if(err != nil) {
		return ByteView{},err
	}
	value := ByteView{b:cloneBytes(bytes)}
	g.populateCache(key,value)
	return value,nil
}

func (g* Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key,value)
}
