package gocache

import (
    "testing"
    "time"
)

type String string

func (s String) Len() int {
    return len(s)
}

// 模拟一个简单的获取函数
func mockGetter(key string) ([]byte, error) {
    return []byte("data for " + key), nil
}

// 测试缓存命中
func TestCacheHit(t *testing.T) {
    groupName := "testGroup"
    testKey := "hello"
    testValue := []byte("world")

    getter := GetterFunc(func(key string) ([]byte, error) {
        return testValue, nil
    })

    // 注意增加了过期时间参数
    group := NewGroup(groupName, 64, time.Minute,getter)
    group.Get(testKey)  // 预填充缓存

    // 第二次获取，应从缓存中命中
    value, err := group.Get(testKey)
    if err != nil || string(value.ByteSlice()) != "world" {
        t.Fatalf("Cache hit failed, expected %s got %s", "world", string(value.ByteSlice()))
    }
}

// 测试从数据源加载
func TestLoadFromGetter(t *testing.T) {
    groupName := "testGroup"
    testKey := "hello"
    testValue := []byte("world")

    called := false
    getter := GetterFunc(func(key string) ([]byte, error) {
        called = true
        return testValue, nil
    })

    // 注意增加了过期时间参数
    group := NewGroup(groupName, 64, time.Minute, getter )
    value, err := group.Get(testKey)
    if err != nil || string(value.ByteSlice()) != "world" {
        t.Fatalf("Expected value 'world', got %v", string(value.ByteSlice()))
    }
    if !called {
        t.Fatalf("Expected getter to be called")
    }
}

// 测试过期策略
func TestExpiration(t *testing.T) {
    groupName := "testGroup"
    testKey := "key"
    testValue := []byte("value")

    getter := GetterFunc(func(key string) ([]byte, error) {
        return testValue, nil
    })

    // 设置较短的过期时间进行测试
    group := NewGroup(groupName, 64,  50*time.Millisecond, getter)

    _, err := group.Get(testKey)  // 首次获取，填充缓存
    if err != nil {
        t.Fatalf("Failed to retrieve value: %v", err)
    }

    time.Sleep(100 * time.Millisecond)  // 等待超过过期时间

    if _, ok := group.mainCache.get(testKey); ok {
        t.Fatalf("Expected key %s to be expired", testKey)
    }
}

// 测试 Group 缓存清理过期项功能
func TestGroupCacheExpiration(t *testing.T) {
    // 创建一个 Group，设置较短的过期时间
    group := NewGroup("testGroup", 64*1024,  50*time.Millisecond,GetterFunc(mockGetter))

    key := "testKey"
    _, err := group.Get(key)
    if err != nil {
        t.Fatalf("Error retrieving value from group: %v", err)
    }

    time.Sleep(100 * time.Millisecond)  // 确保有足够的时间过期

    _, err = group.Get(key)
    if err != nil {
        t.Fatalf("Error retrieving value from group after expiration: %v", err)
    }
}
