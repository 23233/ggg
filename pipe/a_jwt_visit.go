package pipe

import (
	"context"
	"github.com/23233/ggg/logger"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"strings"
	"time"
)

var (
	// JwtVisit jwt一整套流程整合 对jwt自动续期
	// 必传origin JwtCheckDep jwt请求头包 需要有 Authorization 字段
	// 必传db 为redis Client
	JwtVisit = &RunnerContext[*JwtCheckDep, any, rueidis.Client, JwtFlatBase]{
		Name: "jwt显示",
		Key:  "jwt_visit",
		call: func(ctx iris.Context, origin *JwtCheckDep, params any, db rueidis.Client, more ...any) *RunResp[JwtFlatBase] {

			dep := origin
			resp := JwtFlat.call(ctx, dep.Authorization, nil, db)
			if resp.Err != nil {
				return NewPipeErr[JwtFlatBase](errors.Wrap(resp.Err, "解析结构"))
			}
			dep.FlatMap = resp.Result

			resp2 := JwtCheck.call(ctx, dep, nil, db)
			if resp2.Err != nil {
				return NewPipeErr[JwtFlatBase](errors.Wrap(resp2.Err, "验证安全"))
			}

			// 进行续期
			pack := resp.Result

			// 续期
			isShort := pack.IsShort()

			rdb := db
			helper := NewJwtHelper(rdb)

			if isShort {
				// 如果是short
				shortToken := strings.TrimPrefix(pack.Raw, JwtShortPrefix)
				shortRedisKey := helper.JwtShortRedisGenKey(shortToken)
				expireSec := int64(time.Hour.Seconds())
				// 仅当新到期时间大于当前到期时间时设置到期时间
				err := rdb.Do(context.Background(), rdb.B().Expire().Key(shortRedisKey).Seconds(expireSec).Gt().Build()).Error()
				if err != nil {
					logger.J.ErrorE(err, "续期short token失败 %s ", shortToken)
				}
			} else {
				// 非short 则续期
				//rawToken := strings.TrimPrefix(pack.Raw, JwtPrefix)
				redisKey := helper.JwtRedisGenKey(pack.UserId, pack.Env)
				// 仅当新到期时间大于当前到期时间时设置到期时间
				expireSec := int64(new(JwtGenPipe).GetExpire().Seconds())
				cmdList := rdb.B().Expire().Key(redisKey).Seconds(expireSec).Gt().Build()
				err := rdb.Do(context.Background(), cmdList).Error()
				if err != nil {
					logger.J.ErrorE(err, "续期token失败 %s %s", pack.UserId, pack.Env)
				}
			}
			return NewPipeResult(resp.Result)
		},
	}
)
