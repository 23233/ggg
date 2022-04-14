package hotrank

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type rangeRankIndex struct {
	Start int64
	End   int64
}

// GetTimeRandomData 获取某个时间随机内容
func (c *WorkCall) GetTimeRandomData(ctx context.Context, count int64, t time.Time) ([]string, error) {
	var result = make([]string, 0, count)
	allCount := c.GetCountOfTimeRange(ctx, []time.Time{t})
	if allCount < 1 {
		return result, EmptyError
	}
	pipeline := c.Rdb.Pipeline()
	rk := c.genTimeKey(t)
	if allCount >= count {
		sj := randomInt(0, allCount-count)
		_, _ = pipeline.ZRange(ctx, rk, sj, sj+count-1).Result()
	} else {
		_, _ = pipeline.ZRange(ctx, rk, 0, -1).Result()
	}
	exec, err := pipeline.Exec(ctx)
	if err != nil {
		return result, err
	}
	for _, cmder := range exec {
		switch cmder.(type) {
		case *redis.StringSliceCmd:
			r := cmder.(*redis.StringSliceCmd).Val()
			result = append(result, r...)
			break
		}
	}

	return result, nil
}

// GetCityRandomData 获取指定数量的随机城市排行榜数据
func (c *WorkCall) GetCityRandomData(ctx context.Context, count int64, cityCode string) ([]string, error) {
	return c.getRandomData(ctx, count, cityCode)
}

// GetAllRandomData 总排行榜获取指定数量随机内容
func (c *WorkCall) GetAllRandomData(ctx context.Context, count int64) ([]string, error) {
	return c.getRandomData(ctx, count, "")
}

// GetAllTopRank 获取总榜的top k
func (c *WorkCall) GetAllTopRank(ctx context.Context, max int64) ([]string, error) {
	// 获取总区间的分布
	distribute := c.GetCountDistribute(ctx)
	r := c.allRange

	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}

	scopeRange := map[string]int64{}

	surplus := max
	for _, h := range r {
		if surplus < 1 {
			break
		}
		k := fmt.Sprintf("%d-%d", h.Min, h.Max)
		if v, ok := distribute[k]; ok {
			if v > 0 {
				rKey := c.genAllRangeKey(h)
				if v >= surplus {
					scopeRange[rKey] = surplus
					break
				}
				surplus -= v
				scopeRange[rKey] = v
			}

		}
	}

	pipeline := c.Rdb.Pipeline()
	for k, v := range scopeRange {
		_, _ = pipeline.ZRevRange(ctx, k, 0, v).Result()
	}
	exec, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, err
	}
	var result = make([]string, 0, max)
	for _, cmder := range exec {
		switch cmder.(type) {
		case *redis.StringSliceCmd:
			r := cmder.(*redis.StringSliceCmd).Val()
			result = append(result, r...)
			break
		}
	}
	return result, nil
}

// GetTodayTopRank 获取今日top k
func (c *WorkCall) GetTodayTopRank(ctx context.Context, max int64) ([]string, error) {
	return c.GetDayTopRank(ctx, time.Now(), max)
}

// GetDayTopRank 获取指定天数的top k
func (c *WorkCall) GetDayTopRank(ctx context.Context, t time.Time, max int64) ([]string, error) {
	if max <= 1 {
		return nil, errors.New("截止位数参数错误")
	}
	rKey := c.genTimeKey(t)
	v, err := c.Rdb.ZRevRange(ctx, rKey, 0, max).Result()
	return v, err
}

// 获取随机数据
func (c *WorkCall) getRandomData(ctx context.Context, count int64, cityCode string) ([]string, error) {
	city := len(cityCode) > 0
	var result = make([]string, 0, count)
	var distribute map[string]int64
	if city {
		distribute = c.GetCityCountDistribute(ctx, cityCode)
	} else {
		distribute = c.GetCountDistribute(ctx)
	}
	// 过滤出有值的区间
	for k, v := range distribute {
		if v < 1 {
			delete(distribute, k)
		}
	}

	var allCount int64
	for _, v := range distribute {
		allCount += v
	}
	if allCount < 1 {
		return result, EmptyError
	}

	sortList := sortMapByValue(distribute)

	// 那个区间取的排行榜排名范围
	var d = make(map[string]rangeRankIndex)
	if allCount >= count {
		// 平均每个区间应该取出来的作品数量
		yz := math.Ceil(float64(count / int64(len(sortList))))
		var yzi = int64(yz)
		// 需要补足的数量
		var supplement int64

		for _, item := range sortList {
			k := item.Key
			v := item.Value

			full := yzi + supplement
			// 如果当前区间的内容大于应该取出的数量还包含补足则直接取出
			if v >= full {
				// 如果只是刚好 没得挑咯
				if v == full {
					d[k] = rangeRankIndex{Start: 0, End: v - 1}
				} else {
					// 有的挑
					sj := randomInt(0, v-full)
					d[k] = rangeRankIndex{Start: sj, End: sj + full - 1}
				}
				supplement = 0
			} else if v >= yzi {
				// 如果当前区间仅仅只能满足应该取出的数量则取出
				d[k] = rangeRankIndex{Start: 0, End: yzi - 1}
			} else {
				// 如果当前区间小于取出因子
				d[k] = rangeRankIndex{Start: 0, End: yzi - v - 1}
				supplement += yzi - v
			}

		}
	} else {
		for k, v := range distribute {
			d[k] = rangeRankIndex{Start: 0, End: v - 1}
		}
	}

	pipeline := c.Rdb.Pipeline()
	for k, v := range d {
		var rk string
		r := rangeStrToRange(k)
		if city {
			rk = c.genKeyOfCityRange(r, cityCode)
		} else {
			rk = c.genAllRangeKey(r)
		}
		_, _ = pipeline.ZRange(ctx, rk, v.Start, v.End).Result()
	}
	exec, err := pipeline.Exec(ctx)
	if err != nil {
		return result, err
	}
	for _, cmder := range exec {
		switch cmder.(type) {
		case *redis.StringSliceCmd:
			r := cmder.(*redis.StringSliceCmd).Val()
			result = append(result, r...)
			break
		}
	}

	return result, nil

}

// e.g: 201-400
func rangeStrToRange(s string) hotRange {
	r := strings.Split(s, "-")
	min, _ := strconv.Atoi(r[0])
	max, _ := strconv.Atoi(r[1])
	return hotRange{
		Min: int64(min),
		Max: int64(max),
	}
}

func randomInt(start, end int64) int64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return int64(r.Intn(int(end-start))) + start
}
