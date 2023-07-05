package pmb

import (
	"fmt"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/23233/jsonschema"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
)

func IrisRespErr(msg string, err error, ctx iris.Context, code ...int) {
	//logger.J.ErrEf(err, msg)
	if len(code) >= 1 {
		ctx.StatusCode(code[0])
	} else {
		ctx.StatusCode(iris.StatusBadRequest)
	}
	if len(msg) >= 1 {
		ctx.JSON(iris.Map{"detail": msg})
	} else {
		ctx.JSON(iris.Map{"detail": err.Error()})
	}
	return
}

type SchemaGetResp struct {
	*ut.MongoFacetResult
	Page     int64          `json:"page,omitempty"`
	PageSize int64          `json:"page_size,omitempty"`
	Filters  *ut.QueryParse `json:"filters,omitempty"`
}

type ContextValueInject struct {
	FromKey    string `json:"from_key,omitempty"`
	ToKey      string `json:"to_key,omitempty"`
	AllowEmpty bool   `json:"allow_empty,omitempty"`
}

type ActionPostPart struct {
	Name     string         `json:"name"`                // action名称
	Rows     []string       `json:"rows,omitempty"`      // 选中的行id
	FormData map[string]any `json:"form_data,omitempty"` // 表单填写的值
}

type SchemaModelAction struct {
	Name  string                                                                              `json:"name,omitempty"`  // 动作名称 需要唯一
	Types []uint                                                                              `json:"types,omitempty"` // 0 表可用 1 行可用
	Form  *jsonschema.Schema                                                                  `json:"form,omitempty"`  // 若form为nil 则不会弹出表单填写
	call  func(ctx iris.Context, rows []map[string]any, formData map[string]any) (any, error) // 处理方法 result 只能返回map或struct
}

func (s *SchemaModelAction) SetForm(raw any) {
	schema := new(jsonschema.Reflector)
	// 默认为true是所有存在的字段均会被标记到required
	// 只要为标记为omitempty的都会进入required
	schema.RequiredFromJSONSchemaTags = false
	// 为true 则会写入Properties 对于object会写入$defs 生成$ref引用
	schema.ExpandedStruct = true
	ref := schema.Reflect(raw)
	s.Form = ref
}

type SchemaModel[T any] struct {
	db       *qmgo.Database
	raw      T
	Schema   *jsonschema.Schema   `json:"schema,omitempty"`
	Group    string               `json:"group,omitempty"`    // 组名
	Priority int                  `json:"priority,omitempty"` // 在组下显示的优先级 越大越优先
	EngName  string               `json:"eng_name,omitempty"` // 英文名 表名
	Alias    string               `json:"alias,omitempty"`    // 别名 中文名
	Actions  []*SchemaModelAction `json:"actions,omitempty"`  // 各类操作
	// 每个查询都注入的内容 从context中去获取 可用于获取用户id等操作
	QueryInjects []ContextValueInject `json:"query_injects,omitempty"`
	WriteInsert  bool                 `json:"write_insert,omitempty"` // 是否把注入内容写入新增体
	PostMustKeys []string             `json:"post_must_keys,omitempty"`
	// 过滤参数能否通过 这里能注入和修改过滤参数和判断参数是否缺失 返回错误则抛出错误
	filterCanPass func(ctx iris.Context, query *ut.QueryFull) error
	uidParamsKey  string
}

func NewSchemaModel[T any](raw T, db *qmgo.Database) *SchemaModel[T] {
	var r = &SchemaModel[T]{
		db:           db,
		Group:        "默认",
		Actions:      make([]*SchemaModelAction, 0),
		uidParamsKey: "uid",
	}
	r.SetRaw(raw)
	return r
}

