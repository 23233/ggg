package contentSafe

import (
	"errors"
	"fmt"
	"github.com/23233/ggg/logger"
	"github.com/23233/ggg/sv"
	"github.com/23233/ggg/ut"
	"github.com/imroc/req/v3"
	"github.com/kataras/iris/v12"
)

var GetTokenFunc = func() string {
	return ""
}

type WordValidReq struct {
	Content string `json:"content" comment:"内容" validate:"required,max=2000"` // 尽量不要有长字符
}

type WordHitResp struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type WxTextV1Resp struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
	TraceId string `json:"trace_id"`
}

func HitText(ctx iris.Context) {
	q := ctx.Values().Get(sv.GlobalContextKey).(*WordValidReq)

	success, message := AutoHitText(q.Content)

	_ = ctx.JSON(WordHitResp{
		Success: success, // 为true则是安全文本
		Msg:     message,
	})
}

func AutoHitText(content string) (bool, string) {
	var success = true
	var message = "ok"
	if len(content) > 200 && ut.RandomInt(1, 5) < 2 {
		// 长文本 20%概率使用接口校验
		pass, err, msg := wxTextCheckV1(content)
		if err != nil {
			logger.J.Error("校验文本失败", logger.J.WithError(err))
		}
		success = pass
		message = msg
	} else {
		pass, msg := hitText(content)
		success = pass
		message = msg
	}
	return success, message
}

func hitText(content string) (bool, string) {

	// 去掉所有标点符号
	clear := ClearText(content)

	if len(clear) >= 1 {
		if LadClient.Match(clear) {
			return false, "中文有不良词汇,请修改"
		}
	}
	return true, "ok"
}

func wxTextCheckV1(content string) (bool, error, string) {
	// https://developers.weixin.qq.com/miniprogram/dev/framework/security.imgSecCheck.html
	uri := "https://api.weixin.qq.com/wxa/msg_sec_check"
	query := map[string]string{
		"access_token": GetTokenFunc(),
	}
	body := map[string]string{
		"content": content,
	}
	var defaultNotMsg = "请求异常,请稍后重试"
	resp, err := req.R().SetQueryParams(query).SetBodyJsonMarshal(body).Post(uri)
	if err != nil {
		return false, err, defaultNotMsg
	}
	if resp.StatusCode != 200 {
		msg, err := resp.ToString()
		if err != nil {
			return false, err, defaultNotMsg
		}
		return false, errors.New(msg), defaultNotMsg
	}
	var j WxTextV1Resp
	err = resp.UnmarshalJson(&j)
	if err != nil {
		return false, err, defaultNotMsg
	}

	// 开发者可使用以上两段文本进行测试，若接口 errcode 返回87014(内容含有违法违规内容)，则对接成功。
	if j.Errcode == 87014 {
		return false, nil, "文本有不良词汇,请修改"
	}
	if j.Errcode != 0 {
		logger.J.Error(fmt.Sprintf("微信内容安全文字检测v1 返回码异常为 %d", j.Errcode))
		return false, errors.New("接口响应码异常"), "校验行为异常,请稍后重试"
	}

	return true, nil, "ok"

}

func AddWord(wordList ...string) {
	if len(wordList) >= 1 {
		logger.J.Infof("新增敏感词 %v", wordList)
		LadClient.AddOfList(wordList)
		LadClient.Build()
	}
}
