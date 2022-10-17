package contentSafe

import (
	"embed"
	"fmt"
	"github.com/23233/ggg/logger"
	"github.com/23233/ggg/sv"
	"github.com/23233/ggg/ut"
	"github.com/23233/lad"
	"github.com/bluele/gcache"
	"github.com/imroc/req/v3"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"unicode"
)

//go:embed *.txt
var words embed.FS

var (
	C *ContentSafe
)

type ContentSafe struct {
	getTokenFunc func() string
	NsfwHost     string
	TmpSaveBase  string
	gc           gcache.Cache
	lad          *lad.AcMachine
}

func NewSafeClient() *ContentSafe {
	client := &ContentSafe{
		NsfwHost:    "http://nsfw.rycsg.com",
		TmpSaveBase: "./tmp",
		gc: gcache.New(10000).
			LRU().
			Build(),
		lad: lad.New(),
	}
	_ = client.InitLadClient()

	return client
}

func (c *ContentSafe) GetToken() string {
	return c.getTokenFunc()
}

func (c *ContentSafe) SetGetTokenFunc(getTokenFunc func() string) {
	c.getTokenFunc = getTokenFunc
}

func (c *ContentSafe) RegistryRouters(party iris.Party) {
	party.Post("/hit_text", sv.Run(new(WordValidReq)), c.HitTextHandler)
	party.Post("/hit_image", sv.Run(new(HitImgReq)), c.HitImgHandler)
}

// 以下是文字部分

func (c *ContentSafe) InitLadClient() error {
	err := c.lad.LoadOfFolder(words)
	if err != nil {
		return err
	}
	c.lad.Build()
	return nil
}

func (c *ContentSafe) HitTextHandler(ctx iris.Context) {
	q := ctx.Values().Get(sv.GlobalContextKey).(*WordValidReq)

	success, message := c.AutoHitText(q.Content)

	_ = ctx.JSON(WordHitResp{
		Success: success, // 为true则是安全文本
		Msg:     message,
	})
}

func (c *ContentSafe) AutoHitText(content string) (success bool, message string) {
	success = true
	message = "ok"
	if len(content) > 200 && ut.RandomInt(1, 5) < 2 && len(c.GetToken()) > 0 {
		// 长文本 20%概率使用接口校验
		pass, err, msg := c.WxTextCheckV1(content)
		if err != nil {
			logger.J.Error("校验文本失败", logger.J.WithError(err))
		}
		success = pass
		message = msg
	} else {
		pass, msg := c.HitText(content)
		success = pass
		message = msg
	}
	return success, message
}

func (c *ContentSafe) HitText(content string) (bool, string) {
	// 去掉所有标点符号
	clear := c.ClearText(content)

	if len(clear) >= 1 {
		if c.lad.Match(clear) {
			return false, "中文有不良词汇,请修改"
		}
	}
	return true, "ok"
}

func (c *ContentSafe) ClearText(words string) string {
	var m = make([]string, 0)
	for _, runeValue := range words {
		// 中文则直接生成 或者 非中文字符 非符号 非空格的内容
		if unicode.Is(unicode.Han, runeValue) || (!unicode.IsPunct(runeValue) && !unicode.IsSymbol(runeValue) && !unicode.IsSpace(runeValue)) {
			m = append(m, string(runeValue))
		}
	}
	return Tc2Cn(strings.Join(m, ""))
}

func (c *ContentSafe) WxTextCheckV1(content string) (bool, error, string) {
	// https://developers.weixin.qq.com/miniprogram/dev/framework/security.imgSecCheck.html
	uri := "https://api.weixin.qq.com/wxa/msg_sec_check"
	query := map[string]string{
		"access_token": c.GetToken(),
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

func (c *ContentSafe) AddWords(wordList ...string) {
	if len(wordList) >= 1 {
		c.lad.AddOfList(wordList)
		c.lad.Build()
	}
}

// 以下是图像部分

func (c *ContentSafe) HitImgHandler(ctx iris.Context) {
	r := ctx.Values().Get(sv.GlobalContextKey).(*HitImgReq)

	value, err := c.gc.Get(r.Uri)
	if err == nil {
		_ = ctx.JSON(value)
		return
	}

	pass, err := c.AutoHitImg(r.Uri)
	if err != nil {
		_ = ctx.StopWithJSON(iris.StatusBadRequest, iris.Map{"detail": "判断图片不适程度失败"})
		return
	}

	resp := iris.Map{
		"success": pass,
	}

	_ = c.gc.Set(r.Uri, resp)

	_ = ctx.JSON(resp)
}

// AutoHitImg 检测是否为正常图片
func (c *ContentSafe) AutoHitImg(uri string) (bool, error) {
	p, err := c.FetchNsfw(uri)
	if err != nil {
		return false, err
	}
	// 20%概率 用接口校验
	if ut.RandomInt(1, 5) < 2 && len(c.GetToken()) > 0 {
		pass, err := c.WxImgCheckV1(uri)
		if err != nil {
			return false, err
		}
		return pass, nil
	}
	return !p.IsNsfw(), nil

}

func (c *ContentSafe) FetchNsfw(remoteUri string) (*Prediction, error) {
	body := map[string]string{
		"url": remoteUri,
	}
	resp, err := req.R().SetBody(&body).Post(c.NsfwHost + "/remote")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		msg, err := resp.ToString()
		if err != nil {
			return nil, err
		}
		return nil, errors.New(msg)
	}
	var j Prediction
	err = resp.UnmarshalJson(&j)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// WxImgCheckV1 返回的是 是否为正常图片
func (c *ContentSafe) WxImgCheckV1(remoteUri string) (bool, error) {
	// https://developers.weixin.qq.com/miniprogram/dev/framework/security.imgSecCheck.html
	uri := "https://api.weixin.qq.com/wxa/img_sec_check" + "?access_token=" + c.GetToken()
	imgPath, _, err := c.DownloadImage(remoteUri)
	if err != nil {
		return false, err
	}
	resp, err := req.R().SetFile("media", imgPath).Post(uri)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != 200 {
		msg, err := resp.ToString()
		if err != nil {
			return false, err
		}
		return false, errors.New(msg)
	}
	var j WxImgV1Resp
	err = resp.UnmarshalJson(&j)
	if err != nil {
		return false, err
	}
	return j.Errcode == 0, nil
}

func (c *ContentSafe) DownloadImage(remoteUri string) (savePath string, fileName string, err error) {
	fileURL, err := url.Parse(remoteUri)
	if err != nil {
		return "", "", err
	}
	fp := fileURL.Path
	segments := strings.Split(fp, "/")
	fileName = segments[len(segments)-1]

	savePath = path.Join(c.TmpSaveBase, fileName)
	_ = os.MkdirAll(c.TmpSaveBase, os.ModePerm)
	st, err := os.Stat(savePath)
	if err == nil {
		if st.Size() > 1*1024 {
			return savePath, fileName, nil
		}
	}
	// Create blank file
	f, err := os.Create(savePath)
	if err != nil {
		return "", "", err
	}

	resp, err := http.Get(remoteUri)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("get remote image bad status: %s", resp.Status)
	}

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading "+fileName,
	)

	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	defer f.Close()

	return savePath, fileName, nil

}

func init() {
	C = NewSafeClient()
}
