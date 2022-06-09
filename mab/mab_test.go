package mab

import (
	_ctx "context"
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"math/rand"
	"os"
	"testing"
	"time"
)

type colorModel struct {
	Name string   `json:"name" bson:"name"`
	Desc string   `json:"desc" bson:"desc"`
	Bll  []string `json:"bll" bson:"bll"`
}

type inlineTest struct {
	Event   string               `json:"event" bson:"event"`
	Age     int64                `json:"age" bson:"age"`
	Year    uint64               `json:"year" bson:"year"`
	Tags    []string             `json:"tags" bson:"tags"`
	Gps     []float64            `json:"gps" bson:"gps"`
	ObjId   primitive.ObjectID   `json:"obj_id" bson:"obj_id"`
	ObjList []primitive.ObjectID `json:"obj_list" bson:"obj_list"`
}

type testModel struct {
	DefaultField `bson:",inline,flatten"`
	Name         string    `bson:"name" json:"name"`
	Age          uint64    `bson:"age" json:"age"`
	Desc         string    `bson:"desc" json:"desc"`
	Code         uint64    `bson:"code" json:"code"`
	Tips         []string  `bson:"tips" json:"tips"`
	Tj           []int     `json:"tj" bson:"tj"`
	Tj2          []int8    `json:"tj2" bson:"tj2"`
	Tj3          []int16   `json:"tj3" bson:"tj3"`
	Tj4          []int32   `json:"tj4" bson:"tj4"`
	Tj5          []int64   `json:"tj5" bson:"tj5"`
	Tj6          []uint    `json:"tj6" bson:"tj6"`
	Tj7          []uint64  `json:"tj7" bson:"tj7"` // uint8 似乎无法正常获取
	Tj8          []uint16  `json:"tj8" bson:"tj8"`
	Tj9          []uint32  `json:"tj9" bson:"tj9"`
	Tj10         []uint64  `json:"tj10" bson:"tj10"`
	Tf           []float32 `json:"tf" bson:"tf"`
	Tf2          []float64 `json:"tf2" bson:"tf2"`
	Address      struct {
		Position string `bson:"position" json:"position"`
	} `bson:"address" json:"address"`
	TestInline struct {
		Music string `bson:"music" json:"music"`
	} `json:"test_inline" bson:"test_inline,m_inline"`
	Location MongoLocation `bson:"location" json:"location"`
	Colors   []colorModel  `json:"colors" bson:"colors"`
	Inline   inlineTest    `json:"inline" bson:"inline"`
}

func (c *testModel) Alias() string {
	return "_组名_表名"
}

type testLocation struct {
	DefaultField `bson:",inline,flatten"`
	Location     MongoLocation `bson:"location" json:"location"`
	Name         string        `json:"name" bson:"name"`
}

type testInline struct {
	DefaultField `bson:",inline,flatten"`
	Colors       []colorModel `json:"colors" bson:"colors"`
	NotInline    inlineTest   `json:"not_inline" bson:"not_inline"`
	Inline       inlineTest   `json:"inline" bson:"inline"`
}

type testFk struct {
	DefaultField `bson:",inline,flatten"`
	Name         string               `bson:"name" json:"name"`
	LookFor      primitive.ObjectID   `json:"look_for" bson:"look_for"` // 外键
	LookUpFor    []primitive.ObjectID `json:"look_up_for" bson:"look_up_for"`
	UserId       primitive.ObjectID   `json:"user_id" bson:"user_id"`
}

type MongoLocation struct {
	GeoJSONType string    `json:"type" bson:"type,omitempty" mapstructure:"type"` // Point
	Coordinates []float64 `json:"coordinates" bson:"coordinates,omitempty"`       // lng,lat
}

func NewLocation(lat, long float64) MongoLocation {
	return MongoLocation{
		"Point",
		[]float64{long, lat},
	}
}

type DefaultField struct {
	Id       primitive.ObjectID `bson:"_id" json:"id"`
	UpdateAt time.Time          `bson:"update_at" json:"update_at"`
}

