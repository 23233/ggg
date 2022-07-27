package main

import (
	_ctx "context"
	"github.com/23233/ggg/smab"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
	"time"
)

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

type DefaultField struct {
	Id       primitive.ObjectID `bson:"_id" json:"id" comment:"Id"`
	UpdateAt time.Time          `bson:"update_at" json:"update_at" comment:"更新时间"`
}

func (u *DefaultField) BeforeInsert(ctx _ctx.Context) error {
	if u.Id.IsZero() {
		u.Id = primitive.NewObjectID()
	}
	u.UpdateAt = time.Now().Local()
	return nil
}

func (u *DefaultField) BeforeUpdate(ctx _ctx.Context) error {
	u.UpdateAt = time.Now().Local()
	return nil
}

func (u *DefaultField) BeforeUpsert(ctx _ctx.Context) error {
	u.UpdateAt = time.Now().Local()
	return nil
}

type InlineTest struct {
	Event   string               `json:"event" bson:"event"  comment:"事件名"`
	Age     int64                `json:"age" bson:"age"`
	Year    uint64               `json:"year" bson:"year"`
	Tags    []string             `json:"tags" bson:"tags"`
	Gps     []float64            `json:"gps" bson:"gps"`
	ObjId   primitive.ObjectID   `json:"obj_id" bson:"obj_id" mab:"fk=WorkDetail"`
	ObjList []primitive.ObjectID `json:"obj_list" bson:"obj_list"`
}

type WorkDetail struct {
	DefaultField `bson:",inline,flatten"`
	Name         string     `json:"name" bson:"name"`
	Desc         string     `json:"desc" bson:"desc" mab:"t=markdown"`
	Bll          []string   `json:"bll" bson:"bll"`
	TestInline   InlineTest `json:"test_inline" bson:"test_inline"`
	ImgTest      []string   `json:"img_test,omitempty" bson:"img_test,omitempty" mab:"t=img,thumbnail"`
}

func (c *WorkDetail) Alias() string {
	return "_作品3-3_作品详情-4"
}

type ComplexModel struct {
	DefaultField    `bson:",inline,flatten"`
	Name            string       `json:"name" bson:"name" mab:"t=textarea"`
	Age             uint64       `json:"age" bson:"age"`
	Point           float64      `json:"point" bson:"point"`
	SelectTime      time.Time    `json:"select_time" bson:"select_time"`
	SliceBase       []string     `json:"slice_base" bson:"slice_base" mab:"t=img"`
	TestInline      InlineTest   `json:"test_inline" bson:"test_inline,inline" comment:"内链"`
	TestNotInline   InlineTest   `json:"test_not_inline" bson:"test_not_inline" comment:"非内链"`
	TestSliceInline []InlineTest `json:"test_slice_inline" bson:"test_slice_inline" comment:"数组链接"`
	HiddenTable     string       `json:"hidden_table" bson:"hidden_table" comment:"表格中不显示" mab:"hide=table"`
	HiddenAll       string       `json:"hidden_all" bson:"hidden_all" comment:"全都不显示" mab:"hide"`
	HiddenForm      string       `json:"hidden_form" bson:"hidden_form" comment:"表单中不显示" mab:"hide=form"`
	HiddenAdd       string       `json:"hidden_add" bson:"hidden_add" comment:"新增不显示" mab:"hide=add"`
	HiddenEdit      string       `json:"hidden_edit" bson:"hidden_edit" comment:"修改不显示" mab:"hide=edit"`
}

func (c *ComplexModel) Alias() string {
	return "_组名1-1_复杂-3"
}

type WorkInfo struct {
	TestInline InlineTest `json:"test_inline" bson:"inline"`
	Name       string     `json:"name" bson:"name"`
}

func (c *WorkInfo) Alias() string {
	return "_组名2-2_作品内容-2"
}

type User struct {
	DefaultField `bson:",inline,flatten"`
	Name         string `json:"name" bson:"name" comment:"用户名"`
}

func (c *User) Alias() string {
	return "用户1-1"
}

type TwoInline struct {
	DefaultField `bson:",inline,flatten"`
	InlineTest   `bson:",inline,flatten"`
}

func (c *TwoInline) Alias() string {
	return "两个内联3-3"
}

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	customLogger := logger.New(logger.Config{
		// Status displays status code
		Status: true,
		// IP displays request's remote address
		IP: true,
		// Method displays the http method
		Method: true,
		// Path displays the request path
		Path: true,
		// Query appends the url query to the Path.
		Query: true,

		// Columns: true,

		// if !empty then its contents derives from `ctx.Values().Get("logger_message")
		// will be added to the logs.
		MessageContextKeys: []string{"logger_message"},

		// if !empty then its contents derives from `ctx.GetHeader("SmUserModel-Agent")
		MessageHeaderKeys: []string{"SmUserModel-Agent"},
	})
	app.Use(customLogger)
	tmpl := iris.Blocks("_examples/templates", ".html")
	app.RegisterView(tmpl)
	var _, err = smab.New(getTestDb(), smab.Configs{
		App: app,
		ModelList: []interface{}{
			new(WorkDetail),
			new(WorkInfo),
			new(ComplexModel),
			new(User),
			new(TwoInline),
		},
		CasbinConfig: smab.CasbinConfigDefine{
			Uri: getEnv("mongo_uri", ""),
		},
		GlobalQianKun: []smab.QianKunConfigExtra{
			{
				Name:  "app1",
				Entry: "http://localhost:8001",
				Path:  "/app1",
				Label: "应用1",
			},
		},
		WelComeConfig: smab.WelComeConfigDefine{
			Title:    "欢迎登陆吊炸天的后台管理面板",
			MainText: []string{"请不要作死", "都有记录", "违规直接封号"},
			Desc:     []string{"好的", "我都懂", "发个什么通知也是可以的"},
		},
		PublicKey: getEnv("public_key", ""),
	})
	if err != nil {
		panic(err)
	}

	app.Get("/", func(ctx iris.Context) {
		_ = ctx.View("index")
	})

	app.Post("/test", func(ctx iris.Context) {
		_ = ctx.JSON(iris.Map{"detail": "处理完成"})
	})

	go func() {
		var m = smab.GenTaskAtRoot("新测试任务", "任务描述")
		m.Group = "赵日天"
		m.Type = 10
		m.InjectData = `{"obj_id":"98439834"}`
		m.Action = smab.PassOrNotReasonAction("/test", "/test")
		m.Content = `### 我是赵日天  我今天贼开心 你开不开心啊 \n 这是换行的图片 \n ![这是个图片](https://huyaimg.msstatic.com/avatar/1026/a3/21c624bf332d4ad165d20f66d5b590_180_135.jpg)`
		err := smab.CreateTask(_ctx.Background(), m)
		if err != nil {
			panic(err)
		}
	}()

	_ = app.Listen(":8080")

}
