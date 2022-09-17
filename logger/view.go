package logger

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"
)

//go:embed online.html
var onlineHtmlFs []byte

//go:embed stats.html
var statsHtmlFs []byte

type htmlData struct {
	Label string   `json:"label"`
	Array []string `json:"array"`
}
type htmlResp struct {
	Data   []htmlData `json:"data"`
	IsJson bool       `json:"is_json"`
}

func (c *Log) ViewQueueFunc(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("online").Parse(string(onlineHtmlFs))
	if err != nil {
		panic(err.Error())
	}
	var tempData []htmlData

	tempData = append(tempData, htmlData{
		Label: "info log list view",
		Array: reverse(c.Op.InfoQueue().ItemsStr()),
	})
	tempData = append(tempData, htmlData{
		Label: "error log list view",
		Array: reverse(c.Op.ErrorQueue().ItemsStr()),
	})

	var resp htmlResp
	resp.Data = tempData
	resp.IsJson = c.Op.Encoding != _defaultEncoding

	if err := t.Execute(w, resp); err != nil {
		panic(err.Error())
	}
}

func (c *Log) ViewStatsFunc(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("stats").Parse(string(statsHtmlFs))
	if err != nil {
		panic(err.Error())
	}
	var st = c.Op.GetStats()
	stByte, err := json.Marshal(st)
	if err != nil {
		c.Errorf("marshal stats data fail %v", err)
		return
	}

	if err := t.Execute(w, string(stByte)); err != nil {
		panic(err.Error())
	}
}

func reverse[T any](original []T) (reversed []T) {
	reversed = make([]T, len(original))
	copy(reversed, original)

	for i := len(reversed)/2 - 1; i >= 0; i-- {
		tmp := len(reversed) - 1 - i
		reversed[i], reversed[tmp] = reversed[tmp], reversed[i]
	}

	return
}
