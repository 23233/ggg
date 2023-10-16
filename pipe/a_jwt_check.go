package pipe

import (
	"github.com/go-redis/redis/v8"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"strings"
)

type JwtFlatBase struct {
	UserId    string `json:"user_id,omitempty"`
	Env       string `json:"env,omitempty"`
	Raw       string `json:"raw,omitempty"`
	LoginTime string `json:"login_time,omitempty"`
}

func (c JwtFlatBase) IsShort() bool {
	return strings.HasPrefix(c.Raw, JwtShortPrefix)
}

type JwtCheckDep struct {
	FlatMap       JwtFlatBase
	Env           string
	Authorization string // Auth头
}

var (
	// JwtCheck jwt验证
	// 必传origin JwtCheckDep
	// 必传db 为redis Client
	JwtCheck = &RunnerContext[*JwtCheckDep, any, rueidis.Client, JwtFlatBase]{
		Name: "jwt验证",
		Key:  "jwt_check",
		call: func(ctx iris.Context, origin *JwtCheckDep, params any, db rueidis.Client, more ...any) *RunResp[JwtFlatBase] {
			if origin == nil {
				return newPipeErr[JwtFlatBase](PipeDepError)
			}

			dep := origin
			rdb := db
			helper := NewJwtHelper(rdb)

			pack := dep.FlatMap

			if len(pack.UserId) < 1 {
				return newPipeErr[JwtFlatBase](errors.New("jwt数据包关键参数缺失"))
			}
			if len(pack.Env) < 1 {
				return newPipeErr[JwtFlatBase](errors.New("jwt数据包参数缺失"))
			}
			if len(pack.Raw) < 1 {
				return newPipeErr[JwtFlatBase](errors.New("jwt数据包结构错误"))
			}

			// jwt 中的数据集
			raw := pack.Raw

			// 进行安全校验
			if pack.IsShort() {
				// 获取最新的token
				resp := rdb.Do(ctx, rdb.B().Get().Key(helper.JwtRedisGenKey(pack.UserId, dep.Env)).Build())
				if resp.Error() != nil {
					// 只要不是为空错误 则为其他错误都直接返回
					if resp.Error() != redis.Nil {
						return newPipeErr[JwtFlatBase](resp.Error())
					}
					// 如果是为空 则是错误
					return newPipeErr[JwtFlatBase](errors.New("jwt数据获取失败"))

				}
				st, err := resp.ToString()
				if err != nil {
					return newPipeErr[JwtFlatBase](err)
				}
				raw = strings.TrimPrefix(raw, JwtPrefix)
				if raw != st {
					return newPipeErr[JwtFlatBase](errors.New("jwt数据包已过期"))
				}
				return newPipeResult(pack)
			}

			// 完整判断环境是否正常
			if dep.Env != pack.Env {
				return newPipeErr[JwtFlatBase](errors.New("jwt数据包环境校验失败"))
			}
			return newPipeResult[JwtFlatBase](pack)
		},
	}
)
