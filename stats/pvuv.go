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
	DefaultPrefix = "st"
)

var (
	TimeRangerParamsError  = errors.New("params error : endTime <= startTime ")
	ParamsLengthEmptyError = errors.New("params error : input params is empty")
	ParamsKeyEmptyError    = errors.New("params error : default key is empty please use newStats or NewStatsKey init struct")
)

// HyperStats HyperLog redis的统计
type HyperStats struct {
	Prefix string
	Event  string
	Rdb    *redis.Client
	nowKey string
}

// NewStats 默认统计方法 以今日时间为维度
func NewStats(event string, rdb *redis.Client) *HyperStats {
	var h HyperStats
	h.Prefix = DefaultPrefix
	h.Event = event
	h.Rdb = rdb
	h.nowKey = h.GenerateKey(time.Now().Format("2006-01-02"))
	return &h
}

// NewStatsKey 可选统计方法 指定key后缀 适用于文章等以ID为主的维度
func NewStatsKey(event string, rdb *redis.Client, key string) *HyperStats {
	var h = NewStats(event, rdb)
	h.ChangeDefaultKey(key)
	return h
}

// GenerateKey 通用生成规则的k [prefix]:[event]:[string]
func (c *HyperStats) GenerateKey(k string) string {
	var st strings.Builder
	st.WriteString(c.Prefix + ":")
	st.WriteString(c.Event + ":")
	st.WriteString(k)
	return st.String()
}

// ChangeDefaultKey 变更默认的key生成方式 仅变更最后的string
func (c *HyperStats) ChangeDefaultKey(k string) {
	c.nowKey = c.GenerateKey(k)
}

// ChangePrefix 变更前缀 请在生成时就变更 最好别变更
func (c *HyperStats) ChangePrefix(newPrefix string) {
	c.Prefix = newPrefix
}

// Add 已存在的重复元素不会计数
func (c *HyperStats) Add(ctx context.Context, elements ...string) error {
	return c.AddAny(ctx, c.nowKey, elements...)
}

// MustAdd 必然新增
func (c *HyperStats) MustAdd(ctx context.Context, elements ...string) {
	c.MustAddAny(ctx, c.nowKey, elements...)
	return
}

// AddAny 任何key赋值
func (c *HyperStats) AddAny(ctx context.Context, key string, elements ...string) error {
	if len(key) < 1 {
		return ParamsKeyEmptyError
	}
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
	return c.SummaryKeys(ctx, c.nowKey)
}

// NowCountVal 统计当前 仅返回数量
func (c *HyperStats) NowCountVal(ctx context.Context) int64 {
	val, _ := c.NowCount(ctx)
	return val
}

// Merges 合并多个keys  一定要注意 是合并 如果A存在于 key1 和 key2中 只会计数1次
func (c *HyperStats) Merges(ctx context.Context, keys ...string) (saveKey string, err error) {
	var st strings.Builder
	for _, key := range keys {
		st.WriteString(key)
	}
	saveKey = c.Prefix + ":merge:" + ut.StrToB58(st.String())
	err = c.Rdb.PFMerge(ctx, saveKey, keys...).Err()
	return saveKey, err
}

// Counts 统计多个keys 先合并 再计数 hold为统计之后是否保留
func (c *HyperStats) Counts(ctx context.Context, hold bool, keys ...string) (saveKey string, allCount int64, err error) {
	saveKey, err = c.Merges(ctx, keys...)
	if err != nil {
		return saveKey, 0, err
	}
	defer func() {
		if !hold {
			_ = c.Rdb.Del(context.TODO(), saveKey).Err()
		}
	}()

	allCount, err = c.SummaryKeys(ctx, saveKey)
	return saveKey, allCount, err
}

// DayTimeRangerCount 时间范围统计 包含开始和结束当天
func (c *HyperStats) DayTimeRangerCount(ctx context.Context, start time.Time, end time.Time) (int64, error) {
	diff := end.Sub(start)
	days := int(math.Ceil(diff.Hours() / 24))
	if days < 1 {
		return 0, TimeRangerParamsError
	}
	var keys = make([]string, 0, days)
	for i := 0; i < days; i++ {
		keys = append(keys, c.GenerateKey(start.AddDate(0, 0, i).Format("2006-01-02")))
	}
	// 获取总数量
	allCount, err := c.SummaryKeys(ctx, keys...)
	return allCount, err
}

// SummaryKeys 汇总统计多个KEY 对每个key进行计数 仅计数不合并不生成额外key
func (c *HyperStats) SummaryKeys(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) < 1 {
		return 0, ParamsLengthEmptyError
	}
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

// SummaryKeysUseRule 汇总统计多个KEY 使用规则生成最终的key eg: [prefix]:[event]:[id]
func (c *HyperStats) SummaryKeysUseRule(ctx context.Context, ids ...string) (int64, error) {
	allKeys := make([]string, 0, len(ids))
	for _, id := range ids {
		allKeys = append(allKeys, c.GenerateKey(id))
	}
	return c.SummaryKeys(ctx, allKeys...)
}

// GetNowWeekCount 获取本周汇总 周一到今天
func (c *HyperStats) GetNowWeekCount(ctx context.Context) (int64, error) {
	return c.DayTimeRangerCount(ctx, ut.GetFirstDateOfWeek(), time.Now())
}

// GetNowMonthCount 获取本月汇总 从1号到今天
func (c *HyperStats) GetNowMonthCount(ctx context.Context) (int64, error) {
	return c.DayTimeRangerCount(ctx, ut.GetFirstDateOfMonth(), time.Now())
}

// GetAnyMonthCount 获取任何月份整月汇总 请输入1-12的月份
func (c *HyperStats) GetAnyMonthCount(ctx context.Context, monthNumber time.Month) (int64, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), monthNumber, 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, -1)
	return c.DayTimeRangerCount(ctx, monthStart, monthEnd)
}
