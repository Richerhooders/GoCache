package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrnet access
type Cache struct {
	maxBytes  int64                         //允许使用的最大内存
	nbytes    int64                         // 当前已使用的内存
	ll        *list.List                    //指向list.List的指针
	//list.Element是container/list包中的一个结构体，用于表示双向链表中的一个元素
	cache     map[string]*list.Element      // 一个字符串到list.Element的映射，，键是字符串，值是双向链表中对应节点的指针
	OnEvicted func(key string, value Value) // optional and executed when an entry is purged.回调函数
}

//在链表中仍保存每个值对应的 key 的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射。
type entry struct { //双向链表的数据类型
	key   string
	value Value
}

// Value use Len to count how many bytes it 
//为了通用性，我们允许值是实现了 Value 接口的任意类型，该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小。
type Value interface {  //实现了value接口的任意类型
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}


//get look ups a key's value
//如果键对应的链表节点存在，则将对应节点移动到头部，并返回查找到的值。
func (c *Cache) Get(key string) (value Value,ok bool) {
	if ele,ok := c.cache[key];ok { //找到后ele是指向list.Element的指针
		c.ll.MoveToFront(ele)     //约定front是头部
		//结构体Element中有名为Value的字段，类型是interface{},意味着其可以存储任意类型的值 ele.Value 存储了与该元素相关联的数据。
		//使用类型断言，将interface{}类型的ele.Value转换为具体的*entry类型便能，够访问 entry 结构体的 key 和 value 字段。
		kv := ele.Value.(*entry)   
		return kv.value,true
	}
	return
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	//取到末尾节点，从链表中删除
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}


// Add adds a value to the cache.
func (c *Cache) Add(key string,value Value) {
	//如果键存在，则更新对应节点的值，并将该节点移到头部。
	if ele, ok := c.cache[key];ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else { //不存在则是新增场景，首先头部添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}//更新 c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}