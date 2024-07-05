package gocache

import (
	"gocache/lru"
	"sync"
	"time"
)

const defaultExpiration = 1 * time.Minute

type cache struct {
	mu       sync.Mutex
	lru      *lru.Cache
	capacity int64
}

func newCache(capacity int64) *cache {
    return &cache{capacity: capacity}
}

// 在 add 方法中，判断了 c.lru 是否为 nil，如果等于 nil 再创建实例。这种方法称之为延迟初始化(Lazy Initialization)，
// 一个对象的延迟初始化意味着该对象的创建将会延迟至第一次使用该对象时。主要用于提高性能，并减少程序内存要求。
func (c *cache) add(key string, value ByteView,expiration ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.capacity, nil)
	}
	var exp time.Duration
	if len(expiration) > 0 {
        exp = expiration[0]
    } else {
        exp = defaultExpiration
    }
	c.lru.Add(key, value, exp)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return ByteView{}, false
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return ByteView{}, false
}
