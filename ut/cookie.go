package ut

import (
	"fmt"
	"net/http"
	"strings"
)

// ParseCookieStrToCookies 解析指定分隔符的字符串cookie为 http.Cookie
func ParseCookieStrToCookies(rawStr, sep string) []*http.Cookie {
	parts := strings.Split(rawStr, sep)
	cookies := make([]*http.Cookie, 0)
	for _, part := range parts {
		item := strings.Split(part, "=")
		key := strings.TrimSpace(item[0])
		value := strings.TrimSpace(item[1])
		cookie := &http.Cookie{
			Name:  key,
			Value: value,
		}
		cookies = append(cookies, cookie)
	}
	return cookies
}

// ParseMapToCookies 解析map格式的cookie为 http.Cookie
func ParseMapToCookies(inputMap map[string]string) []*http.Cookie {
	var cookies []*http.Cookie
	for key, value := range inputMap {
		cookie := &http.Cookie{
			Name:  key,
			Value: value,
		}
		cookies = append(cookies, cookie)
	}
	return cookies
}

// OverrideTemplateWithCookies 多字段覆盖模板cookie中的字段返回 http.Cookie
func OverrideTemplateWithCookies(templateStr string, templateMap map[string]string) []*http.Cookie {
	// 将cookie字符串分割为单独的键值对
	cookiePairs := strings.Split(templateStr, "; ")
	cookieMap := make(map[string]string)
	for _, pair := range cookiePairs {
		if pair == "" {
			continue
		}
		// 分割每个键值对
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			cookieMap[kv[0]] = kv[1]
		}
	}

	// 使用cookieMap覆盖templateMap中的值
	for key, value := range cookieMap {
		if _, ok := templateMap[key]; ok {
			templateMap[key] = value
		}
	}

	// 将模板map转换为http.Cookie切片
	var cookies []*http.Cookie
	for key, value := range templateMap {
		cookies = append(cookies, &http.Cookie{Name: key, Value: value})
	}

	return cookies
}

// CookieStrReplace 对字符串cookie任何key进行替换改写 返回变更后的cookie字符串
func CookieStrReplace(s, k, v string) string {
	pairs := strings.Split(s, ";")
	updated := false

	for i, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == k {
			pairs[i] = fmt.Sprintf("%s=%s", k, v)
			updated = true
			break
		}
	}

	if !updated {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(pairs, ";")
}

// ParseCookiesToStr 将HTTP Cookie切片编码为Cookie字符串
func ParseCookiesToStr(cookies []*http.Cookie, sep string) string {

	if len(cookies) < 1 {
		return ""
	}
	mp := make(map[string]string, 0)
	for _, cookie := range cookies {
		mp[cookie.Name] = cookie.Value
	}

	var parts []string
	for k, v := range mp {
		var sb strings.Builder
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
		parts = append(parts, sb.String())
	}
	return strings.Join(parts, sep)
}

// FindCookieValue 从字符串cookie中找到任一Key
func FindCookieValue(cookieStr string, key string) (string, bool) {
	// 以分号为分隔符分割cookie字符串
	pairs := strings.Split(cookieStr, ";")

	for _, pair := range pairs {
		// 移除键值对周围的空格
		pair = strings.TrimSpace(pair)
		// 检查键值对是否包含等号
		if strings.Contains(pair, "=") {
			// 分割键值对
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 && kv[0] == key {
				return kv[1], true
			}
		}
	}
	return "", false
}

// MustFindCookieValue 未找到则返回空字符串
func MustFindCookieValue(cookieStr string, key string) string {
	v, _ := FindCookieValue(cookieStr, key)
	return v
}
