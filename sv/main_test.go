package sv

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"strconv"
	"sync"
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
	app.Get("/", Run(new(req)), func(ctx iris.Context) {
		req := ctx.Values().Get("sv").(*req)
		ctx.JSON(iris.Map{"name": req.Name})
	})
	app.Get("/111", Run(new(req2)), func(ctx iris.Context) {
		req := ctx.Values().Get("sv").(*req2)
		ctx.JSON(iris.Map{"desc": req.Desc})
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusBadRequest)

	type ttss struct {
		Path   string
		Key    string
		Val    string
		Status int
		Pass   bool
	}
	var testList = make([]*ttss, 0)
	for i := 0; i < 20; i++ {
		testList = append(testList, &ttss{
			Path:   "/",
			Key:    "name",
			Val:    "",
			Status: iris.StatusBadRequest,
			Pass:   false,
		})
	}
	for i := 0; i < 20; i++ {
		testList = append(testList, &ttss{
			Path:   "/",
			Key:    "name",
			Val:    "33333" + strconv.Itoa(i),
			Status: iris.StatusOK,
			Pass:   true,
		})
	}
	for i := 0; i < 20; i++ {
		testList = append(testList, &ttss{
			Path:   "/111",
			Key:    "desc",
			Val:    "1114444" + strconv.Itoa(i),
			Status: iris.StatusOK,
			Pass:   true,
		})
	}

	var wg sync.WaitGroup

	for _, m := range testList {
		wg.Add(1)
		m := m
		go func() {
			resp := e.GET(m.Path).WithQuery(m.Key, m.Val).Expect().Status(m.Status)
			if m.Pass {
				resp.JSON().Object().Value(m.Key).Equal(m.Val)
			}
			wg.Done()
		}()
	}
	wg.Wait()

}
