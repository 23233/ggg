package pmb

import (
	"context"
	"encoding/json"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/qiniu/qmgo"
	"github.com/redis/rueidis"
	"testing"
)

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

func createFullPostJson(fkUid string) *testModelStruct {

	inlineMap := map[string]any{
		"inline":  "内联1",
		"number":  -12323,
		"integer": 123,
	}

	var value = new(testModelStruct)
	value.Name = "名称"
	value.Age = 10
	value.Code = 100
	value.Desc = "描述"
	value.Tips = []string{"提示1", "提示2"}
	value.StringNoType = "默认是字符串"
	value.Tj = []int{20, 30}
	value.Tj8 = []int8{50, 80}
	value.Tj16 = []int16{2033, 3033}
	value.Tj32 = []int32{20333, 30333}
	value.Tj64 = []int64{203333, 303333}
	value.Uj = []uint{60, 70}
	// []uint8 会被unmashal成 string
	//value.Uj8 = []uint8{66, 77}
	value.Uj16 = []uint16{166, 177}
	value.Uj32 = []uint32{600, 700}
	value.Uj64 = []uint64{6000, 7000}
	value.Fj32 = []float32{8000.32, 9000.33}
	value.Fj64 = []float64{88000.32, 98000.33}
	value.Address.Position = "地址的位置"
	value.TestInline.Music = "音乐"
	value.Location.Type = ""
	value.Location.Coordinates = []float32{103.83797, 1.46103}
	value.ArrayInline = []map[string]any{inlineMap}
	value.NormalInline.Inline = inlineMap["inline"].(string)
	value.NormalInline.Number = float64(inlineMap["number"].(int))
	value.NormalInline.Integer = uint64(inlineMap["integer"].(int))
	value.Fk = fkUid

	return value
}

func TestNewSchemaModel(t *testing.T) {
	app := iris.New()
	app.Configure(iris.WithoutBodyConsumptionOnUnmarshal)

	inst := NewSchemaModel(new(testModelStruct), getMg())
	inst.Registry(app)

	action := new(SchemaModelAction)
	action.Name = "表操作"
	action.Types = []uint{0, 1}
	action.Form = nil
	action.SetCall(func(ctx iris.Context, rows []map[string]any, formData map[string]any, user *SimpleUserModel) (any, error) {
		return nil, nil
	})

	inst.AddAction(action)

	ac2 := new(SchemaModelAction)
	ac2.Name = "行操作"
	ac2.Types = []uint{1}
	ac2.Form = nil
	ac2.SetCall(func(ctx iris.Context, rows []map[string]any, formData map[string]any, user *SimpleUserModel) (any, error) {
		return nil, nil
	})

	inst.AddAction(ac2)

	prefix := "/" + inst.EngName
	var uid string

	e := httptest.New(t, app)

	t.Run("新增", func(t *testing.T) {
		fullPost := createFullPostJson("")
		resp := e.POST(prefix).WithJSON(fullPost).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey(ut.DefaultUidTag)
		respObj.ContainsKey("_id")
		respObj.ContainsKey("update_at")
		respObj.ContainsKey("create_at")
		respObj.ContainsSubset(fullPost)
		uid = respObj.Value(ut.DefaultUidTag).String().Raw()
	})

	t.Run("修改", func(t *testing.T) {
		inlineMap := map[string]any{
			"inline":  "内联122222",
			"number":  -34123,
			"integer": 123544,
		}
		inlineMap2 := map[string]any{
			"inline": "内联22",
		}
		fullPut := map[string]any{
			"name": "修改的名称",
			"age":  1090,
			"desc": "描述",
			"tips": []string{"字符串数组"},
			"tj":   []int{50, 100},
			"address": map[string]any{
				"position": "修改后的地址",
				"dontShow": "不应该出现",
			},
			"array_inline": []map[string]any{inlineMap, inlineMap2},
		}
		resp := e.PUT(prefix + "/" + uid).WithJSON(fullPut).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.NotContainsKey("desc")
		respObj.ContainsKey("update_at")
		respObj.ContainsKey("name")
		respObj.ContainsKey("age")
		respObj.ContainsKey("tips")
		respObj.ContainsKey("address")
		respObj.ContainsKey("array_inline")
		t.Log("获取单条" + uid)
	})

	t.Run("获取单条", func(t *testing.T) {
		e.GET(prefix + "/" + uid).Expect().Status(iris.StatusOK).JSON().Object().ContainsKey("update_at")
	})

	t.Run("获取所有", func(t *testing.T) {
		resp := e.GET(prefix).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey("page")
		respObj.ContainsKey("page_size")
		respObj.ContainsKey("data")
		respObj.ContainsKey("filters")
		respObj.Value("data").Array().NotEmpty()
	})

	t.Run("删除", func(t *testing.T) {
		e.DELETE(prefix + "/" + uid).Expect().Status(iris.StatusOK).Body().IsEqual("ok")
	})

	t.Run("再次访问应该为空", func(t *testing.T) {
		e.GET(prefix + "/" + uid).Expect().Status(iris.StatusBadRequest)
	})

	t.Run("获取配置", func(t *testing.T) {
		resp := e.GET(prefix + "/config").Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey("schema")
		respObj.ContainsKey("actions")
	})

}

