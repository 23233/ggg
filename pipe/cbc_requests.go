package pipe

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/kataras/iris/v12"
)

type CbcService struct {
	Key []byte
}

func NewCbcService(key []byte) *CbcService {
	return &CbcService{Key: key}
}

func (c *CbcService) Decrypt(cipherText string) ([]byte, error) {
	cipherData, err := base64.URLEncoding.DecodeString(cipherText)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(c.Key)
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

func (c *CbcService) Encrypt(plainText []byte) (string, error) {
	block, err := aes.NewCipher(c.Key)
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
func (c *CbcService) CbcMainHandler(ctx iris.Context) {
	// 添加这个 defer 语句
	defer func() {
		if r := recover(); r != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.JSON(iris.Map{"message": "执行过程错误", "code": 500})
		}
	}()
	encryptedData, _ := ctx.GetBody()
	decryptedData, err := c.Decrypt(string(encryptedData))
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(iris.Map{"message": "参数错误", "code": 1006})
		return
	}

	var requestData CbcRequestData
	err = json.Unmarshal(decryptedData, &requestData)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{"message": "参数格式错误", "code": 1007})
		return
	}
	if len(requestData.Body) < 5 {
		requestData.Body = json.RawMessage(`{}`)
	}

	handlerValue, ok := routerMap[requestData.Router]
	if !ok {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{"message": "未找到对应路由", "code": 1001})
		return
	}

	handlerType := handlerValue.Type()
	thirdParamType := handlerType.In(2)
	var thirdParamValue interface{}

	// 如果thirdParamType本身就是指针类型
	if thirdParamType.Kind() == reflect.Ptr {
		thirdParamValue = reflect.New(thirdParamType.Elem()).Interface()
	} else { // 如果thirdParamType是非指针类型
		thirdParamValue = reflect.New(thirdParamType).Interface()
	}

	err = json.Unmarshal(requestData.Body, &thirdParamValue)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(iris.Map{"message": "解析参数失败", "code": 400})
		return
	}

	var callParams []reflect.Value
	callParams = append(callParams, reflect.ValueOf(ctx), reflect.ValueOf(requestData.Query))

	// 根据thirdParamType是不是指针来决定如何添加到callParams
	if thirdParamType.Kind() == reflect.Ptr {
		callParams = append(callParams, reflect.ValueOf(thirdParamValue))
	} else {
		callParams = append(callParams, reflect.Indirect(reflect.ValueOf(thirdParamValue)))
	}

	handlerValue.Call(callParams)

}

type CbcRequestData struct {
	Query  map[string]interface{} `json:"query"`
	Body   json.RawMessage        `json:"body"`
	Router string                 `json:"router"`
}

type Handler[T any] func(ctx iris.Context, query map[string]interface{}, body T)

var routerMap = map[string]reflect.Value{}

func CbcRegisterHandler[T any](route string, handler Handler[T]) {
	routerMap[route] = reflect.ValueOf(handler)
}
