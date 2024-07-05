// Copyright 2021 Peanutzhen. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gocache

import "time"
// byteview 模块定义读取缓存结果
// 实际上 byteview 只是简单的封装了byte slice，让其只读。
// 试想一下，直接返回slice，在golang里，一切参数按值传递。
// slice底层只是一个struct，记录着ptr/len/cap，相当于
// 复制了一份这三者的值。因此[]byte底层指向同一片内存区域
// 我们的缓存底层是存储在LRU的双向链表的Element里，因此
// 可以被恶意修改。因此需要将slice封装成只读的ByteView

type ByteView struct {
	b      []byte
	s      string
	expire time.Time
}


func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// 注意到 ByteView 的方法接收者都是对象 这样是为了不影响调用对象本身

func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回一个拷贝的字节切片, 防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	if v.b != nil {
		return cloneBytes(v.b)
	}
	return []byte(v.s)
}

func (v ByteView) String() string {
	return string(v.b)
}