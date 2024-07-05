package lru

import (
	"fmt"
	"testing"
	"time"
	"sync"
)

// String 类型实现 Lengthable 接口
type String string

func (s String) Len() int {
	return len(s)
}

func TestCache_AddAndGet(t *testing.T) {
	lru := New(int64(1024), nil)
	key := "testKey"
	value := String("hello world")
	lru.Add(key, value, 0)

	retrievedValue, ok := lru.Get(key)
	if !ok || retrievedValue != value {
		t.Fatalf("Expected value %s for key %s, got %v", value, key, retrievedValue)
	}
}

func TestCache_Expiration(t *testing.T) {
	lru := New(int64(1024), nil)
	key := "testKey"
	value := String("hello world")
	expiration := 5 * time.Second
	lru.Add(key, value, expiration)

	if v, ok := lru.Get(key); !ok || v != value {
		t.Fatalf("before expiration: expected value %s to be available", value)
	}

	time.Sleep(expiration + 1*time.Second)

	if _, ok := lru.Get(key); ok {
		t.Fatalf("after expiration: expected %s to be expired", key)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	lru := New(int64(2048), nil)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			value := String(fmt.Sprintf("value%d", i))

			lru.Add(key, value, 0)
			if v, ok := lru.Get(key); !ok || v != value {
				t.Errorf("Failed to get value for key: %s", key)
			}
		}(i)
	}

	wg.Wait()
}

func TestCache_RemoveOldest(t *testing.T) {
	lru := New(int64(10), nil)
	lru.Add("k1", String("v1"), 0)
	lru.Add("k2", String("v2"), 0)
	lru.Add("k3", String("v3"), 0)

	if _, ok := lru.Get("k1"); ok {
		t.Fatal("Oldest entry k1 was not removed.")
	}
}

func TestCache_OnEvicted(t *testing.T) {
	evictedKeys := make([]string, 0)
	onEvicted := func(key string, value Lengthable) {
		evictedKeys = append(evictedKeys, key)
	}

	lru := New(int64(10), onEvicted)
	lru.Add("k1", String("short"), 0)
	lru.Add("k2", String("longer value"), 0)

	if len(evictedKeys) == 0 {
		t.Fatalf("Expected eviction did not happen.")
	}
}
