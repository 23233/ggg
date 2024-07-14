package htct

import (
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func getHTML(url string) (string, error) {
	// 创建一个新的 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}
	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送 HTTP 请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 将响应体转换为字符串
	html := string(body)
	return html, nil
}
func Test_contentExtract(t *testing.T) {
	target := "https://www.163.com/dy/article/J73AFOGR0526SD6N.html"
	//target := "https://www.163.com/dy/article/J73AIAIT0526K1KN.html"
	source, err := getHTML(target)
	if err != nil {
		t.Fatal(err)
		return
	}
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(source))
	body := dom.Find("body")
	//normalize(body, Options{
	//	NoiseNodeList: []string{".post_statement", ".N-nav-bottom"},
	//})
	normalize(body, DefaultOptions)
	content := contentExtract(body, "div.post_body")
	t.Log(content.density.tiText)
	t.Log(content.node.Html())
}
