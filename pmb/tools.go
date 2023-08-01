package pmb

import (
	"fmt"
	"github.com/23233/ggg/ut"
	"reflect"
	"regexp"
	"strings"
)

func CheckConditions(data map[string]interface{}, conds []ut.Kov) (bool, string) {
	for _, cond := range conds {
		value, ok := data[cond.Key]
		if !ok {
			return false, fmt.Sprintf("键 %s 不存在", cond.Key)
		}
		switch cond.Op {
		case "eq":
			if cond.Value == nil && value != nil {
				return false, fmt.Sprintf("键 %s 的值不等于nil", cond.Key)
			}
			if cond.Value != nil && value == nil {
				return false, fmt.Sprintf("键 %s 的值等于nil", cond.Key)
			}
			if cond.Value != nil && !reflect.DeepEqual(value, cond.Value) {
				return false, fmt.Sprintf("键 %s 的值不等于%s", cond.Key, cond.Value)
			}
		case "ne":
			if cond.Value == nil && value == nil {
				return false, fmt.Sprintf("键 %s 的值等于nil", cond.Key)
			}
			if cond.Value != nil && reflect.DeepEqual(value, cond.Value) {
				return false, fmt.Sprintf("键 %s 的值等于%s", cond.Key, cond.Value)
			}
		case "gt":
			if reflect.ValueOf(value).Float() <= reflect.ValueOf(cond.Value).Float() {
				return false, fmt.Sprintf("键 %s 的值不大于%s", cond.Key, cond.Value)
			}
		case "gte":
			if reflect.ValueOf(value).Float() < reflect.ValueOf(cond.Value).Float() {
				return false, fmt.Sprintf("键 %s 的值不大于等于%s", cond.Key, cond.Value)
			}
		case "lt":
			if reflect.ValueOf(value).Float() >= reflect.ValueOf(cond.Value).Float() {
				return false, fmt.Sprintf("键 %s 的值不小于%s", cond.Key, cond.Value)
			}
		case "lte":
			if reflect.ValueOf(value).Float() > reflect.ValueOf(cond.Value).Float() {
				return false, fmt.Sprintf("键 %s 的值不小于等于%s", cond.Key, cond.Value)
			}
		case "in":
			if !strings.Contains(reflect.ValueOf(value).String(), reflect.ValueOf(cond.Value).String()) {
				return false, fmt.Sprintf("键 %s 的值不包含%s", cond.Key, cond.Value)
			}
		case "nin":
			if strings.Contains(reflect.ValueOf(value).String(), reflect.ValueOf(cond.Value).String()) {
				return false, fmt.Sprintf("键 %s 的值包含%s", cond.Key, cond.Value)
			}
		case "regex":
			match, _ := regexp.MatchString(reflect.ValueOf(cond.Value).String(), reflect.ValueOf(value).String())
			if !match {
				return false, fmt.Sprintf("键 %s 的值不匹配正则表达式", cond.Key)
			}
		default:
			return false, fmt.Sprintf("不支持的操作: %s", cond.Op)
		}
	}
	return true, "所有条件都满足"
}
