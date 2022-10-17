package contentSafe

import (
	"math"
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
