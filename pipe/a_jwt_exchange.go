package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"strings"
	"time"
)

var (
	// JwtExchange jwt交换
	// 必传origin JwtCheckDep jwt解析出来的内容包
	// 必传params JwtGenPipe jwt的配置信息
	// 必传db redis Client
	JwtExchange = &RunnerContext[*JwtCheckDep, *JwtGenPipe, rueidis.Client, string]{
		Name: "jwt交换",
		Key:  "jwt_exchange",
		call: func(ctx iris.Context, origin *JwtCheckDep, params *JwtGenPipe, db rueidis.Client, more ...any) *RunResp[string] {
			if params == nil {
				params = new(JwtGenPipe)
			}

			// 先判断当前环境是否合规
			resp := JwtVisit.call(ctx, origin, nil, db)
			if resp.Err != nil {
				return NewPipeErr[string](resp.Err)
			}
			flatMap := resp.Result

			if flatMap.IsShort() {
				return NewPipeErr[string](errors.New("短令牌无法生成短令牌"))
			}

			helper := NewJwtHelper(db)

			raw := flatMap.Raw
			raw = strings.TrimPrefix(raw, JwtPrefix)

			// 生成 short token
			shortToken := ut.RandomStr(JwtShortLen)
			// 保存到redis
			shortRedisKey := helper.JwtShortRedisGenKey(shortToken)
			err := helper.JwtSaveToken(ctx, shortRedisKey, raw, params.GetExpire(time.Hour))
			if err != nil {
				return NewPipeErr[string](err)
			}

			return NewPipeResult[string](shortToken)
		},
	}
)
