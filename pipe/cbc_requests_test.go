package pipe

import (
	"encoding/json"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"testing"
)

type TestPayload struct {
	Message string `json:"message"`
}

func TestCbcService(t *testing.T) {
	app := iris.New()
	cbcService := NewCbcService([]byte("mysecretpassword"))

	CbcRegisterHandler("/test1", func(ctx iris.Context, query map[string]interface{}, body TestPayload) {
		ctx.JSON(body)
	})

	CbcRegisterHandler("/test2", func(ctx iris.Context, query map[string]interface{}, body *TestPayload) {
		ctx.JSON(body)
	})

	app.Post("/cbc", cbcService.CbcMainHandler)

	testApp := httptest.New(t, app)

	// Test with registered router and non-empty body and query
	reqData := CbcRequestData{
		Query:  map[string]interface{}{"key": "value"},
		Body:   json.RawMessage(`{"message":"hello"}`),
		Router: "/test1",
	}
	reqDataBytes, _ := json.Marshal(reqData)
	encrypted, _ := cbcService.Encrypt(reqDataBytes)
	resp := testApp.POST("/cbc").WithText(encrypted).Expect().Status(iris.StatusOK)
	resp.JSON().Object().ValueEqual("message", "hello")
	//
	// Test with unregistered router
	reqData.Router = "/test-unregistered"
	reqDataBytes, _ = json.Marshal(reqData)
	encrypted, _ = cbcService.Encrypt(reqDataBytes)
	testApp.POST("/cbc").WithText(encrypted).Expect().Status(iris.StatusBadRequest)
	//
	// Test with empty query
	reqData.Query = nil
	reqData.Router = "/test1"
	reqDataBytes, _ = json.Marshal(reqData)
	encrypted, _ = cbcService.Encrypt(reqDataBytes)
	resp = testApp.POST("/cbc").WithText(encrypted).Expect().Status(iris.StatusOK)
	resp.JSON().Object().ValueEqual("message", "hello")
	//
	// Test with empty body
	reqData.Body = json.RawMessage("{}")
	reqDataBytes, _ = json.Marshal(reqData)
	encrypted, _ = cbcService.Encrypt(reqDataBytes)
	resp = testApp.POST("/cbc").WithText(encrypted).Expect().Status(iris.StatusOK)
	resp.JSON().Object().ValueEqual("message", "")

	// Test with nil body
	reqData.Router = "/test2"
	reqDataBytes, _ = json.Marshal(reqData)
	encrypted, _ = cbcService.Encrypt(reqDataBytes)
	resp = testApp.POST("/cbc").WithText(encrypted).Expect().Status(iris.StatusOK)
	resp.JSON().Object().ValueEqual("message", "")
}
