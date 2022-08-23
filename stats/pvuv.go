package stats

import (
	"context"
	"errors"
	"github.com/23233/ggg/ut"
	"github.com/go-redis/redis/v8"
	"math"
	"strings"
	"time"
)

var (
	TimeRangerParamsError = errors.New("params error : endTime <= startTime ")
)

// HyperStats HyperLog redis的统计
type HyperStats struct {
	Prefix string
	Event  string
	Rdb    *redis.Client
}

func NewStats(event string, rdb *redis.Client) *HyperStats {
	return &HyperStats{
		Prefix: "st",
		Event:  event,
		Rdb:    rdb,
	}
}

func (c *HyperStats) GenerateKey(t time.Time) string {
	var st strings.Builder
	st.WriteString(c.Prefix + ":")
	st.WriteString(c.Event + ":")
	st.WriteString(t.Format("2006-01-02"))
	return st.String()
}

func (c *HyperStats) TodayKey() string {
	return c.GenerateKey(time.Now())
}

// Add 已存在的重复元素不会计数
func (c *HyperStats) Add(ctx context.Context, elements ...string) error {
	return c.AddAny(ctx, c.TodayKey(), elements...)
}

// MustAdd 必然新增
func (c *HyperStats) MustAdd(ctx context.Context, elements ...string) {
	c.MustAddAny(ctx, c.TodayKey(), elements...)
	return
}

// AddAny 任何key赋值
func (c *HyperStats) AddAny(ctx context.Context, key string, elements ...string) error {
	return c.Rdb.PFAdd(ctx, key, elements).Err()
}

// MustAddAny 必然新增任何any
func (c *HyperStats) MustAddAny(ctx context.Context, key string, elements ...string) {
	_ = c.AddAny(ctx, key, elements...)
	return
}

// Del 删除key
func (c *HyperStats) Del(ctx context.Context, keys ...string) error {
	return c.Rdb.Del(ctx, keys...).Err()
}

// NowCount 统计当前
func (c *HyperStats) NowCount(ctx context.Context) (int64, error) {
	return c.SummaryKeys(ctx, c.TodayKey())
}

// Merges 合并多个keys
func (c *HyperStats) Merges(ctx context.Context, keys ...string) (saveKey string, err error) {
	var st strings.Builder
	for _, key := range keys {
		st.WriteString(key)
	}
	saveKey = c.Prefix + ":merge:" + ut.StrToB58(st.String())
	err = c.Rdb.PFMerge(ctx, saveKey, keys...).Err()
	return saveKey, err
}

// Counts 合并多个key 一定要注意 是合并 如果A存在于 key1 和 key2中 只会计数1次
func (c *HyperStats) Counts(ctx context.Context, hold bool, keys ...string) (string, int64, error) {
	saveKey, err := c.Merges(ctx, keys...)
	if err != nil {
		return saveKey, 0, err
	}
	defer func() {
		if !hold {
			_ = c.Rdb.Del(context.TODO(), saveKey).Err()
		}
	}()

	allCount, err := c.SummaryKeys(ctx, saveKey)
	return saveKey, allCount, err
}

// TimeRangerCount 时间范围统计 包含开始和结束当天
func (c *HyperStats) TimeRangerCount(ctx context.Context, start time.Time, end time.Time) (int64, error) {
	diff := end.Sub(start)
	days := int(math.Ceil(diff.Hours() / 24))
	if days < 1 {
		return 0, TimeRangerParamsError
	}
	var keys = make([]string, 0, days)
	for i := 0; i < days; i++ {
		keys = append(keys, c.GenerateKey(start.AddDate(0, 0, i)))
	}
	// 获取总数量
	allCount, err := c.SummaryKeys(ctx, keys...)
	return allCount, err
}

// SummaryKeys 汇总统计多个KEY 对每个key进行计数 不是合并
func (c *HyperStats) SummaryKeys(ctx context.Context, keys ...string) (int64, error) {
	var allCount int64
	pipeline := c.Rdb.Pipeline()
	for _, k := range keys {
		pipeline.PFCount(ctx, k).Val()
	}
	exec, err := pipeline.Exec(ctx)
	if err != nil {
		return 0, err
	}
	for _, cmder := range exec {
		switch cmder.(type) {
		case *redis.IntCmd:
			v := cmder.(*redis.IntCmd).Val()
			allCount += v
			break
		}
	}
	return allCount, nil
}

// GetNowWeekCount 获取本周汇总 周一到今天
func (c *HyperStats) GetNowWeekCount(ctx context.Context) (int64, error) {
	return c.TimeRangerCount(ctx, ut.GetFirstDateOfWeek(), time.Now())
}

// GetNowMonthCount 获取本月汇总 从1号到今天
func (c *HyperStats) GetNowMonthCount(ctx context.Context) (int64, error) {
	return c.TimeRangerCount(ctx, ut.GetFirstDateOfMonth(), time.Now())
}
