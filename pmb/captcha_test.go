package pmb

import (
	"errors"
	"testing"
	"time"
)

func TestNewImgCaptcha(t *testing.T) {
	// 测试生成验证码
	id, img, err := ImgCaptchaInst.GetNewImg(100, 40, 20, 0)
	if err != nil {
		t.Errorf("生成验证码失败: %v", err)
	}
	if id == "" {
		t.Error("验证码ID不应为空")
	}
	if len(img) == 0 {
		t.Error("验证码图片数据不应为空")
	}

	// 测试验证码记录是否存在
	if len(ImgCaptchaInst.Records) == 0 {
		t.Error("验证码记录未被保存")
	}

	// 测试验证码验证 - 错误的验证码
	ok, err := ImgCaptchaInst.Verify(id, "wrong_code")
	if ok {
		t.Error("使用错误的验证码不应验证通过")
	}
	if !errors.Is(err, ErrCaptchaMismatch) {
		t.Errorf("期望错误 ErrCaptchaMismatch，得到：%v", err)
	}

	// 测试不存在的验证码ID
	ok, err = ImgCaptchaInst.Verify("non_existent_id", "any_code")
	if ok {
		t.Error("使用不存在的ID不应验证通过")
	}
	if !errors.Is(err, ErrCaptchaNotFound) {
		t.Errorf("期望错误 ErrCaptchaNotFound，得到：%v", err)
	}

	// 测试验证码过期清理
	time.Sleep(6 * time.Minute)
	ok, err = ImgCaptchaInst.Verify(id, "any_code")
	if ok {
		t.Error("过期的验证码不应验证通过")
	}
	if !errors.Is(err, ErrCaptchaNotFound) {
		t.Errorf("期望错误 ErrCaptchaNotFound，得到：%v", err)
	}
}
