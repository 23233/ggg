package htct

import (
	"github.com/PuerkitoBio/goquery"
)

// linkExtract 抽取所有链接
func linkExtract(body *goquery.Selection) []KvMap {
	var r = make([]KvMap, 0)

	body.Find("a").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		href, has := s.Attr("href")
		if has {
			r = append(r, KvMap{
				Key: text,
				Val: href,
			})
		}
	})
	return r
}