func (u *DefaultField) BeforeInsert(ctx _ctx.Context) error {
	if u.Id.IsZero() {
		u.Id = primitive.NewObjectID()
	}
	if u.UpdateAt.IsZero() {
		u.UpdateAt = time.Now().Local()
	}
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

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func randomStr(n int) string {
	randBytes := make([]byte, n/2)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}

func getDb() *qmgo.Database {
	mongoUri := getEnv("mongo_uri", "")
	ctx := _ctx.Background()
	mgCli, err := qmgo.NewClient(ctx, &qmgo.Config{Uri: mongoUri})
	if err != nil {
		panic(err)
	}
	mgDb := mgCli.Database("ttt")
	return mgDb
}

// 测试普通增删改查
func TestNormalCrud(t *testing.T) {
	t.Log("run add")
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/"
	checkMc := &Config{
		Party: app.Party(prefix),
		Mdb:   mdb,
		Models: []*SingleModel{
			{
				Model:        new(testModel),
				ShowCount:    true,
				ShowDocCount: true,
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "test_model"
	name := "赵日天"
	bodyMap := map[string]interface{}{
		"name": name,
		"age":  68,
		"desc": "desc",
		"tips": []string{"测试", "吃吃睡睡", randomStr(4)},
		"tj":   []int{123, 12},
		"tj2":  []int8{1, 2},
		"tj3":  []int16{3, 4},
		"tj4":  []int32{5, 6},
		"tj5":  []int64{7, 8},
		"tj6":  []uint{9, 10},
		"tj7":  []uint64{1, 32},
		"tj8":  []uint16{13, 14},
		"tj9":  []uint32{15, 16},
		"tj10": []uint64{17, 18},
		"tf":   []float32{12.23, 13.45},
		"tf2":  []float64{11.32, 22.56},
		"address": map[string]interface{}{
			"position": "地理位置" + randomStr(4),
		},
		"music": randomStr(4),
		"test_inline": map[string]interface{}{
			"music": "dangdangdang",
		},
		"location": map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{101.325648, 34.605063},
		},
		"colors": []map[string]interface{}{
			{
				"name": "颜色名",
				"desc": "这是个颜色",
				//"bll":[]string{"c","d"}, // 暂不支持
			},
			{
				"name": "color",
				"desc": "color desc",
				//"bll":[]string{"a","b"}, // 暂不支持
			},
		},
	}
	addData := e.POST(fp).WithJSON(bodyMap).Expect().Status(httptest.StatusOK)
	resp := addData.JSON().Object()
	resp.Value("id").NotNull()
	resp.Value("name").Equal(name)
	resp.Value("desc").Equal("desc")
	resp.Value("age").Equal(68)
	id := resp.Value("id").String().Raw()
	fs := fp + "/" + id

	// get all
	getAll := e.GET(fp).WithQueryObject(map[string]interface{}{
		"age": 68,
	}).Expect().Status(httptest.StatusOK)
	getAll.JSON().Object().ContainsKey("data").Value("data").Array().NotEmpty()
	getAll.JSON().Object().Value("data").Array().Element(0).Object().Value("test_inline").Object().Value("music").String().NotEmpty()
	t.Log("find data")
	// get single
	getSingle := e.GET(fs).Expect().Status(httptest.StatusOK)
	getSingle.JSON().Object().ContainsKey("name")
	t.Log("get single data success")

	editPosition := randomStr(4)
	editMap := map[string]interface{}{"name": "edit", "address": map[string]interface{}{"position": editPosition}}
	edit := e.PUT(fs).WithJSON(editMap).Expect().Status(httptest.StatusOK)
	edit.JSON().Object().Value("name").Equal("edit")
	edit.JSON().Object().Value("address").Object().Value("position").Equal(editPosition)
	t.Log("put data success")

	// delete data
	deleteData := e.DELETE(fs).Expect().Status(httptest.StatusOK)
	deleteData.JSON().Object().Value("id").NotNull()
	t.Log("delete data success")

	err := mdb.Collection("test_model").DropCollection(_ctx.TODO())
	if err != nil {
		t.Fatal(err, "移除Collection失败")
	}

}

// 外键增删改查
func TestFkCrud(t *testing.T) {
	t.Log("run add")
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	fkAlias := "podcast"
	checkMc := &Config{
		Party: app.Party(prefix),
		Mdb:   mdb,
		Models: []*SingleModel{

			{

				Model: new(testModel),
			},
			{

				Model: new(testFk),
				Pk: func() []bson.D {
					look := bson.D{{"$lookup", bson.D{
						{"from", "test_model"},
						{"localField", "look_for"},
						{"foreignField", "_id"},
						{"as", fkAlias},
					}}}
					return []bson.D{look}
				},
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	fkp := prefix + "/" + "test_model"
	// 新增一条外键数据
	name := "测试外键关联"
	bodyMap := map[string]interface{}{
		"name": name,
	}
	addData := e.POST(fkp).WithJSON(bodyMap).Expect().Status(httptest.StatusOK)
	resp := addData.JSON().Object()
	resp.Value("id").NotNull()
	resp.Value("name").Equal(name)
	id := resp.Value("id").String().Raw()

	// 新增数据
	contentMap := map[string]interface{}{
		"name":        name,
		"look_for":    id,
		"look_up_for": []string{id},
	}
	addData = e.POST(fp).WithJSON(contentMap).Expect().Status(httptest.StatusOK)
	addJson := addData.JSON().Object()
	addJson.Value("name").Equal(name)
	addJson.Value("look_for").Equal(id)
	addJson.Value("look_up_for").Array().NotEmpty()
	fkId := addJson.Value("id").String().Raw()
	t.Log("add fk data success")

	// 获取数据
	getAll := e.GET(fp).WithQueryObject(map[string]interface{}{
		"_o":           "name",
		"name":         name,
		"create_at_gt": "2020-12-05T13:28:49Z",
	}).Expect().Status(httptest.StatusOK)
	getAll.JSON().Object().ContainsKey("page")
	getAll.JSON().Object().Value("data").Array().Element(0).Object().ContainsKey(fkAlias)
	t.Log("get all data ")

	fs := fp + "/" + fkId
	getSingle := e.GET(fs).Expect().Status(httptest.StatusOK)
	getJson := getSingle.JSON().Object()
	getJson.ContainsKey("name")
	getJson.ContainsKey(fkAlias)
	t.Log("get single data")

	editMap := map[string]interface{}{"name": "edit4", "look_for": "607915172b17f6564db99184"}
	edit := e.PUT(fs).WithJSON(editMap).Expect().Status(httptest.StatusOK)
	edit.JSON().Object().Value("name").Equal("edit4")
	edit.JSON().Object().Value("look_for").Equal("607915172b17f6564db99184")
	t.Log("put data")

	deleteData := e.DELETE(fs).Expect().Status(httptest.StatusOK)
	deleteData.JSON().Object().ContainsKey("id")
	t.Log("delete data")
}

// 上下文传递
func TestContextAdd(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	userId := "607a92d8055b53cf0b3e89cc"
	part := app.Party(prefix)
	part.Use(func(ctx iris.Context) {
		obj, _ := primitive.ObjectIDFromHex(userId)
		ctx.Values().Set("userId", obj)
		ctx.Next()
	})
	checkMc := &Config{
		Party: part,
		Mdb:   mdb,
		Models: []*SingleModel{
			{
				PrivateContextKey: "userId",
				PrivateColName:    "user_id",
				Model:             new(testFk),
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	name := "测试私密参数传递"
	contentMap := map[string]interface{}{
		"name": name,
	}
	addData := e.POST(fp).WithJSON(contentMap).Expect().Status(httptest.StatusOK)
	resp := addData.JSON().Object()
	resp.Value("name").Equal(name)
	resp.Value("user_id").Equal(userId)
	id := resp.Value("id").String().Raw()

	e.DELETE(fp + "/" + id).Expect().Status(httptest.StatusOK)

}

// 必传参数验证
func TestMustFilter(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	checkMc := &Config{
		Party: part,
		Mdb:   mdb,
		Models: []*SingleModel{
			{
				Model: new(testFk),
				PostMustFilters: map[string]string{
					"user_id": "",
				},
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	contentMap := map[string]interface{}{
		"name": "不传用户ID",
	}
	e.POST(fp).WithJSON(contentMap).Expect().Status(httptest.StatusBadRequest)
	userId := "607a92d8055b53cf0b3e89cc"
	successMap := map[string]interface{}{
		"user_id": userId,
	}
	successData := e.POST(fp).WithJSON(successMap).Expect().Status(httptest.StatusOK)
	successData.JSON().Object().Value("user_id").Equal(userId)
	id := successData.JSON().Object().Value("id").String().Raw()

	e.DELETE(fp + "/" + id).Expect().Status(httptest.StatusOK)

	t.Log("must filter success")
}

// 测试敏感词
func TestSensitive(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	name := "赵日天"
	checkMc := &Config{
		SensitiveWords: []string{"赵日天", "天日照"},
		Party:          part,
		Mdb:            mdb,
		Models: []*SingleModel{
			{
				Model: new(testFk),

				SensitiveFields: []string{"name"},
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	contentMap := map[string]interface{}{
		"name": name,
	}
	successMap := map[string]interface{}{
		"name": "能通过",
	}
	e.POST(fp).WithJSON(contentMap).Expect().Status(httptest.StatusBadRequest)

	successData := e.POST(fp).WithJSON(successMap).Expect().Status(httptest.StatusOK)
	id := successData.JSON().Object().Value("id").String().Raw()

	e.DELETE(fp + "/" + id).Expect().Status(httptest.StatusOK)

	t.Log("sensitive success")

}

// 测试生成器模式
func TestGenerator(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	checkMc := &Config{
		Generator: true,
		Party:     part,
		Mdb:       mdb,
		Models: []*SingleModel{
			{
				Model: new(testFk),
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	name := "测试生成器模式"
	bodyMap := map[string]interface{}{
		"name": name,
	}
	// 错误的url
	e.GET(fp + "rr").Expect().Status(httptest.StatusBadRequest)
	// 正确的
	successData := e.POST(fp).WithJSON(bodyMap).Expect().Status(httptest.StatusOK)
	id := successData.JSON().Object().Value("id").String().Raw()

	e.GET(fp).Expect().Status(httptest.StatusOK)

	single := fp + "/" + id
	e.GET(single).Expect().Status(httptest.StatusOK)
	editMap := map[string]interface{}{
		"name": "修改后",
	}
	e.PUT(single).WithJSON(editMap).Expect().Status(httptest.StatusOK).JSON().Object().ContainsKey("name")
	e.DELETE(single).Expect().Status(httptest.StatusOK)
}

// 测试MustSearch搜索
func TestSearch(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	checkMc := &Config{
		Generator: true,
		Party:     part,
		Mdb:       mdb,
		Models: []*SingleModel{
			{
				Model: new(testFk),

				MustSearch: true,
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	bodyMap := map[string]interface{}{
		"name": "新增测试搜索",
	}
	successData := e.POST(fp).WithJSON(bodyMap).Expect().Status(httptest.StatusOK)
	id := successData.JSON().Object().Value("id").String().Raw()
	// 进行搜索
	searchParams := map[string]interface{}{
		"_s": "__" + id + "__",
	}

	result := e.GET(fp).WithQueryObject(searchParams).Expect().Status(httptest.StatusOK)
	result.JSON().Object().ContainsKey("data").Value("data").Array().NotEmpty()

	// 二次搜索
	searchParams["_s"] = "__新增测试__"
	result = e.GET(fp).WithQueryObject(searchParams).Expect().Status(httptest.StatusOK)
	result.JSON().Object().ContainsKey("data").Value("data").Array().NotEmpty()

	single := fp + "/" + id
	e.DELETE(single).Expect().Status(httptest.StatusOK)
}

// 测试指定字段搜索
func TestFieldsSearch(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	checkMc := &Config{
		Generator: true,
		Party:     part,
		Mdb:       mdb,
		Models: []*SingleModel{
			{
				Model: new(testFk),

				AllowSearchFields: []string{"Name"},
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	bodyMap := map[string]interface{}{
		"name": "新增测试搜索",
	}
	successData := e.POST(fp).WithJSON(bodyMap).Expect().Status(httptest.StatusOK)
	id := successData.JSON().Object().Value("id").String().Raw()
	// 进行搜索
	searchParams := map[string]interface{}{
		"_s": "__新增测试__",
	}

	result := e.GET(fp).WithQueryObject(searchParams).Expect().Status(httptest.StatusOK)
	result.JSON().Object().ContainsKey("data").Value("data").Array().NotEmpty()

	single := fp + "/" + id
	e.DELETE(single).Expect().Status(httptest.StatusOK)
}

// 测试查询参数 生成器模式
func TestGetAllFilter(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	checkMc := &Config{
		Generator: true,
		Party:     part,
		Mdb:       mdb,
		Models: []*SingleModel{
			{
				Model: new(testFk),
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	bodyMap := map[string]interface{}{
		"name":        "新增测试搜索",
		"look_up_for": []string{"61b561dd5b370e3b42635dc7", "61b561dd5b370e3b42635dc8", "61b561dd5b370e3b42635dc9"},
	}
	successData := e.POST(fp).WithJSON(bodyMap).Expect().Status(httptest.StatusOK)
	id := successData.JSON().Object().Value("id").String().Raw()

	// 通过_id进行查询
	paramsData := map[string]interface{}{
		//"_id":             id,
		"look_up_for_in_": "61b561dd5b370e3b42635dc9",
		//"name": "新增测试搜索",
	}

	filterReq := e.GET(fp).WithQueryObject(paramsData).Expect().Status(httptest.StatusOK)
	filterReq.JSON().Object().Value("data").Array().NotEmpty()

	single := fp + "/" + id
	e.DELETE(single).Expect().Status(httptest.StatusOK)

}

// 测试缓存
func TestCache(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	checkMc := &Config{
		Generator: true,
		Party:     part,
		Mdb:       mdb,
		Models: []*SingleModel{
			{
				Model: new(testFk),

				CacheTime: 5 * time.Second,
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + "test_fk"
	bodyMap := map[string]interface{}{
		"name": "新增测试缓存",
	}
	successData := e.POST(fp).WithJSON(bodyMap).Expect().Status(httptest.StatusOK)
	id := successData.JSON().Object().Value("id").String().Raw()
	single := fp + "/" + id
	// 获取列表的缓存
	paramsData := map[string]interface{}{
		"name": "新增测试缓存",
	}
	filterReq := e.GET(fp).WithQueryObject(paramsData).Expect().Status(httptest.StatusOK)
	filterReq.JSON().Object().Value("data").Array().NotEmpty()
	// 二次获取
	filterReq = e.GET(fp).WithQueryObject(paramsData).Expect().Status(httptest.StatusOK)
	filterReq.Header("is_cache").Equal("1")
	// 缓存失效
	time.Sleep(5 * time.Second)
	filterReq = e.GET(fp).WithQueryObject(paramsData).Expect().Status(httptest.StatusOK)
	filterReq.Header("is_cache").Empty()

	// 单个获取
	e.GET(single).Expect().Status(httptest.StatusOK)
	// 再次获取有缓存
	e.GET(single).Expect().Status(httptest.StatusOK).Header("is_cache").Equal("1")
	// 等待5秒
	time.Sleep(5 * time.Second)
	// 获取后无缓存
	e.GET(single).Expect().Status(httptest.StatusOK).Header("is_cache").Empty()

	// 删除这条数据
	e.DELETE(single).Expect().Status(httptest.StatusOK)

}

// 测试geo
func TestGeo(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	checkMc := &Config{
		Party: part,
		Mdb:   mdb,
		Models: []*SingleModel{
			{
				Model: new(testLocation),
			},
		},
	}

	name := "test_location"

	// 创建索引
	cli, _ := mdb.Collection(name).CloneCollection()
	_, err := cli.Indexes().CreateOne(_ctx.TODO(), mongo.IndexModel{
		Keys: bson.M{"location": "2dsphere"},
	})
	if err != nil {
		t.Fatal(err, "创建location 2dsphere索引失败")
	}
	// 插入数据
	var dataList = make([]*testLocation, 0)
	dataList = append(dataList, &testLocation{
		DefaultField: DefaultField{
			Id:       primitive.NewObjectID(),
			UpdateAt: time.Now(),
		},
		Location: NewLocation(45.771562, 131.009219),
		Name:     "黑龙江省七台河市旭日街",
	})
	dataList = append(dataList, &testLocation{
		DefaultField: DefaultField{
			Id:       primitive.NewObjectID(),
			UpdateAt: time.Now().Add(10 * time.Second),
		},
		Location: NewLocation(29.580466, 106.527473),
		Name:     "重庆市重庆市观音桥茂业百货",
	})
	dataList = append(dataList, &testLocation{
		DefaultField: DefaultField{
			Id:       primitive.NewObjectID(),
			UpdateAt: time.Now().Add(20 * time.Second),
		},
		Location: NewLocation(29.579253, 106.5288),
		Name:     "重庆市重庆市红鼎国际C座",
	})
	dataList = append(dataList, &testLocation{
		DefaultField: DefaultField{
			Id:       primitive.NewObjectID(),
			UpdateAt: time.Now().Add(30 * time.Second),
		},
		Location: NewLocation(29.589502, 106.53086),
		Name:     "重庆市重庆市洋河北路6号力帆体育场南粤银行",
	})
	dataList = append(dataList, &testLocation{
		DefaultField: DefaultField{
			Id:       primitive.NewObjectID(),
			UpdateAt: time.Now().Add(40 * time.Second),
		},
		Location: NewLocation(29.57316, 106.5298),
		Name:     "重庆市重庆市观音桥cosmoA栋",
	})

	_, err = mdb.Collection(name).InsertMany(_ctx.TODO(), dataList)
	if err != nil {
		t.Fatal("生成基础数据失败")
		return
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/" + name

	paramsData := map[string]interface{}{
		"_g":    "106.527473,29.580466",
		"_gmax": 5000,
		"_gmin": 1,
		//"name":"重庆市重庆市观音桥cosmoA栋",
		//"_od":   "update_at", // 按距离排序是最佳的选择 不要擅自再进行排序 geo就是给位置准备的
	}
	filterReq := e.GET(fp).WithQueryObject(paramsData).Expect().Status(httptest.StatusOK)
	filterReq.JSON().Object().Value("data").Array().NotEmpty()

	err = mdb.Collection(name).DropCollection(_ctx.TODO())
	if err != nil {
		t.Fatal(err, "移除Collection失败")
	}
}

// 测试操作符
func TestOp(t *testing.T) {
	mdb := getDb()
	app := iris.New()
	iris.WithoutBodyConsumptionOnUnmarshal(app)
	prefix := "/api/v1"
	part := app.Party(prefix)
	checkMc := &Config{
		Party: part,
		Mdb:   mdb,
		Models: []*SingleModel{
			{
				Model: new(testModel),
			},
		},
	}
	New(checkMc)
	e := httptest.New(t, app)
	fp := prefix + "/test_model"

	bodyMap := map[string]interface{}{
		"age":  68,
		"desc": "desc",
	}
	addData := e.POST(fp).WithJSON(bodyMap).Expect().Status(httptest.StatusOK)
	resp := addData.JSON().Object()
	resp.Value("id").NotNull()
	resp.Value("desc").Equal("desc")
	resp.Value("age").Equal(68)
	id := resp.Value("id").String().Raw()
	fs := fp + "/" + id

	getAll := e.GET(fp).WithQueryObject(map[string]interface{}{
		"age_gte_": 68,
	}).Expect().Status(httptest.StatusOK)
	getAll.JSON().Object().ContainsKey("data").Value("data").Array().NotEmpty()
	existsGet := e.GET(fp).WithQueryObject(map[string]any{
		"name_exists_": true,
	}).Expect().Status(httptest.StatusOK)
	existsGet.JSON().Object().ContainsKey("data").Value("data").Array().NotEmpty()

	nullGet := e.GET(fp).WithQueryObject(map[string]any{
		"name_null_": true,
	}).Expect().Status(httptest.StatusOK)
	nullGet.JSON().Object().ContainsKey("data").Value("data").Array().NotEmpty()

	deleteData := e.DELETE(fs).Expect().Status(httptest.StatusOK)
	deleteData.JSON().Object().Value("id").NotNull()

}

func TestAll(t *testing.T) {
	t.Run("getAllFilter", TestGetAllFilter)
	t.Run("normalCurd", TestNormalCrud)
	t.Run("fkCurd", TestFkCrud)
	t.Run("privateContext", TestContextAdd)
	t.Run("mustFilter", TestMustFilter)
	t.Run("sensitive", TestSensitive)
	t.Run("generator", TestGenerator)
	t.Run("search", TestSearch)
	t.Run("fields_search", TestFieldsSearch)
	t.Run("cache", TestCache)
	t.Run("geo", TestGeo)
	t.Run("op", TestOp)
}
