package scene

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/23233/ggg/pipe"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis/rueidiscompat"
	"io"
	"net/http"
	"time"
)

var (
	ScenesInst = NewScenes()
)

type ScenesRateKeyFunc func(ctx iris.Context) (string, error)

type ScenesOptions struct {
	Scene   ISceneModelItem
	rate    *RateLimiter
	rateKey ScenesRateKeyFunc
}

type ScenesItem struct {
	scenes map[string]*ScenesOptions
	scope  string
	model  string
}

type ScenesHandler func(ctx iris.Context, mapper *RequestData) *pipe.RunResp[any]

type Scenes struct {
	modules      []*ScenesItem
	cbcIv        []byte
	preHandler   []ScenesHandler
	afterHandler []ScenesHandler
	prefix       string
	party        iris.Party
}

type IScenes interface {
	GetPrefix() string
	SetPrefix(prefix string)
	SetParty(party iris.Party)
	GetParty() iris.Party

	GetCbcSign() bool
	SetCbcSign(iv []byte)
	GetCbcIv() []byte

	Decrypt(cipherText string) ([]byte, error)
	Encrypt(plainText []byte) (string, error)

	AddPreHandler(handler ...ScenesHandler)
	GetPreHandler() []ScenesHandler
	AddAfterHandler(handler ...ScenesHandler)
	GetAfterHandler() []ScenesHandler

	RegistryRouter(party iris.Party, prefix string)
	ParseRequest() iris.Handler

	GetItem(scope string, model string, scene string) (*ScenesOptions, bool)
	GetScope(scope string) []*ScenesItem
	GetModel(scope string, model string) []*ScenesItem
	RegistryItem(scope string, model string, scene string, item ISceneModelItem) error
	AddItemRate(scope string, model string, scene string, redisClient rueidiscompat.Cmdable, period time.Duration, maxCount int, interfaceKey string, keyFunc ScenesRateKeyFunc) error
}

func (m *Scenes) GetCbcSign() bool {
	return m.cbcIv != nil
}

func (m *Scenes) SetCbcSign(iv []byte) {
	m.cbcIv = iv
}

func (m *Scenes) GetCbcIv() []byte {
	return m.cbcIv
}

func (m *Scenes) GetItem(scope string, model string, scene string) (*ScenesOptions, bool) {
	for _, module := range m.modules {
		if module.scope == scope && module.model == model {
			item, exists := module.scenes[scene]
			return item, exists
		}
	}
	return nil, false
}

func (m *Scenes) GetScope(scope string) []*ScenesItem {
	var result []*ScenesItem
	for _, module := range m.modules {
		if module.scope == scope {
			result = append(result, module)
		}
	}
	return result
}

func (m *Scenes) GetModel(scope string, model string) []*ScenesItem {
	var result []*ScenesItem
	for _, module := range m.modules {
		if module.scope == scope && module.model == model {
			result = append(result, module)
		}
	}
	return result
}

func (m *Scenes) RegistryItem(scope string, model string, scene string, item ISceneModelItem) error {
	var module *ScenesItem
	for _, mod := range m.modules {
		if mod.scope == scope && mod.model == model {
			module = mod
			break
		}
	}

	if module == nil {
		module = &ScenesItem{
			scenes: make(map[string]*ScenesOptions),
			scope:  scope,
			model:  model,
		}
		m.modules = append(m.modules, module)
	}

	if _, exists := module.scenes[scene]; exists {
		return fmt.Errorf("scene already exists")
	}

	module.scenes[scene] = &ScenesOptions{
		Scene: item,
	}
	return nil
}

func (m *Scenes) Decrypt(cipherText string) ([]byte, error) {
	cipherData, err := base64.URLEncoding.DecodeString(cipherText)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(m.cbcIv)
	if err != nil {
		return nil, err
	}

	if len(cipherData) < aes.BlockSize {
		return nil, fmt.Errorf("cipher text too short")
	}

	iv := cipherData[:aes.BlockSize]
	cipherData = cipherData[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(cipherData, cipherData)

	padding := cipherData[len(cipherData)-1]
	return cipherData[:len(cipherData)-int(padding)], nil
}

func (m *Scenes) Encrypt(plainText []byte) (string, error) {
	block, err := aes.NewCipher(m.cbcIv)
	if err != nil {
		return "", err
	}

	padding := aes.BlockSize - len(plainText)%aes.BlockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	plainText = append(plainText, padText...)

	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[aes.BlockSize:], plainText)

	return base64.URLEncoding.EncodeToString(cipherText), nil
}

func (m *Scenes) AddPreHandler(handler ...ScenesHandler) {
	m.preHandler = append(m.preHandler, handler...)
}

func (m *Scenes) AddAfterHandler(handler ...ScenesHandler) {
	m.afterHandler = append(m.afterHandler, handler...)
}

