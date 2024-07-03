package lru

import (
	"reflect"
	"testing"
	"time"
)

type String string

func (d String) Len() int {
	return len(d)
}

func TestGet(t *testing.T) {
	lru := New(int64(0), nil)
	lru.Add("key1", String("1234"),time.Time{})
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)
	lru := New(int64(cap), nil)
	lru.Add(k1, String(v1),time.Time{})
	lru.Add(k2, String(v2),time.Time{})
	lru.Add(k3, String(v3),time.Time{})

	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := New(int64(10), callback)
	lru.Add("key1", String("123456"),time.Time{})
	lru.Add("k2", String("k2"),time.Time{})
	lru.Add("k3", String("k3"),time.Time{})
	lru.Add("k4", String("k4"),time.Time{})

	expect := []string{"key1", "k2"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}

func TestAdd(t *testing.T) {
	lru := New(int64(0), nil)
	lru.Add("key", String("1"),time.Time{})
	lru.Add("key", String("111"),time.Time{})

	if lru.nbytes != int64(len("key")+len("111")) {
		t.Fatal("expected 6 but got", lru.nbytes)
	}
}

func TestExpire(t *testing.T) {
	var tests = []struct {
		name       string
		key        string
		expectedOk bool
		expire     time.Duration
		wait       time.Duration
	}{
		{"not-expired", "myKey", true, time.Second * 1, time.Duration(0)},
		{"expired", "expiredKey", false, time.Millisecond * 100, time.Millisecond * 150},
	}

	for _, tt := range tests {
		lru := New(int64(0), nil)
		lru.Add("key", String("123456"), time.Now().Add(tt.expire))
		time.Sleep(tt.wait)
		val, ok := lru.Get("key")
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok {
            retrievedVal := val.(String)  // 使用类型断言来获取 String 值
            if string(retrievedVal) != "123456" {
                t.Fatalf("%s expected get to return 123456 but got %v", tt.name, retrievedVal)
            }
        }
    }
}