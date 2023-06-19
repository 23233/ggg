package pipe

import (
	"encoding/base64"
	"encoding/json"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/redis/rueidis"
	"testing"
	"time"
)

// 滑块验证码验证
func TestSwipeValid(t *testing.T) {
	app := iris.New()
	rdb, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{"127.0.0.1:6379"},
		Password:    "password",
		SelectDB:    4,
	})
	if err != nil {
		t.Error(err)
	}
	inst := NewSwipeValidInst(rdb)
	name := "/swipe"
	getRouter := app.Get(name, func(ctx iris.Context) {
		sp, err := inst.Gen(ctx)
		if err != nil {
			panic(err)
		}
		raw := sp.ToString()
		sEnc := base64.StdEncoding.EncodeToString([]byte(raw))
		ctx.WriteString(sEnc)
	})
	postRouter := app.Post(name, func(ctx iris.Context) {
		var raw string
		body, err := ctx.GetBody()
		if err != nil {
			panic(err)
		}
		raw = string(body)
		check, err := inst.Check(ctx, raw)
		if err != nil {
			ctx.StopWithJSON(400, iris.Map{"detail": err.Error()})
			return
		}

		ctx.JSON(check)
	})

	e := httptest.New(t, app)

	resp := e.GET(getRouter.Path).Expect().Status(iris.StatusOK)
	b64Str := resp.Body().Raw()
	rawStr, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		t.Fatal(err)
	}

	var rawItem = new(SwipeItem)
	err = rawItem.ParseStr(string(rawStr))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("获取到参数 ", rawItem.Id)
	body := map[string]any{
		"sid":           rawItem.Id,
		"refresh_count": 0,
		"x":             int64(rawItem.X - (rawItem.B / 2) - 10),
		"y":             int64(rawItem.Y - (rawItem.B / 2) - 10),
		"t":             time.Now(),
		"te":            time.Now().Add(300 * time.Millisecond),
		"s":             1.64,
		"n":             0.9,
		"nm":            0.3,
	}
	req := e.POST(postRouter.Path)

	// 如果存在 则需要在请求包中传递
	if rawItem.N {
		for i := 0; i < int(rawItem.C); i++ {
			body[rawItem.P+"-"+ut.RandomStr(3)] = ut.RandomStr(6)
		}
	} else {
		// 需要在请求头中传递
		for i := 0; i < int(rawItem.C); i++ {
			req.WithHeader(rawItem.P+"-"+ut.RandomStr(3), ut.RandomStr(6))
		}
	}

	rawPost, _ := json.Marshal(&body)
	// base64加密
	enc := base64.StdEncoding.EncodeToString(rawPost)

	resp = req.WithBytes([]byte(enc)).Expect().Status(iris.StatusOK)
	resp.JSON().Object().Value("sid").String().Equal(rawItem.Id)

}
