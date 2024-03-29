package main

import (
	"context"
	"fmt"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/pmb"
	"github.com/23233/ggg/ut"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"github.com/redis/rueidis"
	"go.mongodb.org/mongo-driver/bson"
	"time"
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

type testActionDesc struct {
	Desc string `json:"desc"`
}

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")
	app.Configure(iris.WithoutBodyConsumptionOnUnmarshal)
	party := app.Party("/")

	bk := pmb.NewBackend()
	bk.AddDb(getMg())
	bk.AddRdb(getRdb())
	bk.AddRbacUseUri(getRdbInfo())
	bk.RegistryRoute(party)

	bk.InsertLogModel()
	bk.InsertUserModel()

	pmb.UserInstance.SetConn(bk.CloneConn())

	model := pmb.NewSchemaModel[any](new(testModelStruct), bk.CloneConn().Db())

	// 测试dynamicField
	df1 := pmb.NewDynamicField("tt", "动态字段1").SetTriggerInterval()
	df1.AddCall(func(fieldId string, model pmb.IModelItem, user *pmb.SimpleUserModel, row map[string]any) (*pmb.DynamicResult, error) {
		var result = new(pmb.DynamicResult)
		result.Normal("赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1 赵日天1/1")
		return result, nil
	})

	df2 := pmb.NewDynamicField("bb", "动态字段2").SetTriggerClick()
	df2.AddCall(func(fieldId string, model pmb.IModelItem, user *pmb.SimpleUserModel, row map[string]any) (*pmb.DynamicResult, error) {
		var result = new(pmb.DynamicResult)
		result.Raw([]any{123, "3456", "456"})
		return result, nil
	})

	model.AddDynamicField(df1)
	model.AddDynamicField(df2)

	// 测试action
	var action = pmb.NewAction[map[string]any, *testActionDesc]("设置desc为新的", new(testActionDesc))
	action.GetBase().Prefix = "前缀测试"
	action.GetBase().TableEmptySelectUseAllSheet = true
	action.SetCall(func(ctx iris.Context, args any) (any, error) {

		// 这里没办法跟上面对齐 主要是F没有地方透传 所以只能是map
		part := args.(*pmb.ActionPostArgs[map[string]any, map[string]any])

		// 批量变更
		ids := make([]string, 0, len(part.Rows))
		for _, row := range part.Rows {
			ids = append(ids, row[ut.DefaultUidTag].(string))
		}

		var filter = bson.M{ut.DefaultUidTag: bson.M{"$in": ids}}
		if len(ids) < 1 {
			if !action.GetBase().TableEmptySelectUseAllSheet {
				return nil, errors.New("未选中任意行数 无法变更")
			}
			filter = bson.M{}
		}

		result, err := model.GetCollection().UpdateAll(ctx, filter, bson.M{
			"$set": bson.M(part.FormData),
		})
		if err != nil {
			return nil, err
		}
		return iris.Map{"detail": fmt.Sprintf("设置成功%d条", result.ModifiedCount)}, nil
	})
	model.AddAction(action)

	// 执行条件
	var action2 = pmb.NewAction[map[string]any, map[string]any]("判断执行条件", nil)
	action2.GetBase().AddCondition(ut.Kov{
		Key:   "desc",
		Op:    "ne",
		Value: nil,
	})
	action2.SetCall(func(ctx iris.Context, args any) (any, error) {
		return iris.Map{}, nil
	})
	model.AddAction(action2)

	var action3 = pmb.NewAction[map[string]any, map[string]any]("desc必须为323423423", nil)
	action3.GetBase().AddCondition(ut.Kov{
		Key:   "desc",
		Op:    "eq",
		Value: "323423423",
	})
	action3.SetCall(func(ctx iris.Context, args any) (any, error) {
		return iris.Map{}, nil
	})
	model.AddAction(action3)

	bk.AddModel(model)
	bk.AddModelAny(new(testModelTwo))
	bk.AddModelAny(new(testModelThree))
	go func() {
		for {
			time.Sleep(5 * time.Second)
			bk.SendMsg(context.TODO(), "123123", "测试一下消息1")
		}
	}()

	_ = app.Run(iris.Addr(":8080"))

}

