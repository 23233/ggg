package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/google/go-cmp/cmp"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"strings"
	"time"
)

type ModelPutConfig struct {
	QueryFilter *ut.QueryFull `json:"query_filter,omitempty"`
	DropKeys    []string      `json:"drop_keys,omitempty"` // 最后的diff还需要丢弃的key
	ModelId     string        `json:"model_id,omitempty"`
	RowId       string        `json:"row_id,omitempty"`
	UpdateTime  bool          `json:"update_time,omitempty"`
}

func compareAndDiff(origin interface{}, bodyData map[string]interface{}, oldData map[string]interface{}) map[string]interface{} {
	diff := make(map[string]interface{})
	v := reflect.Indirect(reflect.ValueOf(origin))

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			jsonTag := v.Type().Field(i).Tag.Get("json")
			name := strings.Split(jsonTag, ",")[0]
			if name == "" {
				continue
			}
			if val, ok := bodyData[name]; ok {
				oldVal, ok := oldData[name]
				// 原始数据中存在这个条目
				if ok {
					// 判断原始数据的条目值和当前是否一致 不一致则加入diff
					if !cmp.Equal(val, oldVal) {
						diff[name] = val
					}
				} else {
					// 原始数据中不存在 那么就新增
					diff[name] = val
				}
			}
		}
	case reflect.Map:
		for key, val := range bodyData {

			oldVal, ok := oldData[key]
			if ok {
				if !cmp.Equal(val, oldVal) {
					diff[key] = val
				}
			} else {
				// 原始数据中不存在 那么就新增
				diff[key] = val
			}

		}
	}

	return diff
}

var (
	// ModelPut 模型修改 origin需要是一个map或struct 只会修改与原始条目的diff项
	ModelPut = &RunnerContext[any, *ModelPutConfig, *qmgo.Database, map[string]any]{
		Key:  "model_ctx_put",
		Name: "模型单条修改",
		call: func(ctx iris.Context, origin any, params *ModelPutConfig, db *qmgo.Database, more ...any) *RunResp[map[string]any] {
			bodyData := make(map[string]any)
			err := ctx.ReadBody(&bodyData)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			if params == nil {
				return newPipeErr[map[string]any](PipeParamsError)
			}

			ft := params.QueryFilter
			if ft == nil {
				ft = new(ut.QueryFull)
			}

			if ft.QueryParse == nil {
				ft.QueryParse = new(ut.QueryParse)
			}

			if ft.QueryParse.And == nil {
				ft.QueryParse.And = make([]*ut.Kov, 0)
			}
			if ft.QueryParse.Or == nil {
				ft.QueryParse.Or = make([]*ut.Kov, 0)
			}
			ft.QueryParse.And = append(ft.QueryParse.And, &ut.Kov{
				Key:   ut.DefaultUidTag,
				Value: params.RowId,
			})

			pipeline := ut.QueryToMongoPipeline(ft)

			// 获取原始那一条
			var result = make(map[string]any)
			err = db.Collection(params.ModelId).Aggregate(ctx, pipeline).One(&result)
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			// 如果不传origin则直接序列化body
			// 这里有风险 会有不相关数据被序列化
			if origin == nil {
				origin = bodyData
			}

			diff := compareAndDiff(origin, bodyData, result)

			// 删除不允许变更的数据
			params.DropKeys = append(params.DropKeys, "_id", ut.DefaultUidTag, "update_at", "create_at")

			for _, key := range params.DropKeys {
				if _, ok := diff[key]; ok {
					delete(diff, key)
				}
			}

			if len(diff) < 1 {
				return newPipeErr[map[string]any](errors.New("未获取到更新项"))
			}
			if params.UpdateTime {
				diff["update_at"] = time.Now().Local()
			}

			err = db.Collection(params.ModelId).UpdateOne(ctx, bson.M{ut.DefaultUidTag: params.RowId}, bson.M{"$set": diff})
			if err != nil {
				return newPipeErr[map[string]any](err)
			}

			return newPipeResult[map[string]any](diff)
		},
	}
)
