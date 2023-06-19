package pipe

import (
	"github.com/go-redis/redis/v8"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"strings"
)

type JwtCheckDep struct {
	FlatMap       map[string]any
	Env           string
	Authorization string // Auth头
}

var (
	JwtCheck = &RunnerContext[*JwtCheckDep, any, rueidis.Client, map[string]any]{
		Name: "jwt交换",
		Key:  "jwt_exchange",
		call: func(ctx iris.Context, origin *JwtCheckDep, params any, db rueidis.Client, more ...any) *RunResp[map[string]any] {
			if origin == nil {
				return newPipeErr[map[string]any](PipeDepError)
			}

			dep := origin
			rdb := db
			helper := NewJwtHelper(rdb)

			pack := dep.FlatMap

			// pack格式判断
			if _, ok := pack["userId"]; !ok {
				return newPipeErr[map[string]any](errors.New("jwt数据包关键参数缺失"))

			}
			if _, ok := pack["env"]; !ok {
				return newPipeErr[map[string]any](errors.New("jwt数据包参数缺失"))

			}
			if _, ok := pack["Raw"]; !ok {
				return newPipeErr[map[string]any](errors.New("jwt数据包结构错误"))
			}

			// jwt 中的数据集
			userId := pack["userId"].(string)
			packEnv := pack["env"].(string)
			isShort := false
			if v, ok := pack["Short"]; ok {
				isShort = v.(bool)
			}
			raw := pack["Raw"].(string)

			// 进行安全校验
			if isShort {
				// 获取最新的token
				resp := rdb.Do(ctx, rdb.B().Get().Key(helper.JwtRedisGenKey(userId, dep.Env)).Build())
				if resp.Error() != nil {
					// 只要不是为空错误 则为其他错误都直接返回
					if resp.Error() != redis.Nil {
						return newPipeErr[map[string]any](resp.Error())
					}
					// 如果是为空 则是错误
					return newPipeErr[map[string]any](errors.New("jwt数据获取失败"))

				}
				st, err := resp.ToString()
				if err != nil {
					return newPipeErr[map[string]any](err)
				}
				raw = strings.TrimPrefix(raw, JwtPrefix)
				if raw != st {
					return newPipeErr[map[string]any](errors.New("jwt数据包已过期"))

				}
				return newPipeResult(pack)
			}

			// 完整判断环境是否正常
			if dep.Env != packEnv {
				return newPipeErr[map[string]any](errors.New("jwt数据包环境校验失败"))
			}
			return newPipeResult[map[string]any](pack)
		},
	}
)
