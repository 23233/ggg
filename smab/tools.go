package smab

import (
	_ctx "context"
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"math/rand"
	"os"
)

// 错误返回
func fastError(err error, ctx iris.Context, msg ...string) {
	ctx.StatusCode(iris.StatusBadRequest)
	var m string
	if err == nil {
		m = "请求解析出错"
	} else {
		m = err.Error()
	}
	if len(msg) >= 1 {
		m = msg[0]
	}
	_ = ctx.JSON(iris.Map{
		"detail": m,
	})
	return
}

func fastMethodNotAllowedError(msg string, ctx iris.Context) {
	ctx.StatusCode(iris.StatusMethodNotAllowed)
	_ = ctx.JSON(iris.Map{
		"detail": msg,
	})
	return
}

func msgLog(msg string, args ...string) error {
	return errors.New(fmt.Sprintf("[%s]:%s", "smab", msg))
}

func stringsContains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func randomStr(n int) string {
	randBytes := make([]byte, n/2)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getTestDb() *qmgo.Database {
	mongoUri := getEnv("mongo_uri", "")
	ctx := _ctx.Background()
	mgCli, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: mongoUri})
	if err != nil {
		panic(err)
	}
	mgDb := mgCli.Database("ttt")
	return mgDb
}