func (m *Scenes) GetPreHandler() []ScenesHandler {
	return m.preHandler
}
func (m *Scenes) GetAfterHandler() []ScenesHandler {
	return m.afterHandler
}

func (m *Scenes) GetPrefix() string {
	return m.prefix
}
func (m *Scenes) SetPrefix(prefix string) {
	m.prefix = prefix
}
func (m *Scenes) SetParty(party iris.Party) {
	m.party = party
}
func (m *Scenes) GetParty() iris.Party {
	return m.party
}

func (m *Scenes) errMsg(ctx iris.Context, msg string, err error, codes ...int) {
	code := iris.StatusBadRequest
	if len(codes) >= 1 {
		code = codes[0]
	}
	ctx.StatusCode(code)
	ctx.JSON(iris.Map{"detail": msg})
}

func (m *Scenes) ParseRequest() iris.Handler {
	return func(ctx iris.Context) {
		// 仅接受POST请求
		if ctx.Method() != "POST" {
			ctx.StatusCode(iris.StatusMethodNotAllowed)
			return
		}

		var requestData = new(RequestData)

		// 检查是否需要解密
		if m.GetCbcSign() {
			// 读取请求体
			body, err := ctx.GetBody()
			if err != nil {
				m.errMsg(ctx, "获取body失败", err)
				return
			}

			// 解密请求体
			decryptedBody, err := m.Decrypt(string(body))
			if err != nil {
				m.errMsg(ctx, "解密失败", err)
				return
			}

			// 解析请求数据
			if err := json.Unmarshal(decryptedBody, requestData); err != nil {
				m.errMsg(ctx, "请求参数错误", err)
				return
			}
		} else {
			// 直接解析请求数据
			if err := ctx.ReadJSON(requestData); err != nil {
				m.errMsg(ctx, "解析请求参数格式失败", err)
				return
			}
		}
		if len(requestData.Module.Scope) < 1 || len(requestData.Module.Model) < 1 || len(requestData.Module.Scene) < 1 {
			m.errMsg(ctx, "必传参数缺失", nil)
			return
		}
		ctx.Values().Set("body_data", requestData)
		ctx.Next()
	}
}

func (m *Scenes) RegistryRouter(party iris.Party, prefix string) {
	m.SetPrefix(prefix)
	m.SetParty(party)
	req := party.Post("/"+prefix, m.ParseRequest())

	// 添加预处理程序
	for _, handler := range m.GetPreHandler() {
		req.Use(func(ctx iris.Context) {
			resp := handler(ctx, ctx.Values().Get("body_data").(*RequestData))
			resp.Return(ctx)
			if resp.Err != nil {
				return
			}
			ctx.Next()
		})
	}

	// 主处理程序
	req.Handlers = append(req.Handlers, func(ctx iris.Context) {
		mapperBody := ctx.Values().Get("body_data").(*RequestData)
		item, ok := m.GetItem(mapperBody.Module.Scope, mapperBody.Module.Model, mapperBody.Module.Scene)
		if !ok {
			m.errMsg(ctx, "获取对应handler失败", nil)
			return
		}

		// 限速器实现
		if item.rate != nil {
			key, err := item.rateKey(ctx)
			if err != nil {
				m.errMsg(ctx, "获取限速器字段失败", err)
				return
			}
			allow, err := item.rate.Allow(ctx, key)
			if err != nil {
				m.errMsg(ctx, "限速器执行失败", err)
				return
			}
			if !allow {
				m.errMsg(ctx, "当前请求过多,请稍后重试", nil, http.StatusTooManyRequests)
				return
			}
		}

		cloneItem := item.Scene.Clone()
		cloneItem.SetMapper(mapperBody)
		resp := cloneItem.GetHandler()(ctx, cloneItem)
		resp.Return(ctx)
		if resp.Err != nil {
			return
		}
		ctx.Next()
	})

	// 添加后处理程序
	for _, handler := range m.GetAfterHandler() {
		req.Done(func(ctx iris.Context) {
			resp := handler(ctx, ctx.Values().Get("body_data").(*RequestData))
			resp.Return(ctx)
			if resp.Err != nil {
				return
			}
			ctx.Next()
		})
	}
}

func (m *Scenes) AddItemRate(scope string, model string, scene string, redisClient rueidiscompat.Cmdable, period time.Duration, maxCount int, interfaceKey string, keyFunc ScenesRateKeyFunc) error {
	item, ok := m.GetItem(scope, model, scene)
	if !ok {
		return errors.New("未找到对应的scene")
	}
	item.rate = NewRateLimiter(redisClient, period, maxCount, interfaceKey)
	item.rateKey = keyFunc
	return nil
}

func NewScenes() IScenes {
	return new(Scenes)
}
