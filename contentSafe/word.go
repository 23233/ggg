package contentSafe

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
