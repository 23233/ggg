package pmb

import (
	"context"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestNewBackend(t *testing.T) {
	app := iris.New()
	app.Configure(iris.WithoutBodyConsumptionOnUnmarshal)

	bk := NewBackend()
	bk.AddDb(getMg())
	bk.AddRdb(getRdb())
	bk.AddRbacUseUri(getRdbInfo())
	bk.RegistryRoute(app)
	bk.RegistryLoginRegRoute(app, true)

	model := bk.AddModelAny(new(testModelStruct))

	UserInstance.SetConn(bk.CloneConn())
	_ = UserInstance.SyncIndex(context.TODO())

	var userId string
	var token string

	loginBody := map[string]any{
		"user_name": "test123",
		"password":  "test321321",
	}

	e := httptest.New(t, app)

	t.Run("注册用户", func(t *testing.T) {
		resp := e.POST("/reg").WithJSON(loginBody).Expect().Status(http.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey("token")
		respObj.ContainsKey("info")
		userId = respObj.Value("info").Object().Value(ut.DefaultUidTag).String().Raw()
		token = respObj.Value("token").String().Raw()
	})
	t.Run("用户登录", func(t *testing.T) {
		t.Run("登录失败", func(t *testing.T) {
			body := map[string]any{
				"user_name": "test123",
				"password":  "test321123",
			}
			e.POST("/login").WithJSON(body).Expect().Status(http.StatusBadRequest)
		})
		t.Run("登录成功", func(t *testing.T) {
			e.POST("/login").WithJSON(loginBody).Expect().Status(http.StatusOK)
		})
	})

	t.Run("设置为root", func(t *testing.T) {
		err := UserInstance.SetRoleUseUserName(context.TODO(), loginBody["user_name"].(string), "root")
		assert.Equal(t, nil, err)
	})

	prefix := "/" + model.EngName
	var uid string

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	t.Run("未携带请求头抛出错误", func(t *testing.T) {
		e.GET(prefix).Expect().Status(http.StatusUnauthorized)
	})

	t.Run("新增数据", func(t *testing.T) {
		fullPost := createFullPostJson("")
		resp := e.POST(prefix).WithHeaders(headers).WithJSON(fullPost).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey(ut.DefaultUidTag)
		respObj.ContainsKey("_id")
		respObj.ContainsKey("update_at")
		respObj.ContainsKey("create_at")
		respObj.ContainsSubset(fullPost)
		uid = respObj.Value(ut.DefaultUidTag).String().Raw()
	})

	t.Run("修改数据", func(t *testing.T) {
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
				"dontShow": "不应该出现?", // 依现在的结构来说 这会出现
			},
			"array_inline": []map[string]any{inlineMap, inlineMap2},
		}
		resp := e.PUT(prefix + "/" + uid).WithHeaders(headers).WithJSON(fullPut).Expect().Status(iris.StatusOK)
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
		e.GET(prefix + "/" + uid).WithHeaders(headers).Expect().Status(iris.StatusOK).JSON().Object().ContainsKey("update_at")

	})
	t.Run("获取所有", func(t *testing.T) {
		resp := e.GET(prefix).WithHeaders(headers).Expect().Status(iris.StatusOK)
		respObj := resp.JSON().Object()
		respObj.ContainsKey("page")
		respObj.ContainsKey("page_size")
		respObj.ContainsKey("data")
		respObj.ContainsKey("filters")
		respObj.Value("data").Array().NotEmpty()
	})
	t.Run("删除单条", func(t *testing.T) {
		e.DELETE(prefix + "/" + uid).WithHeaders(headers).Expect().Status(iris.StatusOK).Body().IsEqual("ok")
	})

	t.Run("再次访问应该为空", func(t *testing.T) {
		e.GET(prefix + "/" + uid).WithHeaders(headers).Expect().Status(iris.StatusBadRequest)
	})

	t.Run("获取所有模型", func(t *testing.T) {
		resp := e.GET("/models").WithHeaders(headers).Expect().Status(iris.StatusOK)
		resp.JSON().Object().Value("models").Array().NotEmpty()
	})

	t.Run("获取自身信息", func(t *testing.T) {
		resp := e.GET("/self").WithHeaders(headers).Expect().Status(iris.StatusOK)
		resp.JSON().Object().Value("info").Object().Value(ut.DefaultUidTag).String().IsEqual(userId)
	})

	t.Run("移除用户权限", func(t *testing.T) {
		err := UserInstance.rbac.DelRoot(userId)
		assert.Equal(t, nil, err)
	})

	t.Run("再次获取应该权限错误", func(t *testing.T) {
		e.GET(prefix + "/" + uid).WithHeaders(headers).Expect().Status(iris.StatusMethodNotAllowed)
	})

	t.Run("删除用户", func(t *testing.T) {
		err := UserInstance.RemoveUser(context.TODO(), userId)
		assert.Equal(t, nil, err)
	})
}
