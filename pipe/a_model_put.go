package pipe

import (
	"github.com/23233/ggg/ut"
	"github.com/google/go-cmp/cmp"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strings"
	"time"
)

type ModelPutConfig struct {
	QueryFilter *ut.QueryFull  `json:"query_filter,omitempty"`
	DropKeys    []string       `json:"drop_keys,omitempty"` // 最后的diff还需要丢弃的key
	ModelId     string         `json:"model_id,omitempty"`
	RowId       string         `json:"row_id,omitempty"`
	UpdateTime  bool           `json:"update_time,omitempty"`
	UpdateForce bool           `json:"update_force,omitempty"` // 强行覆盖
	BodyMap     map[string]any `json:"body_map,omitempty"`
}

func parseToTime(val interface{}) (time.Time, bool) {
	switch v := val.(type) {
	case time.Time:
		return v, true
	case primitive.DateTime:
		return v.Time(), true
	case string:
		// Try parsing in different formats
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
			t, err := time.Parse(layout, v)
			if err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// CompareAndDiff 对比不同 逻辑是出现在bodyData中的 然后origin与oldData不一致的 则返回一个diff Map
func CompareAndDiff(origin interface{}, bodyData map[string]interface{}, oldData map[string]interface{}, tagName string) map[string]interface{} {
	diff := make(map[string]interface{})
	v := reflect.Indirect(reflect.ValueOf(origin))

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if field.Type == reflect.TypeOf(ModelBase{}) {
				subDiff := CompareAndDiff(v.Field(i).Interface(), bodyData, oldData, tagName)
				for k, v := range subDiff {
					diff[k] = v
				}
			}
			jsonTag := field.Tag.Get(tagName)
			name := strings.Split(jsonTag, ",")[0]
			if name == "" {
				continue
			}
			if val, ok := bodyData[name]; ok {
				oldVal, ok := oldData[name]
				isDateTime := field.Type.Name() == "DateTime" && field.Type.PkgPath() == "go.mongodb.org/mongo-driver/bson/primitive"
				if field.Type == reflect.TypeOf(time.Time{}) || isDateTime {
					valTime, ok1 := parseToTime(val)
					oldValTime, ok2 := parseToTime(oldVal)
					valTime = valTime.Truncate(time.Second)
					oldValTime = oldValTime.Truncate(time.Second)
					if ok1 && ok2 {
						if !valTime.Equal(oldValTime) {
							diff[name] = val
						}
						continue
					}
				}
				if ok {
					if !cmp.Equal(val, oldVal) {
						diff[name] = val
					}
				} else {
					diff[name] = val
				}
			}
		}
	case reflect.Map:
		for key, val := range bodyData {
			oldVal, ok := oldData[key]
			valTime, ok1 := parseToTime(val)
			oldValTime, ok2 := parseToTime(oldVal)
			if ok {
				// 即使在map中对于time.Time也需要单独处理
				if ok1 && ok2 {
					valTime = valTime.Truncate(time.Second)
					oldValTime = oldValTime.Truncate(time.Second)
					if !valTime.Equal(oldValTime) {
						diff[key] = val
						continue // 跳过此键的后续处理
					}
				}
				// 判断是否一致
				if !cmp.Equal(val, oldVal) {
					diff[key] = val
				}
				continue
			}
			// 如果原始中不存在则直接加入 但是如果是时间则重制类型为 time.Time
			if ok1 {
				diff[key] = valTime
				continue
			}
			diff[key] = val

		}
	}

	return diff
}

var (
	// ModelPut 模型修改 origin需要是一个map或struct 只会修改与原始条目的diff项
	// 必传origin 可选struct或map 可以是ctx.ReadBody的映射
	// 必传params ModelPutConfig 其中ModelId和RowId必传
	// 必传db 为qmgo的Database
	ModelPut = &RunnerContext[any, *ModelPutConfig, *qmgo.Database, map[string]any]{
		Key:  "model_ctx_put",
		Name: "模型单条修改",
		call: func(ctx iris.Context, origin any, params *ModelPutConfig, db *qmgo.Database, more ...any) *RunResp[map[string]any] {
			if params == nil {
				return NewPipeErr[map[string]any](PipeParamsError)
			}
			if params.BodyMap == nil {
				return NewPipeErr[map[string]any](PipeOriginError)
			}
			bodyData := params.BodyMap

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
			err := db.Collection(params.ModelId).Aggregate(ctx, pipeline).One(&result)
			if err != nil {
				return NewPipeErr[map[string]any](err)
			}

			// 如果不传origin则直接序列化body
			// 这里有风险 会有不相关数据被序列化
			if origin == nil {
				origin = bodyData
			}

			diff := CompareAndDiff(origin, bodyData, result, "json")

			// 删除不允许变更的数据
			params.DropKeys = append(params.DropKeys, "_id", ut.DefaultUidTag)

			for _, key := range params.DropKeys {
				if _, ok := diff[key]; ok {
					delete(diff, key)
				}
			}

			if len(diff) < 1 {
				return NewPipeErr[map[string]any](errors.New("未获取到更新项"))
			}
			if params.UpdateTime {
				_, ok := diff["update_at"]
				if params.UpdateForce || !ok {
					diff["update_at"] = time.Now()
				}
			}

			err = db.Collection(params.ModelId).UpdateOne(ctx, bson.M{ut.DefaultUidTag: params.RowId}, bson.M{"$set": diff})
			if err != nil {
				return NewPipeErr[map[string]any](err)
			}

			return NewPipeResult[map[string]any](diff)
		},
	}
)
