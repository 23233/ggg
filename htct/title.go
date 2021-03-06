package htct

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	titleRx = regexp.MustCompile("_|\\|")
	titlePx = []string{
		"articletitle",
		":title",
		"title",
	}
)

// titleExtract 提取文章作者
// source 网页源码
// content 正文
// 依次 从header中找 `meta` title 标签 -> 直接搜索`title` 标签 -> 找 `H1` 标签
func titleExtract(headText []*KvMap, source *goquery.Selection, content *goquery.Selection) string {
	var title string
	for _, v := range headText {
		for _, px := range titlePx {
			if strings.Contains(v.Key, px) {
				if title = strings.TrimSpace(titleRx.Split(v.Val, -1)[0]); title != "" {
					return title
				}
			}
		}
	}
	titleNode := source.Find("title")
	if titleNode.Length() > 0 {
		title = strings.TrimSpace(titleRx.Split(titleNode.Text(), -1)[0])
		if title != "" {
			return title
		}
	}

	title = titleExtractFromHTML(content)
	return title

}

// titleExtractFromHTML 从html标签中查找
func titleExtractFromHTML(content *goquery.Selection) string {
	title, _ := findHtag(content)
	return title
}

func findHtag(content *goquery.Selection) (string, bool) {
	parent := content.Parent()
	htag := parent.Find("h1,h2,h3,h4")
	if htag.Length() > 0 {
		return htag.Eq(0).Text(), true
	}
	if len(parent.Nodes) < 1 || parent.Nodes[0].Parent == nil {
		return "", false
	}
	return findHtag(parent)
}
