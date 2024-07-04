
// package gocache
// import (
// 	"testing"
// 	"reflect"
// 	"fmt"
// 	"log"
// )

// //借助GetterFunc的类型转换，建一个匿名回调函数转换成了接口f Getter
// //调用该接口方法f.get(key string),实际上就是调用匿名回调函数
// func TestGetter(t *testing.T) {
// 	var f Getter = GetterFunc(func(key string) ([]byte, error) {
// 		return []byte(key), nil
// 	})

// 	expect := []byte("key")
// 	if v, _ := f.retrieve("key"); !reflect.DeepEqual(v, expect) {
// 		t.Errorf("callback failed")
// 	}
// }

// var db = map[string]string{
// 	"Tom":  "630",
// 	"Jack": "589",
// 	"Sam":  "567",
// }

// func TestGet(t *testing.T) {
// 	loadCounts := make(map[string]int, len(db))
// 	gee := NewGroup("scores", 2<<10, GetterFunc(
// 		func(key string) ([]byte, error) {
// 			log.Println("[SlowDB] search key", key)
// 			if v, ok := db[key]; ok {
// 				if _, ok := loadCounts[key]; !ok {
// 					loadCounts[key] = 0
// 				}
// 				loadCounts[key] += 1
// 				return []byte(v), nil
// 			}
// 			return nil, fmt.Errorf("%s not exist", key)
// 		}))

// 	for k, v := range db {
// 		if view, err := gee.Get(k); err != nil || view.String() != v { //首次加载测试，预期从db加载数据
// 			t.Fatal("failed to get value of Tom") //获取失败或者返回值与预期不符，测试失败
// 		} // load from callback function
// 		//缓存命中测试，此时数据应该从缓存中返回，而不触发数据库的再次加载。loadCounts记录的值应该仍然是1
// 		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
// 			t.Fatalf("cache %s miss", k) // 缓存没有命中或者数据被重复加载，测试失败
// 		} // cache hit
// 	}

// 	if view, err := gee.Get("unknown"); err == nil { // 非存在键的测试
// 		t.Fatalf("the value of unknow should be empty, but %s got", view)
// 	}
// }

package gocache

import (
	"fmt"
	"log"
	"testing"
)

func TestGet(t *testing.T) {
	mysql := map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}
	loadCounts := make(map[string]int, len(mysql))

	g := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[Mysql] search key", key)
			if v, ok := mysql[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range mysql {
		if view, err := g.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of %s", k)
		}
		if _, err := g.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := g.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	} else {
		log.Println(err)
	}
}