package pipe

import (
	"github.com/23233/ggg/logger"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"strings"
	"time"
)

var (
	JwtVisit = &RunnerContext[*JwtCheckDep, any, rueidis.Client, map[string]any]{
		Name: "jwt显示",
		Key:  "jwt_visit",
		call: func(ctx iris.Context, origin *JwtCheckDep, params any, db rueidis.Client, more ...any) *RunResp[map[string]any] {

			dep := origin
			resp := JwtFlat.call(ctx, dep.Authorization, nil, db)
			if resp.Err != nil {
				return newPipeErr[map[string]any](errors.Wrap(resp.Err, "解析结构"))
			}

			resp2 := JwtCheck.call(ctx, dep, nil, db)
			if resp2.Err != nil {
				return newPipeErr[map[string]any](errors.Wrap(resp2.Err, "验证安全"))
			}

			// 进行续期
			pack := resp.Result
			// 续期
			isShort := false
			if v, ok := pack["Short"]; ok {
				isShort = v.(bool)
			}
			rdb := db
			helper := NewJwtHelper(rdb)

			if isShort {
				// 如果是short
				shortToken := strings.TrimPrefix(pack["Raw"].(string), JwtShortPrefix)
				shortRedisKey := helper.JwtShortRedisGenKey(shortToken)
				expireSec := int64(time.Hour.Seconds())
				// 仅当新到期时间大于当前到期时间时设置到期时间
				err := rdb.Do(ctx, rdb.B().Expire().Key(shortRedisKey).Seconds(expireSec).Gt().Build()).Error()
				if err != nil {
					logger.J.ErrorE(err, "续期short token失败 %s ", shortToken)
				}
			} else {
				// 非short 则续期
				userId := pack["userId"].(string)
				packEnv := pack["env"].(string)
				rawToken := pack["Raw"].(string)
				rawToken = strings.TrimPrefix(rawToken, JwtPrefix)
				redisKey := helper.JwtRedisGenKey(userId, packEnv)
				// 仅当新到期时间大于当前到期时间时设置到期时间
				expireSec := int64(new(JwtGenPipe).GetExpire().Seconds())

				cmdList := rdb.B().Expire().Key(redisKey).Seconds(expireSec).Gt().Build()
				err := rdb.Do(ctx, cmdList).Error()
				if err != nil {
					logger.J.ErrorE(err, "续期token失败 %s %s", userId, packEnv)
				}
			}
			return newPipeResult(resp.Result)
		},
	}
)