func TestMapper(t *testing.T) {
	type sm struct {
		Key    string
		Length int
	}
	type testCase struct {
		name    string
		query   map[string]any
		targets []sm
	}
	var testCases = []testCase{
		{
			// and
			name: "and测试",
			query: map[string]any{
				"name":               "名称",
				"code":               100,
				"age":                10,
				"test_inline__music": "音乐",
			},
			targets: []sm{
				{
					Key:    "and",
					Length: 4,
				},
			},
		},
		{
			// or
			name: "or测试",
			query: map[string]any{
				"_o_age":                   10,
				"_o_desc":                  "描述",
				"_o_normal_inline__inline": "内联1",
			},
			targets: []sm{
				{
					Key:    "or",
					Length: 3,
				},
			},
		},
		{
			// 操作符
			name: "操作符测试",
			query: map[string]any{
				"age_eq":                    10,      // 1
				"address__position_eq":      "地址的位置", // 1
				"code_gt":                   90,      // 1
				"code_gte":                  100,     // 1
				"normal_inline__integer_lt": 150,     // 1
				"normal_inline__number_lte": 1,       // 1
				"desc_ne":                   "不等于",   // 1
				"tips_in":                   "提示1",   // 1
				"tj8_nin":                   "20,30",
			},
			targets: []sm{
				{
					Key:    "and",
					Length: 9,
				},
			},
		},
		{
			name: "搜索字符串",
			query: map[string]any{
				"_s": "名",
			},
			targets: []sm{
				{
					Key:    "search",
					Length: 2,
				},
			},
		},
		{
			// 因为存储是数字的无法进行正则匹配 所以不会匹配age
			name: "搜索数字",
			query: map[string]any{
				"_s": "1",
			},
			targets: []sm{
				{
					Key:    "search",
					Length: 2,
				},
			},
		},
		{
			name: "geo解析",
			query: map[string]any{
				"_g":    "103.83797,1.46103",
				"_gmax": 5000,
			},
			targets: []sm{
				{
					Key: "geo",
				},
			},
		},
		{
			// 外键1对1 则Unwind为true 1对多则 Unwind为false
			// 外键属于注入类 需要先定义好
			name:  "外键",
			query: map[string]any{},
			targets: []sm{
				{
					Key: "pk",
				},
			},
		},
	}

	app := iris.New()
	app.Configure(iris.WithoutBodyConsumptionOnUnmarshal)

	app.Get("/", func(ctx iris.Context) {
		resp := pipe.QueryParse.Run(ctx, nil, &pipe.QueryParseConfig{
			SearchFields: []string{"name", "age"},
			GeoKey:       "location",
			Pks: []*ut.Pk{
				{
					LocalKey:      "look_for",
					RemoteModelId: "default",
					RemoteKey:     ut.DefaultUidTag,
					Alias:         "Singer",
					EmptyReturn:   true,
					Unwind:        true,
				},
			},
		}, nil)
		if resp.Err != nil {
			IrisRespErr("", resp.Err, ctx)
			return
		}
		ctx.JSON(resp.Result)
		return
	})

	e := httptest.New(t, app)

	for _, tt := range testCases {
		resp := e.GET("/").WithQueryObject(tt.query).Expect().Status(iris.StatusOK)
		rawStr := resp.Body().Raw()
		var mt = new(ut.QueryFull)
		_ = json.Unmarshal([]byte(rawStr), &mt)

		for _, target := range tt.targets {
			switch target.Key {
			case "and":
				if len(mt.And) != target.Length {
					t.Fatalf("%s 需要%d个返回 得到%v个", tt.name, len(mt.And), target.Length)
				}
			case "or":
				if len(mt.Or) != target.Length {
					t.Fatalf("%s 需要%d个返回 得到%v个", tt.name, len(mt.And), target.Length)
				}
			case "search":
				if len(mt.Or) != target.Length {
					t.Fatalf("%s 需要%d个返回 得到%v个", tt.name, len(mt.And), target.Length)
				}
			case "geo":
				if mt.Geos.Field != "location" {
					t.Fatalf("%s 解析geoLocation失败", tt.name)
				}
			case "pk":
				if len(mt.Pks) < 1 {
					t.Fatalf("%s 解析geoLocation失败", tt.name)
				}

			}
		}

	}

}
