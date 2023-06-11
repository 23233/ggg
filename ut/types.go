package ut

import (
	"github.com/pkg/errors"
	"github.com/shockerli/cvt"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TypeChange 工具转换
func TypeChange(input any, wantType string) (any, error) {
	if len(wantType) >= 1 {
		switch wantType {
		case "objectId":
			is, err := cvt.StringE(input)
			if err != nil {
				return nil, err
			}
			objId, err := primitive.ObjectIDFromHex(is)
			return objId, err
		case "string":
			return cvt.StringE(input)
		case "uint":
			return cvt.UintE(input)
		case "uint8":
			return cvt.Uint8E(input)
		case "uint16":
			return cvt.Uint16E(input)
		case "uint32":
			return cvt.Uint32E(input)
		case "uint64":
			return cvt.Uint64E(input)
		case "int":
			return cvt.IntE(input)
		case "int8":
			return cvt.Int8E(input)
		case "int16":
			return cvt.Int16E(input)
		case "int32":
			return cvt.Int32E(input)
		case "int64":
			return cvt.Int64E(input)
		case "float32":
			return cvt.Float32E(input)
		case "float64":
			return cvt.Float64E(input)
		case "time":
			return cvt.TimeE(input)
		case "bool", "boolean":
			return cvt.BoolE(input)
			// 数组和map的转换 是否有必要?

		}
		return nil, errors.New(wantType + "类型未找到")
	}
	return input, nil

}
