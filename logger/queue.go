package logger

import (
	"encoding/json"
	"reflect"
	"sync"
)

// itemBase now only stores the raw bytes.
type itemBase struct {
	Raw []byte
}

// circularFifoQueue is a high-performance circular buffer (ring buffer).
type circularFifoQueue struct {
	items []itemBase
	head  int
	tail  int
	count int
	max   uint
	mutex *sync.RWMutex
}

// NewCircularFifoQueue creates a new circularFifoQueue.
func NewCircularFifoQueue(maxSize uint) *circularFifoQueue {
	if maxSize == 0 {
		maxSize = 100 // Default size if 0 is provided
	}
	return &circularFifoQueue{
		max:   maxSize,
		mutex: new(sync.RWMutex),
		items: make([]itemBase, maxSize), // Pre-allocate the fixed-size slice
		head:  0,
		tail:  0,
		count: 0,
	}
}

// Add adds an item to the queue. This is an O(1) operation.
func (c *circularFifoQueue) Add(bs []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[c.tail] = itemBase{Raw: bs}
	c.tail = (c.tail + 1) % int(c.max)

	if c.count == int(c.max) {
		c.head = (c.head + 1) % int(c.max)
	} else {
		c.count++
	}
}

// Write implements io.Writer, allowing the queue to be used as a log output.
func (c *circularFifoQueue) Write(bs []byte) (n int, err error) {
	// The logger may reuse the byte slice, so we must copy it to prevent data races.
	newBs := make([]byte, len(bs))
	copy(newBs, bs)
	c.Add(newBs)
	return len(bs), nil
}

// Size returns the current number of items in the queue.
func (c *circularFifoQueue) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.count
}

// Items returns a copy of all items in the queue.
func (c *circularFifoQueue) Items() []itemBase {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.count == 0 {
		return nil
	}

	out := make([]itemBase, c.count)
	if c.head < c.tail {
		copy(out, c.items[c.head:c.tail])
	} else {
		n := copy(out, c.items[c.head:])
		copy(out[n:], c.items[:c.tail])
	}
	return out
}

// ItemsMap decodes all items in the queue into a slice of maps.
// WARNING: This is a computationally expensive operation. Do not use in performance-critical paths.
func (c *circularFifoQueue) ItemsMap() []map[string]interface{} {
	items := c.Items()
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		map2 := make(map[string]interface{})
		err := json.Unmarshal(item.Raw, &map2)
		if err == nil {
			result = append(result, map2)
		}
	}
	return result
}

// ItemsStr returns a slice of the string representation of all items.
func (c *circularFifoQueue) ItemsStr() []string {
	items := c.Items()
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, string(item.Raw))
	}
	return result
}

// ItemsStruct decodes all items into a slice of the provided struct type.
// WARNING: This is a computationally expensive operation. Do not use in performance-critical paths.
func (c *circularFifoQueue) ItemsStruct(base interface{}) []interface{} {
	items := c.Items()
	result := make([]interface{}, 0, len(items))
	t := reflect.TypeOf(base)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for _, item := range items {
		face := reflect.New(t).Interface()
		_ = json.Unmarshal(item.Raw, &face)
		result = append(result, face)
	}
	return result
}
