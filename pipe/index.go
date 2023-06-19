package pipe

import (
	"encoding/json"
	"fmt"
	"github.com/23233/ggg/ut"
	"github.com/23233/user_agent"
	"github.com/kataras/iris/v12"
	"reflect"
	"strings"
)

func CtxGetEnv(ctx iris.Context, defaultStr ...string) string {
	referrer := ctx.GetReferrer()
	ua := ctx.GetHeader("User-Agent")

	k := user_agent.GetEnvKey(ua, referrer.Raw)
	if len(k) > 0 {
		return k
	}

	if len(defaultStr) >= 1 {
		return defaultStr[0]
	}
	return "default"
}

// OpValid 操作符验证
func OpValid(input any, op string, value string) bool {

	switch op {
	case "=":
	case "in":
	case "nin":
		ic, err := ut.TypeChange(input, "string")
		if err != nil {
			return false
		}
		vc, err := ut.TypeChange(value, "string")
		if err != nil {
			return false
		}
		inputStr := ic.(string)
		needStr := vc.(string)
		switch op {
		case "=":
			return inputStr == needStr
		case "in":
			nvList := strings.Split(value, ",")
			for _, v := range nvList {
				if v == inputStr {
					return true
				}
			}
			return false
		case "nin":
			nvList := strings.Split(value, ",")
			for _, v := range nvList {
				if v == inputStr {
					return false
				}
			}
			return true
		}
		return false
	case ">=":
	case ">":
	case "<=":
	case "<":

		ist, err := GetLen(input)
		if err != nil {
			return false
		}

		nst, err := GetLen(value)
		if err != nil {
			return false
		}
		switch op {
		case ">=":
			return ist >= nst
		case ">":
			return ist > nst
		case "<=":
			return ist <= nst
		case "<":
			return ist < nst
		}
		return false

	}
	return false
}

func MapToSliceString(input map[string]any) ([]string, error) {
	var st = make([]string, 0)
	for _, v := range input {
		vv, err := ut.TypeChange(v, "string")
		if err != nil {
			return nil, err
		}
		st = append(st, vv.(string))
	}
	return st, nil
}

func MapToSliceAny(input map[string]any) ([]any, error) {
	var st = make([]any, 0)
	for _, v := range input {
		st = append(st, v)
	}
	return st, nil
}

func FastMsgMap(code int, msg string, pipeNames ...string) iris.Map {
	mp := iris.Map{
		"code":   code,
		"detail": msg,
	}
	if len(pipeNames) > 0 {
		mp["pipe"] = pipeNames[0]
	}

	return mp
}

func GetLen(input any) (float64, error) {

	// 若非必要不要使用反射
	switch input.(type) {
	case string:
		return float64(len(input.(string))), nil
	case bool:
		if input.(bool) {
			return 1, nil
		}
		return 0, nil
	case int:
	case int8:
	case int16:
	case int32:
	case int64:
	case uint:
	case uint8:
	case uint16:
	case uint32:
	case uint64:
	case float32:
	case float64:
		break
	default:
		t := reflect.TypeOf(input)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		// 数组
		if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
			rv := reflect.Indirect(reflect.ValueOf(input))
			return float64(rv.Len()), nil
		}
		// struct或者map
		if t.Kind() == reflect.Struct || t.Kind() == reflect.Map {
			rv := reflect.Indirect(reflect.ValueOf(input))
			return float64(rv.NumField()), nil
		}
	}

	ic, err := ut.TypeChange(input, "float64")

	return ic.(float64), err

}

func ToMap(v interface{}) (map[string]interface{}, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid value specified, expected a struct but got %T", v)
	}
	m := make(map[string]interface{})
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Type().Field(i)
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			if tag == "-" {
				continue
			}
			name = strings.Split(tag, ",")[0]
		}
		value := rv.Field(i).Interface()
		if _, ok := value.(json.Marshaler); ok {
			jsonValue, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal field %s: %s", name, err)
			}
			var v interface{}
			if err := json.Unmarshal(jsonValue, &v); err != nil {
				return nil, fmt.Errorf("failed to unmarshal field %s: %s", name, err)
			}
			value = v
		}
		m[name] = value
	}
	return m, nil
}
