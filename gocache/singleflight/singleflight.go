package singleflight

import "sync"

//call 代表正在进行中，或已经结束的请求。
type call struct {
	wg	sync.WaitGroup   //使用 sync.WaitGroup 锁避免重入。
	val	interface{}
	err	error
}

//Group 是 singleflight 的主数据结构，管理不同 key 的请求(call)。
type Flight struct {
	mu	sync.Mutex   	//protexts m 为了保护m不被并发读写而加上的锁
	m	map[string]*call
}


//Do 方法，接收 2 个参数，第一个参数是 key，第二个参数是一个函数 fn。Do 的作用就是，针对相同的 key，
// 无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
func (g *Flight) Fly(key string, fn func() (interface{}, error)) (interface{}, error) { 
	g.mu.Lock()		    // 修改m需要加锁
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()   // 如果请求正在进行中，则等待
		return c.val, c.err  //执行完成直接返回
	}
	c := new(call)
	c.wg.Add(1)		 // 发起请求前加锁
	g.m[key] = c	 // 添加到 g.m，表明 key 已经有对应的请求在处理
	g.mu.Unlock()	 // 修改m结束，解锁

	c.val, c.err = fn()  // 调用 fn，发起请求
	c.wg.Done()			 // 请求结束

	g.mu.Lock()			// 修该m,mu加锁
	delete(g.m, key)	//更新 g.m
	g.mu.Unlock()		// 解锁

	return c.val, c.err
}

/*
为了便于理解Do函数，将g.mu暂且去掉，并把g.m延迟初始化的部分去掉，延迟初始化的目的很简单，提高内存的使用率
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	if c, ok := g.m[key]; ok {
		c.wg.Wait()   // 如果请求正在进行中，则等待
		return c.val, c.err  // 请求结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)       // 发起请求前加锁
	g.m[key] = c      // 添加到 g.m，表明 key 已经有对应的请求在处理

	c.val, c.err = fn() // 调用 fn，发起请求
	c.wg.Done()         // 请求结束

    delete(g.m, key)    // 更新 g.m
    
	return c.val, c.err // 返回结果
}
sync.WaitGroup 是一个非常有用的同步原语，用于等待一组并发操作完成。sync.WaitGroup 提供了一种等待多个 goroutine 完成的简单方法，
它的使用场景包括在主 goroutine 中等待所有启动的子 goroutine 执行完毕再继续执行。并发协程之间不需要消息传递，非常适合sync.WaitGroup
wg.Add(1)
wg.Wait()阻塞，直到锁释放
wg.Done()锁减1
*/