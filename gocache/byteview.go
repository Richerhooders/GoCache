package gocache

import (
	"time"
)
//A ByteView holds on immutable view of bytes
//封装一个只读的数据结构，防止修改
type ByteView struct {
	// if b is non-nil,b is used,else s is used
	b []byte
	e time.Time
}

func (v ByteView) Expire() time.Time {
	return v.e
}

//Len return the view’s length
func(v ByteView) Len() int {
	return len(v.b)
}

//ByteSlice return a copy of the data as a byte slice
func(v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

//String returns the data as a string, making a copy if necessary
func(v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte,len(b))
	copy(c,b)
	return c
}