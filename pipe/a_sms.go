package pipe

import (
	"context"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
)

// 短信相关

type SmsPipe struct {
	Mobile     string `json:"mobile,omitempty" bson:"mobile,omitempty"`
	TemplateId string `json:"template_id,omitempty" bson:"template_id,omitempty"`
	Code       string `json:"code,omitempty" bson:"code,omitempty"`
}

var (
	// SmsSend 短信发送
	// 必传params SmsPipe
	// 必传db SmsClient 的实例
	SmsSend = &RunnerContext[any, *SmsPipe, *SmsClient, string]{
		Name: "短信验证码发送",
		Key:  "sms_send",
		call: func(ctx iris.Context, origin any, params *SmsPipe, db *SmsClient, more ...any) *RunResp[string] {
			code, err := db.SendBeforeCheck(ctx, params.TemplateId, params.Mobile)
			return NewPipeResultErr(code, err)
		},
	}
	// SmsValid 短信验证码验证
	// 必传params SmsPipe
	// 必传db SmsClient 的实例
	SmsValid = &RunnerContext[any, *SmsPipe, *SmsClient, bool]{
		Name: "短信验证码验证",
		Key:  "sms_valid",
		call: func(ctx iris.Context, origin any, params *SmsPipe, db *SmsClient, more ...any) *RunResp[bool] {

			pass := db.Valid(ctx, params.Mobile, params.Code)
			if !pass {
				return NewPipeErr[bool](errors.New("短信验证码验证失败"))

			}
			db.DelKey(context.TODO(), params.Mobile)

			return NewPipeResult(pass)
		},
	}
)
