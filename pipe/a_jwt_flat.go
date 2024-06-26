package pipe

import (
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"strings"
)

var (
	// JwtFlat jwt解构出内容包
	// 必传origin 为Authorization头的字符串内容
	// 必传db 为redis Client
	JwtFlat = &RunnerContext[string, any, rueidis.Client, JwtFlatBase]{
		Name: "jwt解构",
		Key:  "jwt_flat",
		call: func(ctx iris.Context, origin string, params any, rdb rueidis.Client, more ...any) *RunResp[JwtFlatBase] {

			if len(origin) < 1 {
				return NewPipeErr[JwtFlatBase](errors.New("获取token令牌错误"))
			}

			helper := NewJwtHelper(rdb)
			isShort := strings.HasPrefix(origin, JwtShortPrefix)

			auth := origin
			// 如果是short token
			if isShort {
				shortToken := strings.TrimPrefix(origin, JwtShortPrefix)
				if len(shortToken) != JwtShortLen {
					return NewPipeErr[JwtFlatBase](errors.New("短token令牌格式错误"))
				}

				// 通过short token 获取完整token
				resp := rdb.Do(ctx, rdb.B().Get().Key(helper.JwtShortRedisGenKey(shortToken)).Build())
				if resp.Error() != nil {
					return NewPipeErr[JwtFlatBase](resp.Error())
				}
				st, err := resp.ToString()
				if err != nil {
					return NewPipeErr[JwtFlatBase](err)
				}
				auth = JwtPrefix + st

			}

			// 如果不是 Bearer 则是格式错误
			if !strings.HasPrefix(auth, JwtPrefix) {
				return NewPipeErr[JwtFlatBase](errors.New("token令牌格式错误"))
			}

			// 解构出map
			tk, err := helper.TokenExtract(auth, ctJwt)
			if err != nil {
				return NewPipeErr[JwtFlatBase](err)
			}
			tk.Raw = auth

			return NewPipeResult(*tk)
		},
	}
)