func (s *SchemaModel[T]) SetRaw(raw T) {
	schema := new(jsonschema.Reflector)
	// 默认为true是所有存在的字段均会被标记到required
	// 只要为标记为omitempty的都会进入required
	schema.RequiredFromJSONSchemaTags = false
	// 为true 则会写入Properties 对于object会写入$defs 生成$ref引用
	schema.ExpandedStruct = true
	ref := schema.Reflect(raw)
	s.raw = raw
	s.Schema = ref
	// 通过反射获取struct的名称 不包含包名

	typ := reflect.TypeOf(raw)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	name := typ.Name()

	s.EngName = name
}

func (s *SchemaModel[T]) AddAction(action *SchemaModelAction) {
	s.Actions = append(s.Actions, action)
}

func (s *SchemaModel[T]) SetFilterCanPass(filterCanPass func(ctx iris.Context, query *ut.QueryFull) error) {
	s.filterCanPass = filterCanPass
}

func (s *SchemaModel[T]) ParseInject(ctx iris.Context) ([]*ut.Kov, error) {
	result := make([]*ut.Kov, 0, len(s.QueryInjects))
	for _, inject := range s.QueryInjects {
		value := ctx.Values().Get(inject.FromKey)
		if value == nil && !inject.AllowEmpty {
			return nil, errors.New(fmt.Sprintf("%s key为空", inject.FromKey))
		}
		tk := inject.ToKey
		if len(tk) < 1 {
			tk = inject.FromKey
		}
		result = append(result, &ut.Kov{
			Key:   tk,
			Value: value,
		})
	}
	return result, nil
}

func (s *SchemaModel[T]) ToAny() *SchemaModel[any] {
	var b = new(SchemaModel[any])
	b.Group = s.Group
	b.Alias = s.Alias
	b.Actions = s.Actions
	b.EngName = s.EngName
	b.Schema = s.Schema
	b.Priority = s.Priority
	b.QueryInjects = s.QueryInjects
	b.WriteInsert = s.WriteInsert
	b.filterCanPass = s.filterCanPass
	b.raw = s.raw
	b.db = s.db
	return b
}

// GetHandler 仅获取数据
func (s *SchemaModel[T]) GetHandler(ctx iris.Context, queryParams pipe.QueryParseConfig, getParams pipe.ModelGetData, uid string) error {
	injectQuery, err := s.ParseInject(ctx)
	if err != nil {
		return err
	}

	if getParams.Single {
		if len(uid) < 1 {
			uid, err = s.getUid(ctx)
			if err != nil {
				return err
			}
		}
		if queryParams.InjectAnd == nil {
			queryParams.InjectAnd = make([]*ut.Kov, 0)
		}
		queryParams.InjectAnd = append(queryParams.InjectAnd, &ut.Kov{
			Key:   ut.DefaultUidTag,
			Value: uid,
		})
	}

	queryParams.InjectAnd = append(queryParams.InjectAnd, injectQuery...)

	// 解析query
	resp := pipe.QueryParse.Run(ctx, nil, &queryParams, nil)
	if resp.Err != nil {
		return resp.Err
	}
	// 过滤参数 外键什么的可以在这里注入
	if s.filterCanPass != nil {
		err = s.filterCanPass(ctx, resp.Result)
		if err != nil {
			return err
		}
	}

	if !getParams.Single {
		if resp.Result.Page < 1 {
			resp.Result.Page = 1
		}
		if resp.Result.Page > 100 {
			resp.Result.Page = 100
		}
		if resp.Result.PageSize <= 0 {
			resp.Result.PageSize = 10
		}
		if resp.Result.PageSize > 100 {
			resp.Result.PageSize = 100
		}
		// 默认按照更新时间倒序
		if resp.Result.SortDesc == nil {
			resp.Result.SortDesc = append(resp.Result.SortDesc, "update_time")
		}
	}

	// 获取数据
	dataResp := pipe.QueryGetData.Run(ctx,
		&pipe.ModelGetDataDep{
			ModelId: s.EngName,
			Query:   resp.Result,
		},
		&getParams,
		s.db)
	if dataResp.Err != nil {
		return dataResp.Err
	}

	if getParams.Single {
		// 未获取到
		if dataResp.Result.Data == nil {
			return errors.New("获取单条数据失败")
		}
		ctx.JSON(dataResp.Result.Data)

		return nil
	}

	var result = new(SchemaGetResp)
	result.MongoFacetResult = dataResp.Result
	result.Page = resp.Result.Page
	result.PageSize = resp.Result.PageSize
	result.Filters = resp.Result.QueryParse

	ctx.JSON(result)
	return nil

}

