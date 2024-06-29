package ut

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// GenericSlice 是一个泛型类型，用于实现 GORM 的 Serializer 接口
// 解决sqlite mysql 不支持[]string 这种数组的问题 现在这样是数组解析成了 字符串的 a,b 这样加入db
type GenericSlice[T any] []T

func (gs *GenericSlice[T]) Scan(value interface{}) error {
	var st string
	switch value.(type) {
	case []uint8:
		st = string(value.([]uint8))
		break
	case string:
		st = value.(string)
	}

	if st == "" {
		*gs = []T{}
		return nil
	}
	parts := strings.Split(st, ",")
	var result []T
	for _, part := range parts {
		var item T
		_, _ = fmt.Sscanf(part, "%v", &item)
		result = append(result, item)
	}
	*gs = result
	return nil
}

// Value 实现了 driver.Valuer 接口
func (gs GenericSlice[T]) Value() (driver.Value, error) {
	var parts []string
	for _, item := range gs {
		parts = append(parts, fmt.Sprintf("%v", item))
	}
	return strings.Join(parts, ","), nil
}
