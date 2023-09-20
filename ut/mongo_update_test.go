package ut

import (
	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"testing"
	"time"
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
		Tm         time.Time   `bson:"tm"`
		SliceStr   []string    `bson:"slice_str"`
		SliceTime  []time.Time `bson:"slice_time"`
	} `bson:"anonymous"`
	Tm        time.Time   `bson:"tm"`
	SliceStr  []string    `bson:"slice_str"`
	SliceTime []time.Time `bson:"slice_time"`
	KeepBool  bool        `bson:"keep_bool"`
	OmitStr   string      `bson:"omit_str"`
}

func TestStructToBsonM(t *testing.T) {
	converter := NewStructConverter()
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
			Tm         time.Time   `bson:"tm"`
			SliceStr   []string    `bson:"slice_str"`
			SliceTime  []time.Time `bson:"slice_time"`
		}{
			Inline: Inline{
				Desc:  "anonymous description",
				ObjId: objID,
			},
			InnerAnonymous: InnerAnonymous{
				Name:     "Anonymous Alice",
				NickName: "AAA",
			},
			InlineName: InlineName{
				InName: "Anonymous Inner Name",
				Desc:   "Anonymous Inner Description",
			},
			Tm:        time.Now(),
			SliceStr:  []string{"222"},
			SliceTime: []time.Time{time.Now()},
		},
		Tm:        time.Now(),
		SliceStr:  []string{"111"},
		SliceTime: []time.Time{time.Now()},
	}

	result, _ := converter.StructToBsonM(main, []string{"omit_str"}, []string{"keep_bool"})

	expectBson := bson.M{
		"desc":           "Inner Description",
		"obj_id":         objID,
		"name":           "Alice",
		"nick_name":      "A",
		"in_name":        "Inner Name",
		"inner_one.city": "New York",
		"avatar":         "avatar.png",
		"NoBsonTag":      "no bson tag",
		"tm":             main.Tm,
		"slice_str":      main.SliceStr,
		"slice_time":     main.SliceTime,
		"anonymous": bson.M{
			"name":       main.Anonymous.Name,
			"in_name":    main.Anonymous.InName,
			"obj_id":     objID,
			"desc":       main.Anonymous.InlineName.Desc,
			"nick_name":  main.Anonymous.InnerAnonymous.NickName,
			"slice_str":  main.Anonymous.SliceStr,
			"slice_time": main.Anonymous.SliceTime,
		},
		"keep_bool": false,
	}

	for k, v := range expectBson {

		if vb, ok := v.(bson.M); ok {
			for ik, iv := range vb {
				vvv, ok := result[k+"."+ik]
				if !ok {
					t.Errorf("%s %s 值不存在", k, ik)
					continue
				}
				same := cmp.Equal(iv, vvv)
				if !same {
					t.Errorf("%s %s 的值%v 与预期%v不一致", k, ik, iv, vvv)
				}
			}
			continue
		}

		vv, ok := result[k]
		if !ok {
			t.Errorf("%s 值不存在", k)
			continue
		}

		same := cmp.Equal(v, vv)
		if !same {
			t.Errorf("%s 的值%v 与预期%v不一致", k, v, vv)
		}

	}

}

func TestDiffToBsonM(t *testing.T) {
	converter := NewStructConverter()
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
		Tm:        time.Now(),
		SliceStr:  []string{"111"},
		SliceTime: []time.Time{time.Now()},
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

	var sliceTime []time.Time
	var sliceStr []string

	expectBson := bson.M{
		// 因为obj_id一致 所以不会出现
		//"obj_id":objID,
		"name":           "Current Alice",
		"desc":           "Current Inner Description",
		"nick_name":      "CA",
		"in_name":        "Current Inner Name",
		"inner_one.city": "Current City",
		"avatar":         "current_avatar.png",
		"NoBsonTag":      "current no bson tag",
		"slice_time":     any(sliceTime),
		"slice_str":      any(sliceStr),
	}

	partCurrent := MainTypes{
		InlineName: InlineName{
			InName: "part Inner Name",
			Desc:   "part Inner Description",
		},
	}

	// 测试多层内联结构体
	result, err := converter.DiffToBsonM(mainOriginal, mainCurrent, nil, nil)
	if err != nil {
		t.Errorf("Error in DiffToBsonM: %v", err)
	}

	for k, v := range expectBson {
		vv := result[k]
		same := cmp.Equal(v, vv)
		if !same {
			t.Errorf("%s 的值%v 与预期%v不一致", k, v, vv)
		}
	}

	result, err = converter.DiffToBsonM(mainOriginal, partCurrent, nil, nil)
	if err != nil {
		t.Errorf("Error in DiffToBsonM: %v", err)
	}
	if result["in_name"] != partCurrent.InlineName.InName || result["desc"] != partCurrent.InlineName.Desc {
		t.Errorf("内联修改获取失败")
	}

	// 测试内联结构体的跳过字段
	result, err = converter.DiffToBsonM(mainOriginal, mainCurrent, []string{"name"}, nil)
	if err != nil {
		t.Errorf("Error in DiffToBsonM with skip: %v", err)
	}
	if _, ok := result["name"]; ok {
		t.Errorf("Unexpected diff for frozen inline field")
	}

	// 测试跳过字段
	result, err = converter.DiffToBsonM(mainOriginal, mainCurrent, []string{"avatar"}, nil)
	if err != nil {
		t.Errorf("Error in DiffToBsonM with skip: %v", err)
	}
	if _, ok := result["avatar"]; ok {
		t.Errorf("Unexpected diff for frozen field")
	}

	// 测试内联结构体的差异
	mainCurrent.InlineName.InName = "Original Inner Name" // 使内联结构体的一个字段与原始值相同
	result, err = converter.DiffToBsonM(mainOriginal, mainCurrent, nil, nil)
	if err != nil {
		t.Errorf("Error in DiffToBsonM: %v", err)
	}
	if _, ok := result["in_name"]; ok {
		t.Errorf("与内联结构原始值一致了则不应该出现")
	}

	// 测试基本字段的差异
	mainCurrent.Avatar = "original_avatar.png" // 使基本字段与原始值相同
	result, err = converter.DiffToBsonM(mainOriginal, mainCurrent, nil, nil)
	if err != nil {
		t.Errorf("Error in DiffToBsonM: %v", err)
	}
	if _, ok := result["avatar"]; ok {
		t.Errorf("Unexpected diff for unchanged basic field")
	}
}
