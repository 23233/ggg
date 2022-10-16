package contentSafe

import (
	"errors"
	"fmt"
	"github.com/23233/ggg/sv"
	"github.com/23233/ggg/ut"
	"github.com/bluele/gcache"
	"github.com/imroc/req/v3"
	"github.com/kataras/iris/v12"
	"github.com/schollz/progressbar/v3"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

var (
	nsfwHost    = "http://nsfw.rycsg.com"
	tmpSaveBase = "./tmp"
	gc          gcache.Cache
)

type WxImgV1Resp struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

type HitImgReq struct {
	Uri string `json:"uri" form:"uri" comment:"图片地址" validate:"required"`
}

type Prediction struct {
	Drawings float32 `json:"drawings"` // 无害的艺术，或艺术绘画 最高100分
	Hentai   float32 `json:"hentai"`   // 色情艺术，不适合大多数工作环境
	Neutral  float32 `json:"neutral"`  // 一般，无害的内容
	Porn     float32 `json:"porn"`     // 不雅的内容和行为，通常涉及生殖器 性行为等
	Sexy     float32 `json:"sexy"`     // 性感 但无色情
}

func (c *Prediction) IsNsfw() bool {
	nsfwMax := math.Max(float64(c.Hentai), float64(c.Porn))
	return nsfwMax > 85
}

func HitImg(ctx iris.Context) {
	r := ctx.Values().Get(sv.GlobalContextKey).(*HitImgReq)

	value, err := gc.Get(r.Uri)
	if err == nil {
		_ = ctx.JSON(value)
		return
	}

	pass, err := AutoHitImg(r.Uri)
	if err != nil {
		_ = ctx.StopWithJSON(iris.StatusBadRequest, iris.Map{"detail": "判断图片不适程度失败"})
		return
	}

	resp := iris.Map{
		"success": pass,
	}

	_ = gc.Set(r.Uri, resp)

	_ = ctx.JSON(resp)
}

// AutoHitImg 检测是否为正常图片
func AutoHitImg(uri string) (bool, error) {
	p, err := fetchNsfw(uri)
	if err != nil {
		return false, err
	}
	// 20%概率 用接口校验
	if ut.RandomInt(1, 5) < 2 {
		pass, err := wxImgCheckV1(uri)
		if err != nil {
			return false, err
		}
		return pass, nil
	}
	return !p.IsNsfw(), nil

}

func fetchNsfw(remoteUri string) (*Prediction, error) {
	body := map[string]string{
		"url": remoteUri,
	}
	resp, err := req.R().SetBody(&body).Post(nsfwHost + "/remote")
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

// 返回的是 是否为正常图片
func wxImgCheckV1(remoteUri string) (bool, error) {
	// https://developers.weixin.qq.com/miniprogram/dev/framework/security.imgSecCheck.html
	uri := "https://api.weixin.qq.com/wxa/img_sec_check" + "?access_token=" + GetTokenFunc()
	imgPath, _, err := downloadImage(remoteUri)
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

func downloadImage(remoteUri string) (string, string, error) {
	fileURL, err := url.Parse(remoteUri)
	if err != nil {
		return "", "", err
	}
	fp := fileURL.Path
	segments := strings.Split(fp, "/")
	fileName := segments[len(segments)-1]

	savePath := path.Join(tmpSaveBase, fileName)
	_ = os.MkdirAll(tmpSaveBase, os.ModePerm)
	c, err := os.Stat(savePath)
	if err == nil {
		if c.Size() > 1*1024 {
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
	gc = gcache.New(10000).
		LRU().
		Build()
}
