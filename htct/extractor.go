package htct

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

// ignoreTag 需要忽略的标签
var ignoreTag = []string{"style", "script", "link", "video", "iframe", "source", "picture", "header", "noscript"}

// ignoreClass 需要忽略的class, 当标签的class属性中包含下列值时, 忽略该标签
var ignoreClass = []string{"share", "contribution", "copyright", "copy-right", "disclaimer", "recommend", "related", "footer", "comment", "social", "submeta", "report-infor"}

// canBeRemoveIfEmpty 可以移除的空标签
var canBeRemoveIfEmpty = []string{"section", "h1", "h2", "h3", "h4", "h5", "h6", "span"}

// Article 提取的文本信息
type Article struct {
	MetaTitle       string `json:"meta_title,omitempty" bson:"meta_title,omitempty"`
	MetaDescription string `json:"meta_description,omitempty" bson:"meta_description,omitempty"`
	MetaKeywords    string `json:"meta_keywords,omitempty" bson:"meta_keywords,omitempty"`
	MetaFav         string `json:"meta_fav,omitempty" bson:"meta_fav,omitempty"`
	Url             string `json:"url,omitempty" bson:"url,omitempty"` // 当前的url
	Icp             string `json:"icp,omitempty" bson:"icp,omitempty"` // icp备案号
	Ipv4            string `json:"ipv_4,omitempty" bson:"ipv_4,omitempty"`
	Ipv6            string `json:"ipv_6,omitempty" bson:"ipv_6,omitempty"`
	// ContentTitle 标题
	ContentTitle string `json:"content_title,omitempty" bson:"content_title,omitempty"`
	// ContentImages 图片
	ContentImages []string `json:"content_images,omitempty" bson:"content_images,omitempty"`
	// Author 作者
	Author string `json:"author,omitempty" bson:"author,omitempty"`
	// PublishTime 发布时间
	PublishTime string `json:"publish_time,omitempty" bson:"publish_time,omitempty"`
	// Content 正文
	Content string `json:"content,omitempty" bson:"content,omitempty"`
	// ContentLine 内容段落 以`\n`为切分标准
	ContentLine []string `json:"content_line,omitempty" bson:"content_line,omitempty"`
	// ContentHTML 正文源码
	ContentHTML string `json:"content_html,omitempty" bson:"content_html,omitempty"`
	// AllLinks 所有链接 内外链
	AllLinks []KvMap `json:"all_links,omitempty" bson:"all_links,omitempty"`
}

// GetOutLinks 获取出所有的外链
func (a *Article) GetOutLinks() []string {

	var cleanUrls = make(map[string]struct{})
	// 发现并加入新的URL
	for _, link := range a.AllLinks {
		// 解析URL以获取域名
		parsedURL, err := url.Parse(link.Val)
		if err != nil {
			// 处理错误
			continue
		}
		if len(parsedURL.Host) < 1 {
			continue
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			continue
		}
		// 包含:则有端口号 这种跳过
		if strings.Contains(parsedURL.Host, ":") {
			continue
		}
		// 重构URL以仅包含方案和主机
		cleanURL := parsedURL.Scheme + "://" + parsedURL.Host
		cleanUrls[cleanURL] = struct{}{}
	}

	var keys = make([]string, 0, len(cleanUrls))
	for k, _ := range cleanUrls {
		keys = append(keys, k)
	}
	return keys

}

// GetInnerLinks 获取内联 可传containerStr 则为包含模式
func (a *Article) GetInnerLinks(containerStr ...string) []string {
	var result []string
	for _, m := range a.AllLinks {
		if m.Val == "javascript:;" {
			continue
		}
		if m.Val == "#" {
			continue
		}
		if len(containerStr) == 0 || containsAny(m.Val, containerStr) {
			result = append(result, m.Val)
		}
	}
	return result
}

// RegMatchLinks 正则匹配出符合的链接
func (a *Article) RegMatchLinks(regStr string) ([]string, error) {
	matchRe, err := regexp.Compile(regStr)
	if err != nil {
		return nil, err
	}
	matchList := make([]string, 0, len(a.AllLinks))
	for _, link := range a.AllLinks {
		if matchRe.MatchString(link.Val) {
			matchList = append(matchList, link.Val)
		}
	}
	return matchList, nil
}

// 辅助函数，检查val是否包含substrs切片中的任意一个子字符串。
func containsAny(val string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(val, substr) {
			return true
		}
	}
	return false
}

type KvMap struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

