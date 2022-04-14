package hotrank

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
	"strconv"
)

// 作品的一级操作 不计入热度 仅记录操作

// 作品redis前缀
var workRedisPrefix = map[string]string{
	"like":    "s:like:",
	"comment": "s:comment:",
}

// GenLikeRedisKey 生成动态点赞key
func GenLikeRedisKey(mid string) string {
	return workRedisPrefix["like"] + mid
}

// GenCommentRedisKey 评论计数key
func GenCommentRedisKey(mid string) string {
	return workRedisPrefix["comment"] + mid
}

// RunCommentRedisChange 评论计数变更
func (c *WorkCall) RunCommentRedisChange(ctx context.Context, mid string, add bool, count int64) error {
	rKey := GenCommentRedisKey(mid)

	var err error
	if add {
		err = c.Rdb.IncrBy(ctx, rKey, count).Err()
	} else {
		err = c.Rdb.DecrBy(ctx, rKey, count).Err()
	}
	if err != nil {
		log.Printf("[comment] 新增评论计数失败 错误:%v", err)
		return err
	}
	return nil

}

// GetCommentCount 获取评论计数
func (c *WorkCall) GetCommentCount(ctx context.Context, mid string) uint64 {
	rKey := GenCommentRedisKey(mid)
	val, err := c.Rdb.Get(ctx, rKey).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("[redis][comment] 获取评论失败 错误:%v", err)
		}
		return 0
	}
	if len(val) > 0 {
		count, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			log.Printf("[redis][comment] %s获取评论数量失败 错误:%v", mid, err)
		}
		return count
	}
	return 0
}

// ChangeLike 点赞变更
// 考虑是否变更为bitmap方式
func (c *WorkCall) ChangeLike(ctx context.Context, mid string, userId string, add bool) error {
	rKey := GenLikeRedisKey(mid)
	has := c.Rdb.SIsMember(ctx, rKey, userId).Val()

	if add {
		if has {
			return LikeExists
		}
		err := c.Rdb.SAdd(ctx, rKey, userId).Err()
		if err != nil {
			return err
		}
	} else {
		if has {
			err := c.Rdb.SRem(ctx, rKey, userId).Err()
			if err != nil {
				return err
			}
		} else {
			return LikeNotExists
		}
	}
	return nil

}

// HasLike 获取是否点赞
func (c *WorkCall) HasLike(ctx context.Context, mid string, userId string) bool {
	rKey := GenLikeRedisKey(mid)
	has := c.Rdb.SIsMember(ctx, rKey, userId).Val()
	return has
}
