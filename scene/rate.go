package scene

import (
	"context"
	"github.com/kataras/iris/v12"
	"github.com/kataras/realip"
	"github.com/pkg/errors"
	"github.com/redis/rueidis/rueidiscompat"
	"strconv"
	"time"
)

// RateLimiter 基于redis的滑窗限速带加入黑名单功能
// 默认黑名单时间为 黑名单次数*滑窗时间周期*2
type RateLimiter struct {
	redisClient  rueidiscompat.Cmdable
	window       time.Duration
	maxCount     int
	blacklistKey string
	countKey     string
	rateLimitKey string
}

// NewRateLimiter 初始化
func NewRateLimiter(redisClient rueidiscompat.Cmdable, period time.Duration, maxCount int, interfaceKey string) *RateLimiter {
	return &RateLimiter{
		redisClient:  redisClient,
		window:       period,
		maxCount:     maxCount,
		blacklistKey: "blacklist:" + interfaceKey,
		countKey:     "blacklist_count:" + interfaceKey,
		rateLimitKey: "rate_limit:" + interfaceKey + ":",
	}
}

// Allow 检查请求是否被允许
func (rl *RateLimiter) Allow(ctx context.Context, identifier string) (bool, error) {
	now := time.Now().UnixMilli() // 获取当前时间的毫秒表示
	windowStart := now - rl.window.Milliseconds()

	luaScript := `
        local rateLimitKey = KEYS[1]
        local blacklistKey = KEYS[2]
        local countKey = KEYS[3]
        local maxCount = tonumber(ARGV[1])
        local windowStart = tonumber(ARGV[2])
        local now = tonumber(ARGV[3])

        if redis.call('sismember', blacklistKey, ARGV[4]) == 1 then
            return 0
        end

        redis.call('zremrangebyscore', rateLimitKey, '0', windowStart)
        redis.call('zadd', rateLimitKey, now, now)
        redis.call('expire', rateLimitKey, ARGV[5])

        local count = redis.call('zcount', rateLimitKey, windowStart, now)
        if count > maxCount then
            local blacklistCount = redis.call('incr', countKey)
            local duration = tonumber(ARGV[5]) * blacklistCount * 2
            redis.call('sadd', blacklistKey, ARGV[4])
            redis.call('expire', blacklistKey, duration)
            return 0
        end

        return 1
    `

	keys := []string{
		rl.rateLimitKey + identifier,
		rl.blacklistKey,
		rl.countKey + ":" + identifier,
	}
	argv := []string{
		strconv.Itoa(rl.maxCount),
		strconv.FormatInt(windowStart, 10),
		strconv.FormatInt(now, 10),
		identifier,
		strconv.Itoa(int(rl.window.Seconds())),
	}

	result, err := rl.redisClient.Eval(ctx, luaScript, keys, argv).Result()
	if err != nil {
		return false, err
	}

	return result == int64(1), nil
}

func RateUserIdKeyFunc(contextUserId string) ScenesRateKeyFunc {
	return func(ctx iris.Context) (string, error) {
		userId := ctx.Values().GetString(contextUserId)
		if len(userId) < 1 {
			return "", errors.New("获取用户唯一标识失败")
		}
		return userId, nil
	}
}
func RateIpKeyFunc() ScenesRateKeyFunc {
	return func(ctx iris.Context) (string, error) {
		ip := realip.Get(ctx.Request())
		return ip, nil
	}
}
func RateUserAgentFunc() ScenesRateKeyFunc {
	return func(ctx iris.Context) (string, error) {
		ua := ctx.GetHeader("User-Agent")
		return ua, nil
	}
}
func RateIpAndUaFunc() ScenesRateKeyFunc {
	return func(ctx iris.Context) (string, error) {
		ip := realip.Get(ctx.Request())
		ua := ctx.GetHeader("User-Agent")
		return ip + ua, nil
	}
}
func RateUserIdAndIpAndUaFunc(contextUserId string) ScenesRateKeyFunc {
	return func(ctx iris.Context) (string, error) {
		userId := ctx.Values().GetString(contextUserId)
		if len(userId) < 1 {
			return "", errors.New("获取用户唯一标识失败")
		}
		ip := realip.Get(ctx.Request())
		ua := ctx.GetHeader("User-Agent")
		return userId + ip + ua, nil
	}
}
