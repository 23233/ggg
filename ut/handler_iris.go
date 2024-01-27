package ut

import "github.com/kataras/iris/v12"

func IrisErrReturn(ctx iris.Context, msg string, err ...error) {
	ctx.StatusCode(iris.StatusBadRequest)
	var content = msg
	if len(content) < 1 {
		if len(err) >= 1 {
			content = err[0].Error()
		}
	}
	_ = ctx.JSON(iris.Map{"detail": content, "code": 400})
	return
}
