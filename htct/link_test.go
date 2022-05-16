package htct

import (
	"github.com/PuerkitoBio/goquery"
	"strings"
	"testing"
)

func TestLinkExtract(t *testing.T) {
	dom, _ := goquery.NewDocumentFromReader(strings.NewReader(testSource))
	body := dom.Find("body")
	allLink := linkExtract(body)
	for _, k := range allLink {
		t.Logf("%s:%s", k.key, k.val)
	}
}
