package hotrank

import (
	"context"
	"github.com/bluele/gcache"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"log"
)

var workGc gcache.Cache

// 热度区间
type hotRange struct {
	Min int64
	Max int64
}

// WorkCall 作品计算
type WorkCall struct {
	HotCallBase
	workId        string // 作品ID
	cityCode      string
	ip            string
	cityThreshold int64 // 城市提升的阈值 超过就提升到下一个区间
	cityRange     []hotRange
	threshold     int64 // 提升的阈值 超过就提升到下一个区间
	allRange      []hotRange
}

// 实际操作
func (c *WorkCall) run(ctx context.Context) bool {
	// 根据event name 运行不同的验证器
	runChange := false

	if fc, ok := scoreTrigger[c.EventName]; ok {
		runChange = fc(c)
	}

	// 如果通过了验证
	if runChange {
		// 变更自身热度并写入历史
		c.applySelf(ctx)
		// 写入每日排行榜
		c.changeTodayRank(ctx)
		// 获取自身数值 判断是否超过阈值
		count := c.getSelf(ctx)
		if len(c.cityCode) >= 1 {
			// 更新城市排行榜
			c.changeCityRank(ctx, count)
		}

		// 更新总排行榜
		c.changeAllRank(ctx, count)
	}
	return false
}

// 查看历史是否存在 目前用于点赞判断 仅点赞需要判断
func (c *WorkCall) hotHistoryExists() bool {
	rKey := c.genHistoryKey()
	fieldKey := c.genHistoryFieldKey()
	has, err := c.Rdb.HExists(context.Background(), rKey, fieldKey).Result()
	if err != nil {
		log.Fatalf("获取热度历史存在出错 redis key:%s 字段key:%s 错误:%v", rKey, fieldKey, err)
		return false
	}
	return has
}

// 变更自己
func (c *WorkCall) applySelf(ctx context.Context) {
	rKey := prefix + c.workId
	err := c.Rdb.IncrBy(ctx, rKey, c.EventPrice).Err()
	if err != nil {
		log.Fatalf("[hot]变更自身热度失败 错误:%v", err)
		return
	}

	if fc, ok := scoreAddAfter[c.EventName]; ok {
		fc(c)
	}

	// 新增变更历史
	c.changeHistoryAdd(ctx)
}

// 写入变更历史
func (c *WorkCall) changeHistoryAdd(ctx context.Context) {
	rKey := c.genHistoryKey()
	fieldKey := c.genHistoryFieldKey()

	// 解决重复写入 使用nx方法 不存在再写入
	err := c.Rdb.HSetNX(ctx, rKey, fieldKey, c.EventPrice).Err()
	if err != nil {
		log.Fatalf("[hot]写入变更历史错误 错误:%v", err)

	}
}

// 获取自己
func (c *WorkCall) getSelf(ctx context.Context) int64 {
	rKey := prefix + c.workId
	val, err := c.Rdb.Get(ctx, rKey).Int64()
	if err != nil {
		if err != redis.Nil {
			log.Fatalf("[hot]获取自身热度失败 错误:%v", err)

		}
	}
	return val
}

// 变更今日排行榜
// 今日可能已上榜 所以使用ZIncrBy ZIncrBy在对象不存在时会变成add
func (c *WorkCall) changeTodayRank(ctx context.Context) {
	rKey := c.genTodayKey()
	err := c.Rdb.ZIncrBy(ctx, rKey, float64(c.EventPrice), c.workId).Err()
	if err != nil {
		log.Fatalf("[hot]写入今日变化排行榜出错 key:%s 错误:%v", rKey, err)
	}
}

// 变更排行榜
func (c *WorkCall) changeRank(ctx context.Context, nowCount int64, threshold int64, keyGen func(hotRange) string, ranges []hotRange) {
	if nowCount > threshold {
		// 获取当前应该存在的区间
		_, preRange, scope := c.inRange(nowCount, ranges)
		if scope.Min > 0 {
			// 变更到最新
			rKey := keyGen(scope)
			err := c.Rdb.ZAdd(ctx, rKey, &redis.Z{
				Score:  float64(nowCount),
				Member: c.workId,
			}).Err()
			if err != nil {
				log.Fatalf("[hot]变更城市区间排行榜错误 key:%s 错误:%v", rKey, err)

			}
			// 才提升到这个区间
			if nowCount-c.EventPrice-scope.Min < 15 {
				if preRange.Min > threshold {
					pKey := keyGen(preRange)
					_ = c.Rdb.ZRem(ctx, pKey, c.workId).Err()
				}
			}
			return
		}
		log.Fatalf("[hot]寻找正确区间失败count:%d mid:%s 错误:%v", nowCount, c.workId, errors.New("区间命中失败"))

	}
}

// 变更城市榜
func (c *WorkCall) changeCityRank(ctx context.Context, nowCount int64) {
	c.changeRank(ctx, nowCount, c.cityThreshold, c.genCityRangeKey, c.cityRange)
}

// 变更总榜
func (c *WorkCall) changeAllRank(ctx context.Context, nowCount int64) {
	c.changeRank(ctx, nowCount, c.threshold, c.genAllRangeKey, c.allRange)
}

func (c *WorkCall) inRange(nowCount int64, rangeData []hotRange) (int, hotRange, hotRange) {
	var index int
	var preObj hotRange
	var nowObj hotRange
	for i := 0; i < len(rangeData); i++ {
		h := rangeData[i]
		if nowCount >= h.Min && nowCount <= h.Max {
			index = i
			if i != 0 {
				preObj = rangeData[i-1]
			}
			nowObj = h
			return index, preObj, nowObj
		}
	}

	// 判断当前值是否已经超出了最大的区间范围 如果是则取最后一个
	lastObj := rangeData[len(rangeData)-1]
	if nowCount >= lastObj.Max {
		preObj = rangeData[len(rangeData)-2]
		nowObj = rangeData[len(rangeData)-1]
	}
	return index, preObj, nowObj
}

// 初始化排名
func (c *WorkCall) initRange() {
	if c.threshold < 1 {
		threshold, allHotRange := genRange(500)
		cityThreshold, cityHotRange := genRange(200)
		c.threshold = threshold
		c.allRange = allHotRange
		c.cityThreshold = cityThreshold
		c.cityRange = cityHotRange
	}
}

// AddHotChange 新增作品热度变化
// 分享 评论必传toUserId
// 匿名者访问 必传ip
func (c *WorkCall) AddHotChange(ctx context.Context, eventName, fromUserId, toUserId, cityCode, ip string) {
	c.EventName = eventName
	c.EventPrice = workScore[eventName]
	c.FromUserId = fromUserId
	c.ToUserId = toUserId
	c.cityCode = cityCode
	c.ip = ip
	c.run(ctx)
}

// NewWorkHot 新增一个实例
func NewWorkHot(rdb *redis.Client, mid string) *WorkCall {
	d := new(WorkCall)
	d.Rdb = rdb
	d.workId = mid
	d.initRange()
	return d
}

func init() {
	// 暂时不存在60s之内有1W个req 所以定1W吧
	workGc = gcache.New(10000).
		LRU().
		Build()
}