type Options struct {
	NoiseNodeList []string `json:"noise_node_list"` // 噪音节点内容是使用css选择器
	ContentSelect string   `json:"content_select"`  // 文章内容的css 如果传入了则会把内容锁定到这个类上 而不会去走分值计算那一套
}

var (
	DefaultOptions = Options{}
)

// Extract 提取信息 sourceUrl 得包含协议头才能获取完整
func Extract(source string, sourceUrl string, ops ...Options) (*Article, error) {
	op := DefaultOptions
	if len(ops) >= 1 {
		op = ops[0]
	}
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	body := dom.Find("body")
	normalize(body, op)
	result := &Article{}
	headText := headTextExtract(dom)
	wg := &sync.WaitGroup{}
	wg.Add(6)
	go func() {
		defer wg.Done()
		result.PublishTime = timeExtract(headText, body)
	}()
	go func() {
		defer wg.Done()
		result.AllLinks = linkExtract(body)
	}()
	go func() {
		defer wg.Done()
		result.Author = authorExtract(headText, body)
	}()
	go func() {
		defer wg.Done()
		content := contentExtract(body, op.ContentSelect)
		result.ContentTitle = titleExtract(headText, dom.Selection, content.node)
		result.Content = content.density.tiText
		result.ContentLine = strings.Split(result.Content, "\n")
		result.ContentHTML, _ = content.node.Html()
		var imgs []string
		content.node.Find("img").Each(func(i int, s *goquery.Selection) {
			if src, ok := s.Attr("src"); ok {
				imgs = append(imgs, src)
			} else {
				var dataSrcs = []string{"data-src", "data-pic", "data-big"}
				for _, dSrc := range dataSrcs {
					if dataSrc, ok := s.Attr(dSrc); ok {
						imgs = append(imgs, dataSrc)
						break
					}
				}
			}

		})
		result.ContentImages = imgs
	}()
	go func() {
		defer wg.Done()
		// 抽取tdk
		dom.Find("title").Each(func(i int, s *goquery.Selection) {
			result.MetaTitle = strings.TrimSpace(s.Text())
		})
		dom.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
			description, _ := s.Attr("content")
			result.MetaDescription = strings.TrimSpace(description)
		})
		dom.Find("meta[name='keywords']").Each(func(i int, s *goquery.Selection) {
			keywords, _ := s.Attr("content")
			result.MetaKeywords = strings.TrimSpace(keywords)
		})

		// 找寻备案号所在的a标签，并提取其文本或href作为备案号
		dom.Find("a[href*='beian.miit.gov.cn']").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists {
				if strings.Contains(href, "beian.miit.gov.cn") {
					result.Icp = strings.TrimSpace(s.Text()) // 假设备案号在文本中
				}
			}
		})

		result.Url = sourceUrl

		// ip地址
		ipv4, ipv6, _ := LookupIPAddresses(sourceUrl)
		result.Ipv6 = ipv6
		result.Ipv4 = ipv4

	}()
	go func() {
		defer wg.Done()
		u, err := url.Parse(sourceUrl)
		if err != nil {
			return
		}

		favB64, err := extractFavicon(dom, u.Scheme+"://"+u.Hostname()+"/")
		if err != nil {
			return
		}
		result.MetaFav = favB64
	}()

	wg.Wait()
	return result, nil
}

func headTextExtract(dom *goquery.Document) []*KvMap {
	var (
		rs       []*KvMap
		head     = dom.Find("head")
		metaSkip = map[string]bool{
			"charset":    true,
			"http-equiv": true,
		}
	)
	for _, v := range iterator(head) {
		if goquery.NodeName(v) != "meta" {
			continue
		}
		for _, v := range v.Nodes {
			key := ""
			val := ""
			for _, v2 := range v.Attr {
				if metaSkip[v2.Key] {
					key = ""
					break
				}
				if v2.Key == "name" || v2.Key == "property" {
					key = strings.ToLower(v2.Val)
				} else if v2.Key == "content" {
					val = v2.Val
				}
			}
			if key != "" && val != "" {
				length := utf8.RuneCountInString(strings.TrimSpace(val))
				if length >= 2 && length <= 50 {
					rs = append(rs, &KvMap{
						Key: key,
						Val: val,
					})
				}
			}
		}
	}
	sort.Slice(rs, func(i, j int) bool {
		return len(rs[i].Key) > len(rs[j].Key)
	})
	return rs
}

