package scene

import (
	"fmt"
	"github.com/23233/ggg/ut"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"regexp"
	"strings"
)

func CleanTypeName(typeName string) string {
	// 移除所有空的花括号
	reBraces := regexp.MustCompile(`\{\s*\}`)
	typeName = reBraces.ReplaceAllString(typeName, "")

	// 替换第一个出现的 [ 为 _
	typeName = strings.Replace(typeName, "[", "_", 1)

	// 移除所有剩余的 [ 和 ]
	reRemainingBrackets := regexp.MustCompile(`[\[\]]`)
	typeName = reRemainingBrackets.ReplaceAllString(typeName, "")

	// 去除字符串末尾的空格
	typeName = strings.TrimRight(typeName, " ")

	return typeName
}

func ParseInject(ctx iris.Context, contextInject ...ContextValueInject) ([]*ut.Kov, error) {
	result := make([]*ut.Kov, 0, len(contextInject))
	for _, inject := range contextInject {
		value := ctx.Values().Get(inject.FromKey)
		if value == nil && !inject.AllowEmpty {
			return nil, errors.New(fmt.Sprintf("%s key为空", inject.FromKey))
		}
		tk := inject.ToKey
		if len(tk) < 1 {
			tk = inject.FromKey
		}
		result = append(result, &ut.Kov{
			Key:   tk,
			Value: value,
		})
	}

	return result, nil
}
