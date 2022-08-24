package stats

import (
	"context"
	"github.com/23233/ggg/ut"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
)

type WorkStatsResp struct {
	Id      string `json:"id"`
	Share   int    `json:"share"`
	Like    int    `json:"like"`
	Comment int    `json:"comment"`
	Pv      int    `json:"pv"`
	Uv      int    `json:"uv"`
}

// WorkStats 作品统计
type WorkStats struct {
	Prefix     string
	WorkId     string
	Rdb        *redis.Client
	pvStats    *HyperStats
	uvStats    *HyperStats
	shareKey   string
	likeKey    string
	commentKey string
	summaryKey string
}

func NewWorkStats(workId string, rdb *redis.Client) *WorkStats {
	var wks = &WorkStats{
		Prefix:  "wks",
		WorkId:  workId,
		Rdb:     rdb,
		pvStats: NewStatsKey("wd_pv", rdb, workId),
		uvStats: NewStatsKey("wd_uv", rdb, workId),
	}
	wks.shareKey = wks.GenerateKey("share")
	wks.likeKey = wks.GenerateKey("like")
	wks.commentKey = wks.GenerateKey("comment")
	wks.summaryKey = wks.Prefix + ":" + workId
	return wks
}

func (c *WorkStats) GenerateKey(event string) string {
	var st strings.Builder
	st.WriteString(c.Prefix + ":")
	st.WriteString(c.WorkId + ":")
	st.WriteString(event)
	return st.String()
}

// AddPv 新增pv ip+ua 做hash
func (c *WorkStats) AddPv(ctx context.Context, ip string, ua string) error {
	k := ut.StrToB58(ip + ua)
	return c.pvStats.Add(ctx, k)
}

// AddUv 新增uv 观看者userId
func (c *WorkStats) AddUv(ctx context.Context, userId string) error {
	return c.uvStats.Add(ctx, userId)
}

// AddShare 新增转发分享
func (c *WorkStats) AddShare(ctx context.Context, userId string) error {
	return c.Rdb.SAdd(ctx, c.shareKey, userId).Err()
}

// InShare 判断是否在分享列表中
func (c *WorkStats) InShare(ctx context.Context, userId string) (bool, error) {
	rl, err := c.Rdb.SIsMember(ctx, c.shareKey, userId).Result()
	return rl, err
}

// AddLike 新增喜欢
func (c *WorkStats) AddLike(ctx context.Context, userId string) error {
	return c.Rdb.SAdd(ctx, c.likeKey, userId).Err()
}

// InLike 判断是否在喜欢列表中
func (c *WorkStats) InLike(ctx context.Context, userId string) (bool, error) {
	rl, err := c.Rdb.SIsMember(ctx, c.likeKey, userId).Result()
	return rl, err

}

// UnLike 取消喜欢
func (c *WorkStats) UnLike(ctx context.Context, userId string) error {
	return c.Rdb.SRem(ctx, c.likeKey, userId).Err()
}

// AddComment 新增评论 用户是可以多条评论的
func (c *WorkStats) AddComment(ctx context.Context, userId string, mid string) error {
	return c.Rdb.SAdd(ctx, c.commentKey, userId+","+mid).Err()
}

// InComment 判断是否在评论列表中
func (c *WorkStats) InComment(ctx context.Context, userId string, mid string) (bool, error) {
	rl, err := c.Rdb.SIsMember(ctx, c.commentKey, userId+","+mid).Result()
	return rl, err

}

// UnComment 删除评论
func (c *WorkStats) UnComment(ctx context.Context, userId string, mid string) error {
	return c.Rdb.SRem(ctx, c.commentKey, userId+","+mid).Err()
}

// SummarySync 汇总同步保存
func (c *WorkStats) SummarySync(ctx context.Context) error {
	shareCount := c.Rdb.SCard(ctx, c.shareKey).Val()
	likeCount := c.Rdb.SCard(ctx, c.likeKey).Val()
	commentCount := c.Rdb.SCard(ctx, c.commentKey).Val()
	pvCount := c.pvStats.NowCountVal(ctx)
	uvCount := c.uvStats.NowCountVal(ctx)
	return c.Rdb.HMSet(ctx, c.summaryKey, "share", shareCount, "like", likeCount, "comment", commentCount, "pv", pvCount, "uv", uvCount).Err()
}

// GetSummary 获取汇总信息
func (c *WorkStats) GetSummary(ctx context.Context) (*WorkStatsResp, error) {
	m, err := c.Rdb.HGetAll(ctx, c.summaryKey).Result()
	if err != nil {
		return nil, err
	}
	var resp WorkStatsResp
	resp.Id = c.WorkId
	if v, ok := m["share"]; ok {
		i, _ := strconv.Atoi(v)
		resp.Share = i
	}
	if v, ok := m["like"]; ok {
		i, _ := strconv.Atoi(v)
		resp.Like = i
	}
	if v, ok := m["comment"]; ok {
		i, _ := strconv.Atoi(v)
		resp.Comment = i
	}
	if v, ok := m["pv"]; ok {
		i, _ := strconv.Atoi(v)
		resp.Pv = i
	}
	if v, ok := m["uv"]; ok {
		i, _ := strconv.Atoi(v)
		resp.Uv = i
	}

	return &resp, nil
}

// ManyWorkGetSummary 多个作品获取汇总信息
func ManyWorkGetSummary(ctx context.Context, rdb *redis.Client, batchLimit int, wordIds ...string) ([]*WorkStatsResp, error) {
	if len(wordIds) < 1 {
		return nil, errors.New("params is empty")
	}

	var wg sync.WaitGroup

	// 获取每个组的id区间
	var scope = make(map[int][]string, 0)
	// 剩余数量
	var remaining = len(wordIds)
	// 单批大小
	var limit = batchLimit

	// 取得最后余数
	var endBlock = remaining % limit
	// 取得批次数量
	var batchCount = int(math.Floor(float64(remaining / limit)))
	// 遍历批次
	for i := 0; i < batchCount; i++ {
		start := i * limit
		end := (i + 1) * limit
		scope[i] = wordIds[start:end]
	}
	scope[-1] = wordIds[len(wordIds)-endBlock:]

	var result = make([]*WorkStatsResp, 0, len(wordIds))

	// 遍历区间
	for _, v := range scope {
		wg.Add(1)
		remain := len(v)
		itemList := v
		go func() {
			for {
				if remain < 1 {
					break
				}
				item := itemList[remain-1]
				remain -= 1
				stats := NewWorkStats(item, rdb)
				err := stats.SummarySync(ctx)
				if err != nil {
					log.Printf("进行作品%s汇总失败 err:%v", item, err)
					continue
				}
				resp, err := stats.GetSummary(ctx)
				if err != nil {
					log.Printf("获取作品%s汇总信息失败 err:%v", item, err)
					continue
				}
				result = append(result, resp)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// 传入数量和结果数量不一致的话?

	return result, nil
}

// BatchWorkGetSummary 自动按批次并发获取作品汇总信息
func BatchWorkGetSummary(ctx context.Context, rdb *redis.Client, wordIds ...string) ([]*WorkStatsResp, error) {
	return ManyWorkGetSummary(ctx, rdb, 10, wordIds...)
}
