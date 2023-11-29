package pipe

import (
	"context"
	"github.com/redis/rueidis/rueidiscompat"
	"time"
)

// 这里面放置redis中最常见的操作

// AcquireLock 加锁 需要设置key和value 以及锁的时间
func AcquireLock(rdb rueidiscompat.Cmdable, key string, value string, expiration time.Duration) bool {
	resp, err := rdb.SetNX(context.Background(), key, value, expiration).Result()
	if err != nil {
		return false
	}
	return resp
}

// ReleaseLock 释放锁 必须传入正确的key和value才能释放 使用lua脚本
func ReleaseLock(client rueidiscompat.Cmdable, key string, value string) error {
	script := `
    if redis.call("get", KEYS[1]) == ARGV[1] then
        return redis.call("del", KEYS[1])
    else
        return 0
    end
    `

	_, err := client.Eval(context.Background(), script, []string{key}, value).Result()
	return err
}
