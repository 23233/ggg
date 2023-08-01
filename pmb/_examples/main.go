package main

import (
	"context"
	"github.com/23233/ggg/pmb"
	"github.com/23233/ggg/ut"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"github.com/redis/rueidis"
	"go.mongodb.org/mongo-driver/bson"
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
	bk.RegistryLoginRegRoute(party, true)

	pmb.UserInstance.SetConn(bk.CloneConn())

	// 还可以测试action
	model := pmb.NewSchemaModel[any](new(testModelStruct), bk.CloneConn().Db())

	var action = pmb.NewAction("设置desc为新的", new(testActionDesc))
	action.SetCall(func(ctx iris.Context, rows []map[string]any, formData map[string]any, user *pmb.SimpleUserModel, model *pmb.SchemaModel[any]) (any, error) {
		// 批量变更
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row[ut.DefaultUidTag].(string))
		}
		result, err := model.GetCollection().UpdateAll(ctx, bson.M{ut.DefaultUidTag: bson.M{"$in": ids}}, bson.M{
			"$set": bson.M(formData),
		})
		if err != nil {
			return nil, err
		}
		println(result.ModifiedCount)
		return iris.Map{}, nil
	})
	model.AddAction(action)

	// 执行条件
	var action2 = pmb.NewAction("判断执行条件", nil)
	action2.Conditions = append(action2.Conditions, ut.Kov{
		Key:   "desc",
		Op:    "ne",
		Value: nil,
	})
	action2.SetCall(func(ctx iris.Context, rows []map[string]any, formData map[string]any, user *pmb.SimpleUserModel, model *pmb.SchemaModel[any]) (any, error) {
		return iris.Map{}, nil
	})
	model.AddAction(action2)

	var action3 = pmb.NewAction("desc必须为323423423", nil)
	action3.Conditions = append(action2.Conditions, ut.Kov{
		Key:   "desc",
		Op:    "eq",
		Value: "323423423",
	})
	action3.SetCall(func(ctx iris.Context, rows []map[string]any, formData map[string]any, user *pmb.SimpleUserModel, model *pmb.SchemaModel[any]) (any, error) {
		return iris.Map{}, nil
	})
	model.AddAction(action3)

	bk.AddModel(model)
	bk.AddModelAny(new(testModelTwo))
	bk.AddModelAny(new(testModelThree))

	_ = app.Listen(":8080")

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
	Name string `json:"name,omitempty" bson:"name,omitempty"`
	Age  uint   `json:"age,omitempty" bson:"age,omitempty"`
	Desc string `json:"desc,omitempty" bson:"desc,omitempty"`
	Code int    `json:"code,omitempty" bson:"code,omitempty"`
}

type testModelThree struct {
	Address struct {
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
