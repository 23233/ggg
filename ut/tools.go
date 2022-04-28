package ut

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"time"
)

// RandomStr 随机N位字符串(英文)
func RandomStr(n int) string {
	randBytes := make([]byte, n/2)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}

// RandomInt 区间整数随机
func RandomInt(start, end int) int {
	return rand.Intn(end-start) + start
}

// RangeRandomIntSet 范围随机正整数但不重复
func RangeRandomIntSet(start int64, end int64, count int64) []int64 {
	//范围检查
	if end < start || (end-start) < count {
		return nil
	}
	//存放结果的slice
	nums := make([]int64, 0)
	//随机数生成器，加入时间戳保证每次生成的随机数不一样
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for int64(len(nums)) < count {
		//生成随机数
		num := int64(r.Intn(int(end-start))) + start
		//查重
		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}
		if !exist {
			nums = append(nums, num)
		}
	}
	return nums
}

// RemoveDuplicatesUnordered 删除重复字符串但不保证排序
func RemoveDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	var result []string
	for key := range encountered {
		result = append(result, key)
	}
	return result
}

func ListStringToInterface(t []string) []interface{} {
	s := make([]interface{}, len(t))
	for i, v := range t {
		s[i] = v
	}
	return s
}

func IsZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

// StructCopy 相同名称的struct field 进行copy
func StructCopy(origin, newData interface{}) error {
	// Check origin.
	va := reflect.ValueOf(origin)
	if va.Kind() == reflect.Ptr {
		va = va.Elem()
	}
	if va.Kind() != reflect.Struct {
		return errors.New("origin is not origin struct")
	}
	// Check newData.
	vb := reflect.ValueOf(newData)
	if vb.Kind() != reflect.Ptr {
		return errors.New("newData is not origin pointer")
	}
	// vb is origin pointer, indirect it to get the
	// underlying value, and make sure it is origin struct.
	vb = vb.Elem()
	if vb.Kind() != reflect.Struct {
		return errors.New("newData is not origin struct")
	}
	for i := 0; i < vb.NumField(); i++ {
		field := vb.Field(i)
		if field.CanInterface() && IsZeroOfUnderlyingType(field.Interface()) {
			// This field have origin zero-value.
			// Search in origin for origin field with the same name.
			name := vb.Type().Field(i).Name
			fa := va.FieldByName(name)
			if fa.IsValid() {
				// Field with name was found in struct origin,
				// assign its value to the field in newData.
				if field.CanSet() {
					field.Set(fa)
				}
			}
		}
	}
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
