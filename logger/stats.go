package logger

import (
	"go.uber.org/atomic"
	"sync"
	"time"
)

// 当前版本仅实现每小时错误日志打印数量统计
// 也仅实现数量统计 请勿实现业务逻辑统计
// 可以自定义拓展 但是尽量不要保存太多东西 占内存

type statsItem struct {
	// 总数量
	Count *atomic.Uint64 `json:"count"`
	// 拓展字段放这里
}

// 统计
type stats struct {
	Items            map[string]map[string]*statsItem `json:"items"` // {day:{format:value}}
	Format           string                           `json:"format"`
	mutex            *sync.RWMutex
	DayClearInterval uint8 `json:"day_clear_interval"`
}

func (c *stats) GenKey() string {
	return time.Now().Format(c.Format)
}
func (c *stats) GenToday() string {
	return time.Now().Format("2006-01-02")
}
func (c *stats) Now() *statsItem {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	todayKey := c.GenToday()
	if _, ok := c.Items[todayKey]; !ok {
		c.Items[todayKey] = make(map[string]*statsItem, 0)
	}
	k := c.GenKey()
	if _, ok := c.Items[todayKey][k]; !ok {
		c.Items[todayKey][k] = &statsItem{
			Count: atomic.NewUint64(0),
		}
	}

	return c.Items[todayKey][k]
}
func (c *stats) ErrInc() {
	n := c.Now()
	n.Count.Inc()
}

func (c *stats) Write(bs []byte) (n int, err error) {
	c.ErrInc()
	return len(bs), nil
}

func (c *stats) ClearOld() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var oldKeys []string
	now := time.Now()
	for k := range c.Items {
		ktime, _ := time.Parse("2006-01-02", k)
		if now.Sub(ktime).Hours() > float64(time.Duration(c.DayClearInterval)*24) {
			oldKeys = append(oldKeys, k)
		}
	}
	for _, k := range oldKeys {
		delete(c.Items, k)
	}
	time.AfterFunc(c.GetClearTime(), c.ClearOld)
}

func (c *stats) GetClearTime() time.Duration {
	return time.Duration(c.DayClearInterval) * time.Hour * 24
}

func NewStats() *stats {
	c := &stats{
		Items:            make(map[string]map[string]*statsItem, 0),
		mutex:            new(sync.RWMutex),
		Format:           _defaultStatFormat,           // 默认按照小时计数
		DayClearInterval: _defaultStatClearIntervalDay, // 7天清理一次数据
	}
	time.AfterFunc(c.GetClearTime(), c.ClearOld)
	return c
}
