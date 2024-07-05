package lru

import (
	"container/list"
	"time"
	"sync"
)

// Cache is a LRU cache. It is not safe for concurrnet access
type Cache struct {
	mu	   sync.Mutex 
	capacity  int64                         //允许使用的最大内存
	length    int64                         // 当前已使用的内存
	doublyLinkedList   *list.List                    //指向list.List的指针
	//list.Element是container/list包中的一个结构体，用于表示双向链表中的一个元素
	hashmap    map[string]*list.Element      // 一个字符串到list.Element的映射，，键是字符串，值是双向链表中对应节点的指针
	OnEvicted func(key string, value Lengthable) // optional and executed when an entry is purged.回调函数
	stopCh    chan struct{}

	// Now is the Now() function the cache will use to determine
	// the current time which is used to calculate expired values
	// Defaults to time.Now()
	interval  time.Duration
}


//在链表中仍保存每个值对应的 key x的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射。
type entry struct { //双向链表的数据类型
	key   string
	value Lengthable
	expire time.Time
}

// Value use Len to count how many bytes it 
//为了通用性，我们允许值是实现了 Value 接口的任意类型，该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小。
type Lengthable interface {  //实现了value接口的任意类型
	Len() int
}

// New is the Constructor of Cache
// New 创建指定最大容量的LRU缓存。
// 当maxBytes为0时，代表cache无内存限制，无限存放。
func New(maxBytes int64, onEvicted func(string, Lengthable)) *Cache {
	cache := &Cache{
		capacity:  maxBytes,
		doublyLinkedList:   list.New(),
		hashmap:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
		interval:  time.Minute,
		stopCh:    make(chan struct{}),
	}
	go cache.startCleanupTimer()
	return cache
}


//get look ups a key's value
//如果键对应的链表节点存在，则将对应节点移动到头部，并返回查找到的值。
func (c *Cache) Get(key string) (value Lengthable,ok bool) {
	if ele,ok := c.hashmap[key];ok { //找到后ele是指向list.Element的指针
	
		//结构体Element中有名为Value的字段，类型是interface{},意味着其可以存储任意类型的值 ele.Value 存储了与该元素相关联的数据。
		//使用类型断言，将interface{}类型的ele.Value转换为具体的*entry类型便能，够访问 entry 结构体的 key 和 value 字段。
		kv := ele.Value.(*entry)   
		// If the entry has expired, remove it from the cache
		if !kv.expire.IsZero() && time.Now().After(kv.expire) {
			c.removeElement(ele)
			return nil,false
		}
		c.doublyLinkedList.MoveToFront(ele)     //约定front是头部
		return kv.value,true
	}
	return
}

func (c *Cache) Remove(key string) {
	if ele,ok := c.hashmap[key];ok {
		c.removeElement(ele)
	}
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	//取到末尾节点，从链表中删除
	ele := c.doublyLinkedList.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *Cache) removeElement(ele *list.Element) {
	c.doublyLinkedList.Remove(ele)
	kv := ele.Value.(*entry)
	delete(c.hashmap,kv.key)
	c.length -= int64(len(kv.key)) + int64(kv.value.Len())
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key,kv.value)
	}
}


// Add adds a value to the cache.
func (c *Cache) Add(key string,value Lengthable,expire time.Duration) {
	//如果键存在，则更新对应节点的值，并将该节点移到头部。
	expires := time.Now().Add(expire)
	if expire == 0 {
		expires = time.Time{}// Set zero time for no expiration
	}
	if ele, ok := c.hashmap[key];ok {
		kv := ele.Value.(*entry)
		if c.OnEvicted != nil {
			c.OnEvicted(key,kv.value)
		}
		c.doublyLinkedList.MoveToFront(ele)
		kv.expire = expires
		c.length += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else { //不存在则是新增场景，首先头部添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
		ele := c.doublyLinkedList.PushFront(&entry{key, value, expires})
		c.hashmap[key] = ele
		c.length += int64(len(key)) + int64(value.Len())
	}//更新 c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
	for c.capacity != 0 && c.capacity < c.length {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	if c.hashmap == nil {
		return 0
	}
	return c.doublyLinkedList.Len()
}

// Clear purges all stored items from the cache.
func (c *Cache) Clear() {
	if c.OnEvicted != nil {
		for _, e := range c.hashmap {
			kv := e.Value.(*entry)
			c.OnEvicted(kv.key, kv.value)
		}
	}
	c.doublyLinkedList = nil
	c.hashmap = nil
}

func (c *Cache) startCleanupTimer() {
    ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for e := c.doublyLinkedList.Back(); e != nil; e = e.Prev() {
				kv := e.Value.(*entry)
				if !kv.expire.IsZero() && time.Now().After(kv.expire) {
					c.removeElement(e)
				} else {
					break
				}
			}
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache) Stop() {
	close(c.stopCh)
}