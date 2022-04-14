package sv

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"
	"testing"
)

func TestRun(t *testing.T) {
	type req struct {
		Name string `json:"name" url:"name" comment:"name" validate:"required"`
	}
	type req2 struct {
		Desc string `json:"desc" url:"desc" comment:"desc"`
	}

	app := iris.New()
	app.Get("/", Run(new(req)), func(ctx context.Context) {
		req := ctx.Values().Get("sv").(*req)
		_, _ = ctx.JSON(iris.Map{"name": req.Name})
	})
	app.Get("/111", Run(new(req2)), func(ctx context.Context) {
		req := ctx.Values().Get("sv").(*req2)
		_, _ = ctx.JSON(iris.Map{"name": req.Desc})
	})

	e := httptest.New(t, app)
	//e.GET("/").Expect().Status(httptest.StatusBadRequest)
	e.GET("/111").WithQuery("desc", "123123").Expect().Status(httptest.StatusOK)
}
