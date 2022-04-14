package hotrank

import (
	"strconv"
	"time"
)

// 操作历史 hash 所以这里是field的key
func (c *WorkCall) genHistoryFieldKey() string {
	var n = c.EventName + ":" + c.FromUserId
	if c.EventName == "view_anonymous" {
		n = c.EventName + ":" + c.ip
	}
	if c.EventName != "like" {
		n += ":" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	return n
}

// genHistoryKey 生成作品热度历史的redis key
func (c *WorkCall) genHistoryKey() string {
	return prefix + "history:" + c.workId
}
func (c *WorkCall) genTodayKey() string {
	return c.genTimeKey(time.Now())
}
func (c *WorkCall) genTimeKey(t time.Time) string {
	return prefix + "day:" + t.Format("2006-01-02")
}
func (c *WorkCall) genCityRangeKey(scope hotRange) string {
	return c.genKeyOfCityRange(scope, c.cityCode)
}
func (c *WorkCall) genKeyOfCityRange(scope hotRange, cityCode string) string {
	return prefix + "city:" + cityCode + ":" + strconv.FormatInt(scope.Min, 10) + "-" + strconv.FormatInt(scope.Max, 10)
}
func (c *WorkCall) genAllRangeKey(scope hotRange) string {
	return prefix + "all:" + strconv.FormatInt(scope.Min, 10) + "-" + strconv.FormatInt(scope.Max, 10)
}

// genRange 生成区间 input为区间宽度 例如500 则会生成 [0-499,500-999]
func genRange(input int64) (int64, []hotRange) {
	r := make([]hotRange, 0, 10)
	dz := input
	for i := 1; i < 10; i++ {
		max := dz * 2
		h := hotRange{
			Min: dz + 1,
			Max: max,
		}
		dz = max
		r = append(r, h)
	}
	return input, r
}
