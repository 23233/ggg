package pipe

import (
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"time"
)

type JwtGenPipe struct {
	ExpireDuration time.Duration `json:"expire_duration,omitempty"` // 过期时间 默认24小时
	Strict         bool          `json:"strict,omitempty"`          // 严格模式才执行同一个环境仅单个登录
	Force          bool          `json:"force,omitempty"`           // 强制覆盖 仅在严格模式下启用
}
type PipeJwtDep struct {
	Env    string
	UserId string
}

func (c *JwtGenPipe) GetExpire(defaultTimes ...time.Duration) time.Duration {
	expire := c.ExpireDuration
	if expire < 1 {
		if len(defaultTimes) > 0 {
			expire = defaultTimes[0]
		} else {
			expire = time.Hour * 24
		}
	}
	return expire
}

var (
	// JwtGen jwt 结构设计 Strict模式下 用户一个env下仅可登陆一个设备  [key为userId:env]:value 为token
	// 必传origin jwt设置
	// 必传params 生成参数
	// 必传db redis Client
	JwtGen = &RunnerContext[*PipeJwtDep, *JwtGenPipe, rueidis.Client, string]{
		Name: "jwt生成",
		Key:  "jwt_gen",
		call: func(ctx iris.Context, origin *PipeJwtDep, params *JwtGenPipe, db rueidis.Client, more ...any) *RunResp[string] {
			if origin == nil {
				return newPipeErr[string](PipeDepError)
			}

			if params == nil {
				params = new(JwtGenPipe)
			}

			helper := NewJwtHelper(db)

			redisKey := helper.JwtRedisGenKey(origin.UserId, origin.Env)

			if params.Strict {
				resp := helper.JwtRedisGetKey(ctx, redisKey)
				if resp.Error() != nil {
					// 只要不是为空错误 则为其他错误都直接返回
					if resp.Error() != rueidis.Nil {
						return newPipeErr[string](resp.Error())
					}
				}

				// token如果存在 但是不强制刷新
				st, _ := resp.ToString()
				if len(st) > 0 && !params.Force {
					return newPipeErr[string](errors.New("当前环境有其他设备在线"))
				}
			}

			token := helper.GenJwtToken(origin.UserId, origin.Env)

			// 直接写入
			err := helper.JwtSaveToken(ctx, redisKey, token, params.GetExpire())
			if err != nil {
				return newPipeErr[string](err)
			}

			// 返回
			return newPipeResult[string](token)
		},
	}
)
