package sv

import (
	"github.com/kataras/iris/v12"
	"reflect"
)

const (
	GKey = "sv"
)

var (
	GlobalFailFunc = func(err error, ctx iris.Context) {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{"detail": err.Error()})
		return
	}
	GlobalContextKey = GKey
)

func Run(valid interface{}) iris.Handler {
	return func(ctx iris.Context) {
		ctx.RecordRequestBody(true)
		// 回复到初始状态
		s := reflect.TypeOf(valid).Elem()
		newS := reflect.New(s)
		v := newS.Interface()
		err := ctx.ReadBody(v)
		if err != nil {
			Warning.Printf("read valid data fail: %s", err.Error())
			GlobalFailFunc(err, ctx)
			return
		}

		if err = GlobalValidator.Check(v); err != nil {
			Warning.Printf("valid fields fail: %s", err.Error())
			GlobalFailFunc(err, ctx)
			return
		}
		// this is point struct
		ctx.Values().Set(GKey, v)
		ctx.Next()
	}
}
