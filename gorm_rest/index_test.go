package gorm_rest

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"strconv"
	"testing"
	"time"
)

type Model struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
type User struct {
	Model
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to the database: %v", err)
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("Failed to migrate the database: %v", err)
	}

	return db
}

func TestNewGormSchemaRest(t *testing.T) {
	db := setupTestDB(t)
	bodyForm := map[string]any{
		"name":  "Alice",
		"age":   25,
		"email": "alice@example.com",
	}
	editForm := map[string]any{
		"name":  "Alice_edit",
		"age":   52,
		"email": "alice_edit@example.com",
	}
	app := iris.New()
	app.Configure(iris.WithoutBodyConsumptionOnUnmarshal)

	testParty := app.Party("/test")
	inst := NewGormSchemaRest(User{}, db)
	inst.PutHandlerConfig = GormModelPutConfig{
		UpdateTime: true,
	}
	inst.WriteInsert = true

	inst.Registry(testParty)

	e := httptest.New(t, app)

	var uid string

	t.Run("测试新增", func(t *testing.T) {
		resp := e.POST(testParty.GetRelPath()).WithJSON(bodyForm).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey(GormIdKey)
		respObj.ContainsKey("updated_at")
		respObj.ContainsKey("created_at")
		respObj.ContainsSubset(bodyForm)
		uid = strconv.Itoa(int(respObj.Value(GormIdKey).Number().Raw()))
	})

	t.Run("修改", func(t *testing.T) {
		resp := e.PUT(testParty.GetRelPath() + "/" + uid).WithJSON(editForm).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey("name")
		respObj.ContainsKey("age")
		respObj.ContainsKey("email")
		respObj.ContainsKey("updated_at")
		t.Log("修改单条" + uid)
	})

	t.Run("获取单条", func(t *testing.T) {
		resp := e.GET(testParty.GetRelPath() + "/" + uid).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.Value("name").IsEqual(editForm["name"])
		respObj.Value("age").IsEqual(editForm["age"])
		respObj.Value("email").IsEqual(editForm["email"])
	})

	t.Run("获取所有", func(t *testing.T) {
		resp := e.GET(testParty.GetRelPath()).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey("page")
		respObj.ContainsKey("page_size")
		respObj.ContainsKey("data")
		respObj.ContainsKey("filters")
		respObj.Value("data").Array().NotEmpty()
	})

	t.Run("删除", func(t *testing.T) {
		e.DELETE(testParty.GetRelPath() + "/" + uid).Expect().Status(iris.StatusOK).Body().IsEqual("ok")
	})

	t.Run("再次访问应该为空", func(t *testing.T) {
		e.GET(testParty.GetRelPath() + "/" + uid).Expect().Status(iris.StatusBadRequest)
	})

}
