package hotrank

import (
	"github.com/go-redis/redis/v8"
)

// HotCallBase 热度计算单条信息基础
type HotCallBase struct {
	EventName  string // 事件名
	EventPrice int64  // 操作事件的价格 对应加分数
	FromUserId string // 操作者 新增做功
	ToUserId   string // 接收者 接收做功
	Rdb        *redis.Client
}

var prefix = "h:"
