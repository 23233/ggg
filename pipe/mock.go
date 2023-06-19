package pipe

import (
	"github.com/kataras/iris/v12"
	irisContext "github.com/kataras/iris/v12/context"
	"net/http"
)

func mockIrisContext() iris.Context {
	rawReq, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		return nil
	}
	ctx := irisContext.NewContext(iris.New())

	ctx.ResetRequest(rawReq)

	return ctx
}
