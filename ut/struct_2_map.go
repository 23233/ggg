package ut

import (
	jsoniter "github.com/json-iterator/go"
	"reflect"
)

func StructToMap(s any) (map[string]any, error) {
	data, err := jsoniter.Marshal(s)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	err = jsoniter.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	// 遍历map，删除值为nil或空map的键值对
	for k, v := range m {
		if v == nil {
			delete(m, k)
		} else if reflect.TypeOf(v).Kind() == reflect.Map {
			subMap, ok := v.(map[string]interface{})
			if ok && len(subMap) == 0 {
				delete(m, k)
			}
		}
	}

	return m, nil
}
