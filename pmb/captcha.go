package pmb

import (
	"time"

	"sync"

	"strings"

	"errors"

	"github.com/23233/gocaptcha"
	"github.com/google/uuid"
)

var (
	ImgCaptchaInst = NewImgCaptcha()
)

type ImgCaptchaItem struct {
	Text       string
	CreateTime time.Time
}

type ImgCaptcha struct {
	Records map[string]ImgCaptchaItem
	mutex   sync.Mutex
}

// 定义验证码相关的错误类型
var (
	ErrCaptchaNotFound = errors.New("验证码不存在或已过期")
	ErrCaptchaMismatch = errors.New("验证码不匹配")
)

func NewImgCaptcha() *ImgCaptcha {
	c := &ImgCaptcha{
		Records: make(map[string]ImgCaptchaItem),
	}
	// 启动清理协程
	go c.cleanExpired()
	return c
}

func (c *ImgCaptcha) GetNewImg(width, height int, textSize int, difficulty gocaptcha.CaptchaDifficulty) (string, []byte, error) {
	text, bt, err := gocaptcha.GenerateCaptcha(width, height, textSize, difficulty)
	if err != nil {
		return "", nil, err
	}

	id := uuid.New().String()
	c.mutex.Lock()
	c.Records[id] = ImgCaptchaItem{
		Text:       text,
		CreateTime: time.Now(),
	}
	c.mutex.Unlock()

	return id, bt, nil
}

// Verify 验证验证码，返回验证结果和具体错误信息
func (c *ImgCaptcha) Verify(id, text string) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, exists := c.Records[id]
	if !exists {
		return false, ErrCaptchaNotFound
	}

	// 验证后删除记录，防止重复使用
	delete(c.Records, id)

	if strings.ToLower(item.Text) != strings.ToLower(text) {
		return false, ErrCaptchaMismatch
	}

	return true, nil
}

// cleanExpired 清理过期的验证码
func (c *ImgCaptcha) cleanExpired() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for id, item := range c.Records {
			if now.Sub(item.CreateTime) > 5*time.Minute {
				delete(c.Records, id)
			}
		}
		c.mutex.Unlock()
	}
}
