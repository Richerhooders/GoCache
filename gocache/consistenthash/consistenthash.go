package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//Hash maps bytes to uint32
//定义函数类型Hash，采取依赖注入的方式，允许用于替换成自定义的Hash函数，也方便测试时替换，默认为crc32.ChecksumIEEE算法
type HashFunc func(data []byte) uint32

//Map constains all hashed keys
type Consistency struct {
	hash	HashFunc 			//哈希函数
	replicas	int		    //虚拟节点的倍数
	keys	[]int  // Sorted	//哈希环
	hashMap	map[int]string	//虚拟节点与真实节点的映射表hashMap,键是虚拟节点的哈希值，值是真实节点的名称
}

//Register将各个peer注册到哈希环上
func (c *Consistency) Register(peers ...string) {
	for _,key := range peers {
		for i:= 0;i < c.replicas;i++ {
			hash := int(c.hash([]byte(strconv.Itoa(i) + key)))
			c.keys = append(c.keys, hash)
			c.hashMap[hash] = key
		}
	}
	sort.Ints(c.keys)
}

//New creates a new instance
//构造函数 New() 允许自定义虚拟节点倍数和 Hash 函数。
func New(replicas int,fn HashFunc) *Consistency {
	c := &Consistency {
		replicas: replicas,
		hash:	fn,
		hashMap: make(map[int]string),
	}
	if c.hash == nil {
		c.hash = crc32.ChecksumIEEE
	}
	return c
}

//Add adds some keys to the hash
//Add 函数允许传入 0个 或 多个真实节点的名称。
// func(m *Consistency) Add(keys ...string) {
// 	for _, key := range keys {
// 		//对每一个真实节点 key，对应创建 m.replicas 个虚拟节点
// 		for i:=0;i < m.replicas;i++ { 
// 			//虚拟节点的名称是：strconv.Itoa(i) + key，即通过添加编号的方式区分不同虚拟节点。使用m.hash()计算虚拟节点的哈希值
// 			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
// 			//使用append(m.keys,hash)添加到环上
// 			m.keys = append(m.keys, hash)
// 			//在 hashMap 中增加虚拟节点和真实节点的映射关系。
// 			m.hashMap[hash] = key
// 		}
// 	}
// 	//环上的哈希值排序。
// 	sort.Ints(m.keys)
// }

//Get gets the closest item in the hash to the provided key
func (c *Consistency) GetPeer(key string) string {
	if len(c.keys) == 0 {
		return ""
	}
	//计算key的哈希值
	hash := int(c.hash([]byte(key)))
	//Binary search for appropriate replica.
	//顺时针找到第一个匹配的虚拟节点的下标 idx,从m.key获取对应的哈希值。
	idx := sort.Search(len(c.keys),func(i int) bool {
		return c.keys[i] >= hash
	})
	return c.hashMap[c.keys[idx%len(c.keys)]]
}

// Remove use to remove a key and its virtual keys on the ring and map
func (c *Consistency) Remove(key string) {
	for i := 0; i < c.replicas; i++ {
		hash := int(c.hash([]byte(strconv.Itoa(i) + key)))
		idx := sort.SearchInts(c.keys, hash)
		c.keys = append(c.keys[:idx], c.keys[idx+1:]...)
		delete(c.hashMap, hash)
	}
}