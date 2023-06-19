package ut

import (
	"net/http"
	"strings"
	"testing"
)

func mapToStr(input map[string]string) string {
	var st = make([]string, 0)
	for k, v := range input {
		st = append(st, k+"="+v)

	}
	if len(st) > 0 {
		return strings.Join(st, "&")
	}
	return ""
}

func TestNewTrieMatch(t *testing.T) {
	mt := NewTrieMatch()

	type tCase struct {
		pattern string
		search  string
		not     bool
		method  string
		raw     string
		params  string
	}

	mts := []tCase{
		{
			pattern: "/a",
			raw:     "传入的内容get",
		},
		{
			pattern: "/a",
			method:  http.MethodPost,
			raw:     "传入的内容post",
		},
		{
			pattern: "/a",
			method:  http.MethodPut,
			raw:     "传入的内容put",
		},
		{
			pattern: "/a",
			method:  http.MethodDelete,
			raw:     "传入的内容delete",
		},
		{
			pattern: "/a/b",
			raw:     "ab",
		},
		{
			pattern: "/a/b/c",
			raw:     "abc",
		},
		{
			pattern: "/abbb/bccc/caaa",
			raw:     "aabbcc",
		},
		{
			pattern: "/a/t/:board",
			search:  "/a/t/123",
			raw:     "board",
			params:  "board=123",
		},
		{
			pattern: "/b/:board/:map",
			search:  "/b/123/321",
			raw:     "dynamicTwo",
			params:  "board=123&map=321",
		},
	}

	for _, m := range mts {
		method := m.method
		if len(method) < 1 {
			method = http.MethodGet
		}
		_ = mt.Add(m.pattern, method, m.raw)
	}

	for _, m := range mts {
		search := m.search
		if len(search) < 1 {
			search = m.pattern
		}
		method := m.method
		if len(method) < 1 {
			method = http.MethodGet
		}
		node, err := mt.Match(search, method)
		if err != nil {
			if !m.not {
				t.Fatalf("%s 应该不匹配但实际 %v", m.pattern, m.not)
				return
			}
		} else {
			if node.GetVal(method) != m.raw {
				t.Fatalf("%s raw 应该为%s 实际为 %s", m.pattern, m.raw, node.GetVal(method))
			}
			mpStr := mapToStr(node.GetMatchParams())
			if mpStr != m.params {
				t.Fatalf("%s matchParams 应该为%s 实际为 %s", m.pattern, m.params, mpStr)
			}
		}

	}

	// 测试删除方法
	mt.RemoveMethod("/a", http.MethodDelete)
	node, err := mt.Match("/a", http.MethodDelete)
	if err == nil || node != nil {
		t.Fatalf("测试删除失败")
	}

	// 测试删除路线
	mt.Remove("/a/b/c")
	node, err = mt.Match("/a/b/c", http.MethodGet)
	if err == nil || node != nil {
		t.Fatalf("测试删除节点失败")
	}

}
