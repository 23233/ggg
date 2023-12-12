package scene

import (
	"context"
	"fmt"
	"github.com/23233/ggg/ut"
	"github.com/redis/rueidis"
	"github.com/redis/rueidis/rueidiscompat"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	rdb, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%s", "127.0.0.1", "6379")},
		Password:    ut.GetEnv("redis_password", ""),
		SelectDB:    7,
	})
	assert.Nil(t, err)
	compat := rueidiscompat.NewAdapter(rdb)

	rateLimiter := NewRateLimiter(compat, 1*time.Minute, 10, "test_interface")
	ctx := context.Background()
	identifier := "test_user"

	// 清理测试数据
	compat.Del(ctx, rateLimiter.rateLimitKey+identifier)
	compat.Del(ctx, rateLimiter.blacklistKey)
	compat.Del(ctx, rateLimiter.countKey+":"+identifier)

	// 测试请求是否被正确允许
	for i := 0; i < 10; i++ {
		allowed, err := rateLimiter.Allow(ctx, identifier)
		assert.Nil(t, err)
		assert.True(t, allowed, "请求应该被允许")
	}

	// 测试超过限制的请求是否被拒绝
	allowed, err := rateLimiter.Allow(ctx, identifier)
	assert.Nil(t, err)
	assert.False(t, allowed, "超过限制的请求应该被拒绝")

	// 等待黑名单过期
	time.Sleep(2 * time.Minute)

	// 测试黑名单过期后请求是否被允许
	allowed, err = rateLimiter.Allow(ctx, identifier)
	assert.Nil(t, err)
	assert.True(t, allowed, "黑名单过期后，请求应该再次被允许")
}
