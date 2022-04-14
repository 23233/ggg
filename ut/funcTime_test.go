package ut

import (
	"testing"
	"time"
)

func TestNewFST(t *testing.T) {
	s := NewFST("时间测试")
	time.Sleep(100 * time.Millisecond)
	s.Add("测试")
	time.Sleep(1 * time.Second)
	s.Add("日天")
	s.Print()
}