func extractFavicon(dom *goquery.Document, domain string) (string, error) {
	// 查找<link>标签
	iconURL, exists := dom.Find("link[rel~='icon']").Attr("href")
	if !exists {
		return "", fmt.Errorf("favicon not found")
	}

	// 处理相对URL
	if !strings.HasPrefix(iconURL, "http") {
		// 这里需要一个完整的URL，例如通过解析当前页面的URL来构造
		// 假设当前页面URL是 http://example.com/
		// 完整的URL将是 http://example.com/favicon.ico
		// 你需要根据实际情况来构造完整的URL
		fullURL := domain + iconURL
		iconURL = fullURL
	}

	// 下载图标
	resp, err := http.Get(iconURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取文件内容
	faviconData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 编码为Base64
	faviconBase64 := base64.StdEncoding.EncodeToString(faviconData)

	return faviconBase64, nil
}

// normalize 初始化节点
func normalize(element *goquery.Selection, op Options) {
	for _, v := range ignoreTag {
		element.Find(v).Remove()
	}
	// 对于定义的噪音节点 进行删除
	for _, noiseSelect := range op.NoiseNodeList {
		element.Find(noiseSelect).Remove()
	}
	for _, v := range iterator(element) {
		tagName := goquery.NodeName(v)
		// 删除注释
		if tagName == "#comment" {
			v.Remove()
			continue
		}
		// 删除一些可以删除的空标签
		if canBeRemove(v) {
			v.Remove()
			continue
		}

		// 删除标签class中含有ignoreClass的标签
		if val, ok := v.Attr("class"); ok {
			for _, class := range ignoreClass {
				if strings.Contains(val, class) {
					v.Remove()
					continue
				}
			}
		}
		// 去除p标签中的span, strong, em, b
		if tagName == "p" {
			v.Find("span,strong,em,b").Each(func(i int, child *goquery.Selection) {
				text := child.Text()
				child.ReplaceWithHtml(text)
			})
		}
		// 将没有子节点的div转换为p
		if tagName == "div" && v.Children().Length() <= 0 {
			v.Get(0).Data = "p"
		}
		// 去除空的p, 因上一步此处必须重新获取tagName
		if goquery.NodeName(v) == "p" {
			if v.Children().Length() <= 0 && len(strings.TrimSpace(v.Text())) == 0 {
				v.Remove()
			}
		}
	}
}

// iterator 遍历所有节点
func iterator(s *goquery.Selection) []*goquery.Selection {
	var result []*goquery.Selection
	iteratorNode(s, func(child *goquery.Selection) {
		result = append(result, child)
	})
	return result
}

func iteratorNode(s *goquery.Selection, fn func(*goquery.Selection)) {
	if s == nil {
		return
	}
	fn(s)
	s.Contents().Each(func(i int, c *goquery.Selection) {
		iteratorNode(c, fn)
	})
}

// canBeRemove 判断节点是否可以移除
// 判定标准为 无子节点并且在去除前后空格的情况下自身文本为空
func canBeRemove(s *goquery.Selection) bool {
	for _, v := range canBeRemoveIfEmpty {
		if strings.ToLower(goquery.NodeName(s)) == v {
			if s.Children().Length() <= 0 && strings.TrimSpace(s.Text()) == "" {
				return true
			}
		}
	}
	return false
}

// LookupIPAddresses 根据给定的域名返回一个IPv4和一个IPv6地址（如果存在的话）
// domain必须不包含协议头
func LookupIPAddresses(domain string) (string, string, error) {
	// 移除可能的HTTP或HTTPS协议头
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")

	// 找到第一个"/"出现的位置；如果找不到，`slashIndex`将为-1
	slashIndex := strings.Index(domain, "/")
	if slashIndex != -1 {
		// 只截取第一个"/"之前的部分，移除了路径和查询字符串
		domain = domain[0:slashIndex]
	}

	ips, err := net.LookupIP(domain)
	if err != nil {
		return "", "", err // 如果查询失败，返回错误
	}

	var ipv4Addr, ipv6Addr string
	for _, ip := range ips {
		ipv4 := ip.To4()
		// 如果找到IPv4地址而且之前没有找到过IPv4地址，就记录下来
		if ipv4 != nil && ipv4Addr == "" {
			ipv4Addr = ipv4.String()
			continue
		}
		// 如果找到IPv6地址而且之前没有找到过IPv6地址，就记录下来
		if ipv4 == nil && ipv6Addr == "" { // 如果不是IPv4，即为IPv6
			ipv6Addr = ip.String()
		}

		// 如果两种地址都已找到，提前结束循环
		if ipv4Addr != "" && ipv6Addr != "" {
			break
		}
	}

	// 返回找到的IPv4和IPv6地址
	// 注意：这二者有可能为空，表示没有找到对应类型的地址
	return ipv4Addr, ipv6Addr, nil
}
