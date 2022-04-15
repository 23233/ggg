package logger

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
)

type itemBase struct {
	Raw  []byte
	Text string
}

type circularFifoQueue struct {
	Max   uint       // 最大允许数量
	items []itemBase // 存储的内容
	mutex *sync.RWMutex
}

func (c *circularFifoQueue) Items() []itemBase {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.items
}

func (c *circularFifoQueue) ItemsMap() []map[string]interface{} {
	items := c.Items()
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		map2 := make(map[string]interface{})
		err := json.Unmarshal([]byte(item.Text), &map2)
		if err == nil {
			result = append(result, map2)
		}
	}
	return result
}

func (c *circularFifoQueue) ItemsStr() []string {
	items := c.Items()
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Text)
	}
	return result
}

func (c *circularFifoQueue) ItemsStruct(base interface{}) []interface{} {
	items := c.Items()
	result := make([]interface{}, 0, len(items))
	t := reflect.TypeOf(base)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for _, item := range items {
		face := reflect.New(t).Interface()
		_ = json.Unmarshal([]byte(item.Text), &face)
		result = append(result, face)
	}
	return result
}

// 只要实现了这个接口 就可以等价为 io.Writer
func (c *circularFifoQueue) Write(bs []byte) (n int, err error) {
	c.Add(bs)
	return len(bs), nil
}

func (c *circularFifoQueue) Add(bs []byte) {
	c.mutex.Lock()
	var buf strings.Builder
	buf.Write(bs)
	c.items = append(c.items, itemBase{
		Raw:  bs,
		Text: buf.String(),
	})

	if len(c.items) > int(c.Max) {
		c.items = c.items[1:len(c.items)]
	}
	c.mutex.Unlock()
}

func (c *circularFifoQueue) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

func NewCircularFifoQueue(maxSize uint) *circularFifoQueue {

	return &circularFifoQueue{
		Max:   maxSize,
		mutex: new(sync.RWMutex),
		items: make([]itemBase, 0, maxSize),
	}
}