func checkKeys(keys []string, raw interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(raw))
	switch v.Kind() {
	case reflect.Struct:
		for _, key := range keys {
			found := false
			for i := 0; i < v.NumField(); i++ {
				tag := v.Type().Field(i).Tag.Get("json")
				if tag == key {
					found = true
					break
				}
			}
			if !found {
				return errors.New(fmt.Sprintf("key %s does not exist in the body", key))
			}
		}
	case reflect.Map:
		body, _ := raw.(map[string]interface{})
		for _, key := range keys {
			if _, ok := body[key]; !ok {
				return errors.New(fmt.Sprintf("key %s does not exist in the body", key))
			}
		}
	default:
		return errors.New("unsupported type, must be a struct or map[string]interface{}")
	}
	return nil
}

func (s *SchemaModel[T]) newRaw() any {
	typ := reflect.TypeOf(s.raw)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	newV := reflect.New(typ).Interface()
	return newV
}

// PostHandler 新增数据
func (s *SchemaModel[T]) PostHandler(ctx iris.Context, params pipe.ModelCtxMapperPack) error {

	if s.WriteInsert {
		injectQuery, err := s.ParseInject(ctx)
		if err != nil {
			return err
		}
		for _, kov := range injectQuery {
			params.InjectData[kov.Key] = kov.Value
		}
	}

	newV := s.newRaw()

	// 通过模型去序列化body 可以防止一些无效的数据注入
	resp := pipe.ModelMapper.Run(ctx, newV, &params, nil)
	if resp.Err != nil {
		return resp.Err
	}

	// 必须出现在body中的字段名
	if s.PostMustKeys != nil {
		err := checkKeys(s.PostMustKeys, resp.Result)
		if err != nil {
			return err
		}
	}

	// 进行新增
	insertResult := pipe.ModelAdd.Run(ctx, resp.Result, &pipe.ModelCtxAddConfig{ModelId: s.EngName}, s.db)
	if insertResult.Err != nil {
		return insertResult.Err
	}

	ctx.JSON(insertResult.Result)

	return nil
}

func (s *SchemaModel[T]) PutHandler(ctx iris.Context, params pipe.ModelPutConfig) error {
	injectQuery, err := s.ParseInject(ctx)
	if err != nil {
		return err
	}

	if len(params.RowId) < 1 {
		uid, err := s.getUid(ctx)
		if err != nil {
			return err
		}
		params.RowId = uid
	}

	if params.QueryFilter == nil {
		params.QueryFilter = new(ut.QueryFull)
	}
	params.QueryFilter.QueryParse.InsertOrReplaces("and", injectQuery...)

	params.ModelId = s.EngName

	newV := s.newRaw()
	err = ctx.ReadBody(&newV)
	if err != nil {
		return err
	}

	resp := pipe.ModelPut.Run(ctx, newV, &params, s.db)
	if resp.Err != nil {
		return resp.Err
	}
	ctx.JSON(resp.Result)
	return nil
}

