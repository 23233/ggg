package pipe

import (
	"github.com/23233/ggg/contentSafe"
	"github.com/kataras/iris/v12"
)

var (
	// ImgSafe 图像安全校验 必传 origin 是一个图像url地址
	ImgSafe = &RunnerContext[string, any, any, bool]{
		Name: "图片安全校验",
		Key:  "img_safe",
		call: func(ctx iris.Context, origin string, params any, db any, more ...any) *RunResp[bool] {

			if len(origin) < 1 {
				return newPipeResult[bool](false)
			}
			pass, err := contentSafe.C.AutoHitImg(origin)
			return newPipeResultErr[bool](pass, err)
		},
	}
)
