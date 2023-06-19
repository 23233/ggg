package pipe

import (
	"context"
	"encoding/json"
	orginErrors "errors"
	"fmt"
	"github.com/23233/ggg/logger"
	"github.com/23233/ggg/ut"
	"github.com/go-redis/redis/v8"
	"github.com/redis/rueidis"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20190711"
	"strconv"
	"strings"
	"time"
)

// 短信相关

type SmsClient struct {
	secretId    string
	secretKey   string
	sign        string
	appId       string
	region      string
	expTime     time.Duration // 过期时间 默认5分钟
	redisPrefix string
	rdb         rueidis.Client
}

func NewSmsClient(secretId, secretKey, sign, appId string, rdb rueidis.Client) *SmsClient {
	var client = new(SmsClient)
	client.secretKey = secretKey
	client.secretId = secretId
	client.sign = sign
	client.appId = appId
	client.expTime = 5 * time.Minute
	client.redisPrefix = "code:"
	client.rdb = rdb
	return client
}

func NewDefaultSmsClient(rdb rueidis.Client) *SmsClient {
	client := NewSmsClient("AKID3tPl04qN4btLZZMn1HRRAd17wVrhCFIF", "sUrhGYbYOxFJBqGQbaIjurZ0AB0YqtmR", "星耀九州", "1400465278", rdb)
	client.region = "ap-chongqing"
	return client
}

func (s *SmsClient) send(phones []string, TemplateID string, TemplateParamSet []string) error {
	credential := common.NewCredential(
		s.secretId,
		s.secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"
	client, _ := sms.NewClient(credential, s.region, cpf)

	request := sms.NewSendSmsRequest()

	request.PhoneNumberSet = common.StringPtrs(phones)
	request.TemplateID = common.StringPtr(TemplateID)
	// 签名内容 https://console.cloud.tencent.com/smsv2/csms-sign
	request.Sign = common.StringPtr(s.sign)
	request.TemplateParamSet = common.StringPtrs(TemplateParamSet)
	// 短信应用ID https://console.cloud.tencent.com/smsv2/app-manage
	request.SmsSdkAppid = common.StringPtr(s.appId)

	response, err := client.SendSms(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		logger.J.Errorf("An API error has returned: %s", err)
		return err
	}
	if err != nil {
		return err
	}

	for _, value := range response.Response.SendStatusSet {
		b, _ := json.Marshal(value)
		if strings.Contains(string(b), "Ok") == false {
			logger.J.ErrEf(orginErrors.New("发送短信失败"), "发送号码:%v 错误:%v", value.PhoneNumber, value.Message)
			return orginErrors.New("短信发送失败,请稍后重试")
		}
	}
	return nil
}

// Send 这个是发送的登录验证码
func (s *SmsClient) Send(ctx context.Context, mobile string) (string, error) {

	var phones = []string{
		"+86" + mobile,
	}
	code := ut.RandomInt(1000, 9999)
	codeStr := strconv.Itoa(code)

	// 参数
	var sendParams = []string{
		codeStr,
		fmt.Sprintf("%v", s.expTime.Minutes()),
	}

	err := s.send(phones, "824190", sendParams)
	if err != nil {
		return "", err
	}

	resp := s.rdb.Do(ctx, s.rdb.B().Set().Key(s.redisPrefix+mobile).Value(codeStr).ExSeconds(int64(s.expTime.Seconds())).Build())
	if resp.Error() != nil {
		return "", resp.Error()
	}

	return codeStr, nil

}

func (s *SmsClient) SendBeforeCheck(ctx context.Context, mobile string) (string, error) {
	// 先验证key是否存在
	resp := s.rdb.Do(ctx, s.rdb.B().Exists().Key(s.redisPrefix+mobile).Build())
	if resp.Error() != nil {
		// 如果不存在的时候才进行发送
		if resp.Error() == redis.Nil {
			return s.Send(ctx, mobile)
		}
		return "", resp.Error()
	}

	has, _ := resp.AsBool()
	if has {
		return "", orginErrors.New("已有信息在路上,若未收到请稍后重试")
	}
	return s.Send(ctx, mobile)
}

func (s *SmsClient) Valid(ctx context.Context, mobile, code string) bool {
	resp := s.rdb.Do(ctx, s.rdb.B().Get().Key(s.redisPrefix+mobile).Build())
	if resp.Error() != nil {
		return false
	}
	val, err := resp.ToString()
	if err != nil {
		return false
	}
	return code == val
}

// DelKey 删除 key 让code验证
func (s *SmsClient) DelKey(ctx context.Context, mobile string) {
	_ = s.rdb.Do(ctx, s.rdb.B().Del().Key(s.redisPrefix+mobile).Build())
}
