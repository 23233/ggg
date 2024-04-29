package ut

import (
	"github.com/23233/ggg/logger"
	"github.com/kataras/iris/v12"
	"net/http"
)

func IrisErrReturn(ctx iris.Context, input any, statusCode int, businessCode int) {
	ctx.StatusCode(statusCode)
	var content any = ""
	switch input.(type) {
	case string:
		content = input.(string)
		break
	case error:
		content = input.(error).Error()
		break
	default:
		content = input
	}
	_ = ctx.JSON(iris.Map{"detail": content, "code": businessCode})
	return
}

func IrisErr(ctx iris.Context, input any) {
	IrisErrReturn(ctx, input, http.StatusBadRequest, http.StatusBadRequest)
}

func IrisErrLog(ctx iris.Context, err error, msg string) {
	if err != nil {
		logger.J.ErrorE(err, msg)
	}
	IrisErr(ctx, msg)
	return
}
