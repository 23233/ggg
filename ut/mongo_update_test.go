package ut

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"testing"
)

type InnerOne struct {
	City string `bson:"city"`
}

type InnerAnonymous struct {
	Name     string `bson:"name"`
	NickName string `bson:"nick_name"`
}

type Inline struct {
	Desc  string             `bson:"desc"`
	ObjId primitive.ObjectID `bson:"obj_id"`
}

type InlineName struct {
	InName string `bson:"in_name"`
	Desc   string `bson:"desc"`
}

type MainTypes struct {
	Inline `bson:",inline"`
	InnerAnonymous
	InlineName `bson:"inline_name,inline"`
	InnerOne   `bson:"inner_one"`
	Avatar     string `bson:"avatar"`
	NoBsonTag  string
	Anonymous  struct {
		Inline `bson:",inline"`
		InnerAnonymous
		InlineName `bson:"inline_name,inline"`
	} `bson:"anonymous"`
}

func TestStructToBsonM(t *testing.T) {
	converter := &StructConverter{}
	converter.AddCustomIsZero(new(primitive.ObjectID), func(v reflect.Value, field reflect.StructField) bool {
		return v.Interface() == primitive.NilObjectID
	})

	objID := primitive.NewObjectID()
	main := MainTypes{
		Inline: Inline{
			Desc:  "description",
			ObjId: objID,
		},
		InnerAnonymous: InnerAnonymous{
			Name:     "Alice",
			NickName: "A",
		},
		InlineName: InlineName{
			InName: "Inner Name",
			Desc:   "Inner Description",
		},
		InnerOne: InnerOne{
			City: "New York",
		},
		Avatar:    "avatar.png",
		NoBsonTag: "no bson tag",
		Anonymous: struct {
			Inline `bson:",inline"`
			InnerAnonymous
			InlineName `bson:"inline_name,inline"`
		}{
			Inline: Inline{
				Desc:  "anonymous description",
				ObjId: objID,
			},
			InnerAnonymous: InnerAnonymous{
				Name:     "Anonymous Alice",
				NickName: "AA",
			},
			InlineName: InlineName{
				InName: "Anonymous Inner Name",
				Desc:   "Anonymous Inner Description",
			},
		},
	}

	result, _ := converter.StructToBsonM(main)

	// 验证主要字段
	// 因为InlineName有inline标签 但是他排在Inline结构的后面有相同的desc字段 应该覆盖前面的desc字段
	if result["desc"] != "Inner Description" || result["obj_id"] != objID {
		t.Errorf("Unexpected values for inline fields")
	}
	if result["name"] != "Alice" || result["nick_name"] != "A" {
		t.Errorf("Unexpected values for InnerAnonymous fields")
	}
	if result["avatar"] != "avatar.png" {
		t.Errorf("Unexpected value for Avatar field")
	}
	if result["NoBsonTag"] != "no bson tag" {
		t.Errorf("Unexpected value for NoBsonTag field")
	}

	// 因为inline_name有inline标签 他不应该出现
	inlineName, ok := result["inline_name"].(bson.M)
	if ok || inlineName != nil {
		t.Errorf("inline标签解析失败 未提取到上一级中")
	}

	// 验证内部字段
	innerOne, ok := result["inner_one"].(bson.M)
	if !ok || innerOne["city"] != "New York" {
		t.Errorf("Unexpected values for InnerOne fields")
	}

	// 验证匿名字段
	anonymous, ok := result["anonymous"].(bson.M)
	if !ok {
		t.Errorf("Unexpected values for anonymous fields")
	}
	if anonymous["obj_id"] != objID || anonymous["desc"] != main.Anonymous.InlineName.Desc || anonymous["nick_name"] != main.Anonymous.InnerAnonymous.NickName {
		t.Errorf("Unexpected values for anonymous fields")
	}

}

func TestDiffToBsonM(t *testing.T) {
	converter := &StructConverter{}
	converter.AddCustomIsZero(new(primitive.ObjectID), func(v reflect.Value, field reflect.StructField) bool {
		return v.Interface() == primitive.NilObjectID
	})

	objID := primitive.NewObjectID()
	mainOriginal := MainTypes{
		Inline: Inline{
			Desc:  "original description",
			ObjId: objID,
		},
		InnerAnonymous: InnerAnonymous{
			Name:     "Original Alice",
			NickName: "OA",
		},
		InlineName: InlineName{
			InName: "Original Inner Name",
			Desc:   "Original Inner Description",
		},
		InnerOne: InnerOne{
			City: "Original City",
		},
		Avatar:    "original_avatar.png",
		NoBsonTag: "original no bson tag",
	}

	mainCurrent := MainTypes{
		Inline: Inline{
			Desc:  "current description",
			ObjId: objID,
		},
		InnerAnonymous: InnerAnonymous{
			Name:     "Current Alice",
			NickName: "CA",
		},
		InlineName: InlineName{
			InName: "Current Inner Name",
			Desc:   "Current Inner Description",
		},
		InnerOne: InnerOne{
			City: "Current City",
		},
		Avatar:    "current_avatar.png",
		NoBsonTag: "current no bson tag",
	}

	partCurrent := MainTypes{
		InlineName: InlineName{
			InName: "part Inner Name",
			Desc:   "part Inner Description",
		},
	}

	// 测试多层内联结构体
	result, err := converter.DiffToBsonM(mainOriginal, mainCurrent)
	if err != nil {
		t.Errorf("Error in DiffToBsonM: %v", err)
	}
	if result["desc"] != "Current Inner Description" || result["in_name"] != "Current Inner Name" {
		t.Errorf("Unexpected values for diff fields")
	}

	result, err = converter.DiffToBsonM(mainOriginal, partCurrent)
	if err != nil {
		t.Errorf("Error in DiffToBsonM: %v", err)
	}

	// 测试内联结构体的冻结字段
	result, err = converter.DiffToBsonM(mainOriginal, mainCurrent, "name")
	if err != nil {
		t.Errorf("Error in DiffToBsonM with freeze: %v", err)
	}
	if _, ok := result["name"]; ok {
		t.Errorf("Unexpected diff for frozen inline field")
	}

	// 测试冻结字段
	result, err = converter.DiffToBsonM(mainOriginal, mainCurrent, "avatar")
	if err != nil {
		t.Errorf("Error in DiffToBsonM with freeze: %v", err)
	}
	if _, ok := result["avatar"]; ok {
		t.Errorf("Unexpected diff for frozen field")
	}

	// 测试内联结构体的差异
	mainCurrent.InlineName.InName = "Original Inner Name" // 使内联结构体的一个字段与原始值相同
	result, err = converter.DiffToBsonM(mainOriginal, mainCurrent)
	if err != nil {
		t.Errorf("Error in DiffToBsonM: %v", err)
	}
	if _, ok := result["in_name"]; ok {
		t.Errorf("与内联结构原始值一致了则不应该出现")
	}

	// 测试基本字段的差异
	mainCurrent.Avatar = "original_avatar.png" // 使基本字段与原始值相同
	result, err = converter.DiffToBsonM(mainOriginal, mainCurrent)
	if err != nil {
		t.Errorf("Error in DiffToBsonM: %v", err)
	}
	if _, ok := result["avatar"]; ok {
		t.Errorf("Unexpected diff for unchanged basic field")
	}
}
