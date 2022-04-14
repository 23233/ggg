package mab

import (
	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/errors"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/kataras/iris/v12"
)

func LimitHandler(l *limiter.Limiter, errBack ...func(*errors.HTTPError, iris.Context)) iris.Handler {
	return func(ctx iris.Context) {
		httpError := tollbooth.LimitByRequest(l, ctx.ResponseWriter(), ctx.Request())
		if httpError != nil {
			if len(errBack) >= 1 {
				if errBack[0] != nil {
					errBack[0](httpError, ctx)
				}
			} else {
				ctx.ContentType(l.GetMessageContentType())
				ctx.StatusCode(httpError.StatusCode)
				_, _ = ctx.WriteString(httpError.Message)
				ctx.StopExecution()
				return
			}
		}
		ctx.Next()
	}
}
