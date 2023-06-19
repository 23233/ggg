package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/google/go-cmp/cmp"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"time"
)

type ModelPutConfig struct {
	ModelId    string `json:"model_id,omitempty"`
	RowId      string `json:"row_id,omitempty"`
	UpdateTime bool   `json:"update_time,omitempty"`
}

func compareAndDiff(origin interface{}, bodyData map[string]interface{}, oldData map[string]interface{}) map[string]interface{} {
	diff := make(map[string]interface{})
	v := reflect.ValueOf(origin)

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			jsonTag := v.Type().Field(i).Tag.Get("json")
			if val, ok := bodyData[jsonTag]; ok {
				if oldVal, ok := oldData[jsonTag]; ok && !cmp.Equal(val, oldVal) {
					diff[jsonTag] = val
				}
			}
		}
	case reflect.Map:
		for key, val := range bodyData {
			if oldVal, ok := oldData[key]; ok && !cmp.Equal(val, oldVal) {
				diff[key] = val
			}
		}
	}

	return diff
}

// origin需要是一个map或struct
var (
	ModelPut = &RunnerContext[any, *ModelPutConfig, *qmgo.Database, map[string]any]{
		Key:  "model_ctx_put",
		Name: "模型单条修改",
		call: func(ctx iris.Context, origin any, params *ModelPutConfig, db *qmgo.Database, more ...any) *PipeRunResp[map[string]any] {
			bodyData := make(map[string]any)
			err := ctx.ReadBody(&bodyData)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			// 获取原始那一条
			var result = make(map[string]any)
			err = db.Collection(params.ModelId).Find(ctx, bson.M{ut.DefaultUidTag: params.RowId}).One(&result)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			// 如果不传origin则直接序列化body
			if origin == nil {
				origin = bodyData
			}

			diff := compareAndDiff(origin, bodyData, result)
			// 删除不允许变更的数据
			delete(diff, "_id")
			delete(diff, ut.DefaultUidTag)
			delete(diff, "update_time")
			delete(diff, "create_time")

			if len(diff) < 1 {
				return newPipeErr[map[string]any](errors.New("未获取到更新项"))
			}
			if params.UpdateTime {
				diff["update_time"] = time.Now().Local()
			}

			err = db.Collection(params.ModelId).UpdateOne(ctx, bson.M{ut.DefaultUidTag: params.RowId}, diff)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			return newPipeResult[map[string]any](diff)
		},
	}
)
