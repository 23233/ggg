package main

import (
	"context"
	"github.com/23233/ggg/pmb"
	"github.com/23233/ggg/ut"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"github.com/redis/rueidis"
)

func getMg() *qmgo.Database {
	err := godotenv.Load()
	if err != nil {
		panic("获取配置文件出错")
	}

	mgCli, err := qmgo.NewClient(context.TODO(), &qmgo.Config{Uri: ut.GetEnv("MongoUri", "")})
	if err != nil {
		panic(err)
	}
	return mgCli.Database("ttts")
}
func getRdbInfo() (string, string) {
	return ut.GetEnv("RedisAddress", ""), ut.GetEnv("RedisPassword", "")
}
func getRdb() rueidis.Client {
	err := godotenv.Load()
	if err != nil {
		panic("获取配置文件出错")
	}

	address, password := getRdbInfo()

	rdb, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{address},
		Password:    password,
		SelectDB:    6,
	})
	if err != nil {
		panic(err)
	}
	return rdb
}

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	app.Configure(iris.WithoutBodyConsumptionOnUnmarshal)

	bk := pmb.NewBackend()
	bk.AddDb(getMg())
	bk.AddRdb(getRdb())
	bk.AddRbacUseUri(getRdbInfo())
	bk.RegistryRoute(app)
	bk.RegistryLoginRegRoute(app, true)

	pmb.UserInstance.SetConn(bk.CloneConn())

	app.Post("/reg", pmb.UserInstance.RegistryUseUserNameHandler())
	app.Post("/login", pmb.UserInstance.LoginUseUserNameHandler())

	_ = app.Listen(":8080")

}