func (s *SchemaModel[T]) DelHandler(ctx iris.Context, params pipe.ModelDelConfig) error {
	params.ModelId = s.EngName

	if len(params.RowId) < 1 {

		uid, err := s.getUid(ctx)
		if err != nil {
			return err
		}
		params.RowId = uid
	}

	injectQuery, err := s.ParseInject(ctx)
	if err != nil {
		return err
	}
	if params.QueryFilter == nil {
		params.QueryFilter = new(ut.QueryFull)
	}
	if params.QueryFilter.QueryParse == nil {
		params.QueryFilter.QueryParse = new(ut.QueryParse)
	}

	params.QueryFilter.QueryParse.InsertOrReplaces("and", injectQuery...)

	resp := pipe.ModelDel.Run(ctx, nil, &params, s.db)
	if resp.Err != nil {
		return resp.Err
	}
	_, _ = ctx.WriteString(resp.Result.(string))
	return nil
}

func (s *SchemaModel[T]) getUid(ctx iris.Context) (string, error) {
	uid := ctx.Params().Get(s.uidParamsKey)
	if len(uid) < 1 {
		return "", errors.New("获取行id失败")
	}
	if len(uid) < 1 {
		return "", errors.New("行id为空")
	}
	return uid, nil
}

func (s *SchemaModel[T]) ActionEntry(ctx iris.Context) {
	// 必须为post
	part := new(ActionPostPart)
	err := ctx.ReadBody(&part)
	if err != nil {
		IrisRespErr("解构action参数包失败", err, ctx)
		return
	}

	var action *SchemaModelAction
	for _, ac := range s.Actions {
		if ac.Name == part.Name {
			action = ac
			break
		}
	}

	if action == nil {
		IrisRespErr("未找到对应action", nil, ctx)
		return
	}
	if action.call == nil {
		IrisRespErr("action未设置执行方法", nil, ctx)
		return
	}

	rows := make([]map[string]any, 0, len(part.Rows))
	if len(part.Rows) >= 1 {
		// 去获取出最新的这一批数据
		err = s.db.Collection(s.EngName).Find(ctx, bson.M{ut.DefaultUidTag: bson.M{"$in": part.Rows}}).All(&rows)
		if err != nil {
			IrisRespErr("获取对应行列表失败", err, ctx)
			return
		}
	}

	result, err := action.call(ctx, rows, part.FormData)
	if err != nil {
		IrisRespErr("", err, ctx)
		return
	}

	if result != nil {
		_ = ctx.JSON(result)
	} else {
		_, _ = ctx.WriteString("ok")
	}

}

func (s *SchemaModel[T]) Registry(part iris.Party) {
	p := part.Party("/" + s.EngName)
	s.RegistryConfigAction(p)
	s.RegistryCrud(p)
}

func (s *SchemaModel[T]) RegistryConfigAction(p iris.Party) {
	// 获取配置文件
	p.Get("/config", func(ctx iris.Context) {
		_ = ctx.JSON(iris.Map{
			"schema":  s.Schema,
			"actions": s.Actions,
		})
		return
	})
	// action
	p.Post("/action", s.ActionEntry)
}

func (s *SchemaModel[T]) RegistryCrud(p iris.Party) {
	p.Get("/", func(ctx iris.Context) {
		err := s.GetHandler(ctx, pipe.QueryParseConfig{}, pipe.ModelGetData{
			Single:        false,
			GetQueryCount: true,
		}, "")
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	p.Get("/{uid:string}", func(ctx iris.Context) {
		err := s.GetHandler(ctx, pipe.QueryParseConfig{}, pipe.ModelGetData{
			Single: true,
		}, "")
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	p.Post("/", func(ctx iris.Context) {
		err := s.PostHandler(ctx, pipe.ModelCtxMapperPack{})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	p.Put("/{uid:string}", func(ctx iris.Context) {
		err := s.PutHandler(ctx, pipe.ModelPutConfig{
			UpdateTime: true,
		})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
	p.Delete("/{uid:string}", func(ctx iris.Context) {
		err := s.DelHandler(ctx, pipe.ModelDelConfig{})
		if err != nil {
			IrisRespErr("", err, ctx)
			return
		}
	})
}
