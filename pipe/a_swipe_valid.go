package pipe

import (
	"encoding/base64"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
)

var (
	// SwipeValidGet 滑块验证码获取
	// 必传db SwipeValidCode 的实例
	SwipeValidGet = &RunnerContext[any, any, *SwipeValidCode, string]{
		Name: "滑块验证码获取",
		Key:  "swipe_valid_get",
		call: func(ctx iris.Context, origin any, params any, db *SwipeValidCode, more ...any) *RunResp[string] {

			sp, err := db.Gen(ctx)
			if err != nil {
				return NewPipeErr[string](err)
			}
			raw := sp.ToString()
			sEnc := base64.StdEncoding.EncodeToString([]byte(raw))

			return NewPipeResult(sEnc)
		},
	}
	// SwipeValidCheck 滑块验证码验证
	// 必传db SwipeValidCode 的实例
	SwipeValidCheck = &RunnerContext[string, any, *SwipeValidCode, *SwipeValid]{
		Name: "滑块验证码验证",
		Key:  "swipe_valid_check",
		call: func(ctx iris.Context, origin string, params any, db *SwipeValidCode, more ...any) *RunResp[*SwipeValid] {
			check, err := db.Check(ctx, origin)
			if err != nil {
				return NewPipeErr[*SwipeValid](errors.New("验证失败 请重试"))
			}
			return NewPipeResult(check)
		},
	}
)