type testModelStruct struct {
	Name         string    `json:"name,omitempty" bson:"name,omitempty"`
	Age          uint      `json:"age,omitempty" bson:"age,omitempty"`
	Desc         string    `json:"desc,omitempty" bson:"desc,omitempty"`
	Code         int       `json:"code,omitempty" bson:"code,omitempty"`
	Tips         []string  `json:"tips,omitempty" bson:"tips,omitempty"`
	StringNoType string    `json:"string_no_type,omitempty" bson:"string_no_type,omitempty"`
	Tj           []int     `json:"tj,omitempty" bson:"tj,omitempty"`
	Tj8          []int8    `json:"tj_8,omitempty" bson:"tj_8,omitempty"`
	Tj16         []int16   `json:"tj_16,omitempty" bson:"tj_16,omitempty"`
	Tj32         []int32   `json:"tj_32,omitempty" bson:"tj_32,omitempty"`
	Tj64         []int64   `json:"tj_64,omitempty" bson:"tj_64,omitempty"`
	Uj           []uint    `json:"uj,omitempty" bson:"uj,omitempty"`
	Uj8          []uint8   `json:"uj_8,omitempty" bson:"uj_8,omitempty"`
	Uj16         []uint16  `json:"uj_16,omitempty" bson:"uj_16,omitempty"`
	Uj32         []uint32  `json:"uj_32,omitempty" bson:"uj_32,omitempty"`
	Uj64         []uint64  `json:"uj_64,omitempty" bson:"uj_64,omitempty"`
	Fj32         []float32 `json:"fj_32,omitempty" bson:"fj_32,omitempty"`
	Fj64         []float64 `json:"fj_64,omitempty" bson:"fj_64,omitempty"`
	Address      struct {
		Position string `json:"position,omitempty" bson:"position,omitempty"`
	} `json:"address,omitempty" bson:"address,omitempty"`
	TestInline struct {
		Music string `json:"music,omitempty" bson:"music,omitempty"`
	} `json:"test_inline,omitempty" bson:"test_inline,omitempty"`
	Location struct {
		Type        string    `json:"type,omitempty" bson:"type,omitempty"`
		Coordinates []float32 `json:"coordinates,omitempty" bson:"coordinates,omitempty"`
	} `json:"location,omitempty" bson:"location,omitempty"`
	ArrayInline  []map[string]any `json:"array_inline,omitempty" bson:"array_inline,omitempty"`
	NormalInline struct {
		Inline  string  `json:"inline,omitempty" bson:"inline,omitempty"`
		Number  float64 `json:"number,omitempty" bson:"number,omitempty"`
		Integer uint64  `json:"integer,omitempty" bson:"integer,omitempty"`
	} `json:"normal_inline,omitempty" bson:"normal_inline,omitempty"`
	Fk     string `json:"fk,omitempty" bson:"fk,omitempty"`
	UserId string `json:"user_id,omitempty" bson:"user_id,omitempty"`
}

type testModelTwo struct {
	pipe.ModelBase `bson:",inline"`

	Name string `json:"name,omitempty" bson:"name,omitempty"`
	Age  uint   `json:"age,omitempty" bson:"age,omitempty"`
	Desc string `json:"desc,omitempty" bson:"desc,omitempty"`
	Code int    `json:"code,omitempty" bson:"code,omitempty"`
}

type testModelThree struct {
	pipe.ModelBase `bson:",inline"`
	Address        struct {
		Position string `json:"position,omitempty" bson:"position,omitempty"`
	} `json:"address,omitempty" bson:"address,omitempty"`
	TestInline struct {
		Music string `json:"music,omitempty" bson:"music,omitempty"`
	} `json:"test_inline,omitempty" bson:"test_inline,omitempty"`
	Location struct {
		Type        string    `json:"type,omitempty" bson:"type,omitempty"`
		Coordinates []float32 `json:"coordinates,omitempty" bson:"coordinates,omitempty"`
	} `json:"location,omitempty" bson:"location,omitempty"`
}
