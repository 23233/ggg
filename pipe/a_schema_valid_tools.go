package pipe

import (
	"bytes"
	"encoding/json"
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

	// 这里有一个关键 必须要重新序列化一下 因为在golang json规范中 数字都会序列化为float64 会存在精度丢失问题
	// 如果数字的位数大于 6 位，都会变成科学计数法，用到的地方都会受到影响。

	// https://github.com/santhosh-tekuri/jsonschema/issues/85 看这里 如果 验证器升级到了5.0.1以上版本 则可以去掉下面的序列化
	m, _ := json.Marshal(input)
	decoder := json.NewDecoder(bytes.NewReader(m))
	decoder.UseNumber()
	jsonNumberMap := make(map[string]any)
	_ = decoder.Decode(&jsonNumberMap)

	return sch.Validate(jsonNumberMap)
}
