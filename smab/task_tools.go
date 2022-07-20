package smab

import (
	"fmt"
	"math"
	"strings"
)

// TaskMarkdownState 简单化的一种markdown每一行数据渲染加载器

type TaskMarkdownState struct {
	Stage []string
}

func (u *TaskMarkdownState) Add(n string) {
	u.Stage = append(u.Stage, n)
}

func (u *TaskMarkdownState) Adds(layout string, inputs ...any) {
	u.Stage = append(u.Stage, fmt.Sprintf(layout, inputs...))
}

func (u *TaskMarkdownState) AddImg(href string, title string) {
	if len(href) < 1 {
		return
	}
	u.Add(fmt.Sprintf(`![%s](%s)`, title, href))
}

func (u *TaskMarkdownState) AddCodeStr(code string, languages ...string) {
	if len(code) < 1 {
		return
	}
	var defaultLanguage = "shell"
	if len(languages) >= 1 {
		defaultLanguage = languages[0]
	}
	u.Add(fmt.Sprintf("```%s \n %s ```", defaultLanguage, code))
}

func (u *TaskMarkdownState) AddTitle(text string, level uint8) {
	if len(text) < 1 {
		return
	}
	m := math.Min(6, float64(level))

	prefix := make([]string, 0, 6)
	for i := 0; i < int(m); i++ {
		prefix = append(prefix, "#")
	}

	u.Add(fmt.Sprintf("%s %s", strings.Join(prefix, ""), text))
}

func (u *TaskMarkdownState) GetStr(sep ...string) string {
	var s = ` \n `
	if len(sep) >= 1 {
		s = sep[0]
	}
	return strings.Join(u.Stage, s)
}
