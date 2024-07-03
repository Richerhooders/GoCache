package gocache

import (
	"testing"
	"time"
)

func TestByteView_Len(t *testing.T) {
	v := ByteView{b: []byte("hello")}
	expectedLen := 5
	if v.Len() != expectedLen {
		t.Errorf("ByteView.Len() = %d, want %d", v.Len(), expectedLen)
	}
}

func TestByteView_ByteSlice(t *testing.T) {
	v := ByteView{b: []byte("hello")}
	result := v.ByteSlice()
	if string(result) != "hello" {
		t.Errorf("ByteView.ByteSlice() = %v, want %v", result, "hello")
	}
	// Check immutability by modifying result
	result[0] = 'x'
	if string(v.b) == "xello" {
		t.Errorf("ByteView.ByteSlice() did not return a copy of the data")
	}
}

func TestByteView_String(t *testing.T) {
	v := ByteView{b: []byte("hello")}
	if v.String() != "hello" {
		t.Errorf("ByteView.String() = %s, want %s", v.String(), "hello")
	}
}

func TestByteView_Expire(t *testing.T) {
	now := time.Now()
	v := ByteView{e: now}
	if !v.Expire().Equal(now) {
		t.Errorf("ByteView.Expire() = %v, want %v", v.Expire(), now)
	}
}