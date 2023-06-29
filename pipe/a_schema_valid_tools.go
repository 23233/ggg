package pipe

import (
	"bytes"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// SchemaValidFunc 验证数据
func SchemaValidFunc(schema []byte, input map[string]any) error {

	compiler := jsonschema.NewCompiler()
	//compiler.Draft = jsonschema.Draft7

	err := compiler.AddResource("schema.json", bytes.NewReader(schema))
	if err != nil {
		return err
	}
	sch, err := compiler.Compile("schema.json")
	if err != nil {
		return err
	}

	// 在golang json规范中 数字都会序列化为float64 会存在精度丢失问题
	// 如果数字的位数大于 6 位，都会变成科学计数法，用到的地方都会受到影响。

	return sch.Validate(input)
}
