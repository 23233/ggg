package sv

import (
	"github.com/kataras/iris/v12"
	"reflect"
	"strings"
)

var (
	GlobalFailFunc = func(err error, ctx iris.Context) {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{"detail": err.Error()})
		return
	}
	GlobalContextKey = "sv"
)

func Run(valid interface{}, mode ...string) iris.Handler {
	var m string
	if len(mode) >= 1 {
		m = mode[0]
	}

	return func(ctx iris.Context) {
		// 回复到初始状态
		s := reflect.TypeOf(valid).Elem()
		newS := reflect.New(s)
		v := newS.Interface()
		var err error
		switch m {
		case "query":
			err = ctx.ReadQuery(v)
			break
		case "json":
			err = ctx.ReadJSON(v)
			break
		case "xml":
			err = ctx.ReadXML(v)
			break
		case "form":
			err = ctx.ReadForm(v)
			break
		default:
			if ctx.Method() == "GET" {
				err = ctx.ReadQuery(v)
			} else {
				contentType := ctx.GetContentTypeRequested()
				if contentType == "application/x-www-form-urlencoded" || strings.HasPrefix(contentType, "multipart/form-data") {
					err = ctx.ReadForm(v)
				} else if contentType == "application/xml" {
					err = ctx.ReadXML(v)
				} else {
					err = ctx.ReadJSON(v)
				}
			}
			break
		}
		if err != nil {
			Warning.Printf("read valid data fail: %s", err.Error())
			GlobalFailFunc(err, ctx)
			return
		}

		if err := GlobalValidator.Check(v); err != nil {
			Warning.Printf("valid fields fail: %s", err.Error())
			GlobalFailFunc(err, ctx)
			return
		}
		// this is point structv
		ctx.Values().Set(GlobalContextKey, v)
		ctx.Next()
	}
}
