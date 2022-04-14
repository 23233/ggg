package hotrank

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
	"strings"
	"time"
)

// 特别注意: 排名都是从0开始 0就代表了最高排名 -1代表没有内容

// 获取redis keys列表的总数量和 ZCard
func (c *WorkCall) getKeysOfCount(ctx context.Context, keys []string) int64 {
	var allCount int64
	pipeline := c.Rdb.Pipeline()
	for _, k := range keys {
		pipeline.ZCard(ctx, k).Val()
	}
	exec, _ := pipeline.Exec(ctx)
	for _, cmder := range exec {
		switch cmder.(type) {
		case *redis.IntCmd:
			v := cmder.(*redis.IntCmd).Val()
			allCount += v
			break
		}
	}
	return allCount
}

// 获取redis keys列表的分布 ZCard
func (c *WorkCall) getKeysOfMap(ctx context.Context, keys []string) map[string]int64 {
	var d = make(map[string]int64)
	pipeline := c.Rdb.Pipeline()
	for _, k := range keys {
		pipeline.ZCard(ctx, k).Val()
	}
	exec, _ := pipeline.Exec(ctx)
	for _, cmder := range exec {
		switch cmder.(type) {
		case *redis.IntCmd:
			r := cmder.(*redis.IntCmd)
			kList := strings.Split(r.Args()[1].(string), ":")
			k := kList[len(kList)-1]
			v := r.Val()
			d[k] = v
			break
		}
	}
	return d
}

// GetCountOfTimeRange 获取某个时间段排行榜数量
func (c *WorkCall) GetCountOfTimeRange(ctx context.Context, timer []time.Time) int64 {
	// 获取总榜数量
	allKeys := make([]string, 0, len(timer))
	for _, t := range timer {
		allKeys = append(allKeys, c.genTimeKey(t))
	}
	return c.getKeysOfCount(ctx, allKeys)
}

// GetCountOfRangeInCity 获取某个区间的城市排行榜数量
func (c *WorkCall) GetCountOfRangeInCity(ctx context.Context, cityCode string, Range []hotRange) int64 {
	allKeys := make([]string, 0, len(Range))
	for _, h := range Range {
		allKeys = append(allKeys, c.genKeyOfCityRange(h, cityCode))
	}
	return c.getKeysOfCount(ctx, allKeys)
}

// GetCountOfRangeInAll 获取某个区间的总排行榜数量
func (c *WorkCall) GetCountOfRangeInAll(ctx context.Context, Range []hotRange) int64 {
	// 获取总榜数量
	allKeys := make([]string, 0, len(Range))
	for _, h := range Range {
		allKeys = append(allKeys, c.genAllRangeKey(h))
	}
	return c.getKeysOfCount(ctx, allKeys)
}

// GetCityCountDistribute 获取某个城市区间数量分布情况
func (c *WorkCall) GetCityCountDistribute(ctx context.Context, cityCode string) map[string]int64 {
	allKeys := make([]string, 0, len(c.cityRange))
	for _, h := range c.cityRange {
		allKeys = append(allKeys, c.genKeyOfCityRange(h, cityCode))
	}
	return c.getKeysOfMap(ctx, allKeys)
}

// GetCountDistribute 获取总区间数量分布情况
func (c *WorkCall) GetCountDistribute(ctx context.Context) map[string]int64 {
	allKeys := make([]string, 0, len(c.allRange))
	for _, h := range c.allRange {
		allKeys = append(allKeys, c.genAllRangeKey(h))
	}
	return c.getKeysOfMap(ctx, allKeys)
}

// GetTimeRank 获取作品某日排名
func (c *WorkCall) GetTimeRank(ctx context.Context, t time.Time) int64 {
	k := c.genTimeKey(t)
	result, err := c.Rdb.ZRevRank(ctx, k, c.workId).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("[hot]获取%s排行榜出错 key:%s 错误:%v", t.Format(time.RFC3339), k, err)
		} else {
			return -1
		}
	}
	return result
}

// GetTimeScore 获取作品某日分数
func (c *WorkCall) GetTimeScore(ctx context.Context, t time.Time) float64 {
	k := c.genTimeKey(t)
	result, err := c.Rdb.ZScore(ctx, k, c.workId).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("[hot]获取%s排行榜分数值出错 key:%s 错误:%v", t.Format(time.RFC3339), k, err)

		}
	}
	return result
}

// GetCityRankOfRange 获取作品在城市某个区间的排名
func (c *WorkCall) GetCityRankOfRange(ctx context.Context, Range hotRange) int64 {
	k := c.genCityRangeKey(Range)
	nowScopeRank, err := c.Rdb.ZRevRank(ctx, k, c.workId).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("[hot]获取城市区间%v排名出错 key:%s 错误:%v", Range, k, err)

		} else {
			return -1
		}
	}

	return nowScopeRank
}

// GetRankOfRange 获取作品在某个区间的排名
func (c *WorkCall) GetRankOfRange(ctx context.Context, Range hotRange) int64 {
	k := c.genAllRangeKey(Range)
	rank, err := c.Rdb.ZRevRank(ctx, k, c.workId).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("[hot]获取区间%v排名出错 key:%s 错误:%v", Range, k, err)
		} else {
			return -1
		}
	}
	return rank
}

// 获取区间排名
func (c *WorkCall) getRank(ctx context.Context, nowCount int64, city bool) float64 {
	var allRank float64 = 0
	// 获取当前数值应该存在的区间
	var ranges []hotRange
	if city {
		ranges = c.cityRange
	} else {
		ranges = c.allRange
	}
	_, _, nowObj := c.inRange(nowCount, ranges)

	// 查询在本区间的排名 若无则是-1 看要不要判断
	var nowScopeRank int64
	if city {
		nowScopeRank = c.GetCityRankOfRange(ctx, nowObj)
	} else {
		nowScopeRank = c.GetRankOfRange(ctx, nowObj)
	}
	if nowScopeRank < 0 {
		return -1
	}
	// 找到大于当前区间的所有区间 生成redis keys
	var nextKeys = make([]string, 0, len(ranges))
	for _, h := range ranges {
		if h.Min > nowObj.Max {
			var k string
			if city {
				k = c.genCityRangeKey(h)
			} else {
				k = c.genAllRangeKey(h)
			}
			nextKeys = append(nextKeys, k)
			continue
		}
	}

	// 批量查询获取总排名
	if len(nextKeys) > 0 {
		allRank += float64(c.getKeysOfCount(ctx, nextKeys))
	}
	allRank += float64(nowScopeRank)
	return allRank
}

// GetCityRank 获取自己的城市总排名
func (c *WorkCall) GetCityRank(ctx context.Context) float64 {
	// 找到自己的区间
	nowCount := c.getSelf(ctx)
	return c.getRank(ctx, nowCount, true)
}

// GetAllRank 获取作品的总排名
func (c *WorkCall) GetAllRank(ctx context.Context) float64 {
	// 找到自己的区间
	nowCount := c.getSelf(ctx)
	return c.getRank(ctx, nowCount, false)
}

// GetAggregationRank 聚合排名 所有排名 0是最高
func (c *WorkCall) GetAggregationRank(ctx context.Context) (today float64, city float64, all float64) {
	t := c.GetTimeRank(ctx, time.Now())
	city = c.GetCityRank(ctx)
	all = c.GetAllRank(ctx)
	today = float64(t)
	return
}
