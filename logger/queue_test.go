package logger

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestNewCircularFifoQueue(t *testing.T) {
	type fileBase struct {
		Level string `json:"level"`
		Msg   string `json:"msg"`
	}

	genMsg := func(msg string) []byte {
		p := &fileBase{
			Level: "debug",
			Msg:   msg,
		}
		bs, _ := json.Marshal(p)
		return bs
	}

	maxSize := 2
	q := NewCircularFifoQueue(uint(maxSize))
	for i := 0; i < maxSize+10; i++ {
		q.Add(genMsg(fmt.Sprintf("第%d条消息", i)))
	}
	t.Log(q.Size())
	t.Log(q.ItemsMap())
	stList := q.ItemsStruct(new(fileBase))
	for _, st := range stList {
		l := st.(*fileBase)
		t.Logf("struct消息 %s", l.Msg)
	}
}
