package smab

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"io/ioutil"
	"testing"
)

type colorModel struct {
	DefaultField `bson:",inline,flatten"`
	Name         string   `json:"name" bson:"name"`
	Desc         string   `json:"desc" bson:"desc"`
	Bll          []string `json:"bll" bson:"bll"`
}

func TestNew(t *testing.T) {

	app := iris.New()

	adminer, err := New(getTestDb(), Configs{
		App: app,
		ModelList: []interface{}{
			new(colorModel),
		},
		CasbinConfig: CasbinConfigDefine{
			Uri: getEnv("mongo_uri", ""),
		},
	})
	if err != nil {
		t.Error(err)
		return
	}

	e := httptest.New(t, app)

	e.GET(adminer.config.Prefix + "/config").Expect().Status(httptest.StatusOK).Body().Contains("window.smab")

	// 测试文件加载
	fileReq := e.GET("/smab_static/umi.js").Expect().Status(httptest.StatusOK)
	t.Logf("静态内嵌文件加载测试:%s", fileReq.Raw().Status)

	// 进行登录
	var password = "123456789"
	pb, err := ioutil.ReadFile("admin_init_password.txt")
	if err == nil {
		password = string(pb)
	}
	username := "root"

	prefix := adminer.config.Prefix + "/v"

	loginMap := map[string]interface{}{
		"user_name": username,
		"password":  password,
	}
	loginReq := e.POST(adminer.config.Prefix + "/login").WithJSON(loginMap).Expect().Status(httptest.StatusOK)
	loginReq.JSON().Object().ContainsKey("token")

	token := loginReq.JSON().Object().Value("token").String().Raw()
	headerMap := map[string]string{
		"Authorization": "Bearer " + token,
	}

	// 获取当前用户信息
	e.GET(prefix + "/user_info").WithHeaders(headerMap).Expect().Status(httptest.StatusOK).JSON().Object().ContainsKey("policy")

	adduserMap := map[string]interface{}{
		"name":     "123123",
		"password": "321321321",
		"permissions": []permissions{
			{
				Scope:  "site",
				Action: "login",
			},
		},
	}

	e.GET(prefix + "/self_users").WithHeaders(headerMap).Expect().Status(httptest.StatusOK)
	e.POST(prefix + "/self_users").WithHeaders(headerMap).WithJSON(adduserMap).Expect().Status(httptest.StatusOK)
}
