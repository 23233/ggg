package pmb

import (
	"encoding/json"
	"fmt"
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/23233/jsonschema"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"time"
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
	Sorts    *ut.BaseSort   `json:"sorts,omitempty"`
	Filters  *ut.QueryParse `json:"filters,omitempty"`
}

type ContextValueInject struct {
	FromKey    string `json:"from_key,omitempty"`
	ToKey      string `json:"to_key,omitempty"`
	AllowEmpty bool   `json:"allow_empty,omitempty"`
}

type ActionPostPart[F any] struct {
	Name     string   `json:"name"`                // action名称
	Rows     []string `json:"rows,omitempty"`      // 选中的行id
	FormData F        `json:"form_data,omitempty"` // 表单填写的值
}

// ActionPostArgs T是行数据 F是表单数据
type ActionPostArgs[T, F any] struct {
	Rows     []T               `json:"rows"`
	FormData F                 `json:"form_data"`
	User     *SimpleUserModel  `json:"user"`
	Model    *SchemaModel[any] `json:"model"`
}

type SchemaActionBase struct {
	Name       string             `json:"name,omitempty"`        // 动作名称 需要唯一
	Prefix     string             `json:"prefix,omitempty"`      // 前缀标识 仅展示用
	Types      []uint             `json:"types,omitempty"`       // 0 表可用 1 行可用
	Form       *jsonschema.Schema `json:"form,omitempty"`        // 若form为nil 则不会弹出表单填写
	MustSelect bool               `json:"must_select,omitempty"` // 必须有所选择表选择适用 行是必须选一行
	Conditions []ut.Kov           `json:"conditions,omitempty"`  // 选中/执行的前置条件 判断数据为选中的每一行数据 常用场景为 限定只有字段a=b时才可用或a!=b时 挨个执行 任意一个不成功都返回
}

func (s *SchemaActionBase) SetType(tp []uint) {
	s.Types = tp
}
func (s *SchemaActionBase) SetForm(raw any) {
	s.Form = ToJsonSchema(raw)
}
func (s *SchemaActionBase) AddCondition(cond ut.Kov) {
	s.Conditions = append(s.Conditions, cond)
}

type ISchemaAction interface {
	Execute(ctx iris.Context, args any) (responseInfo any, err error)
	GetBase() *SchemaActionBase
	SetCall(func(ctx iris.Context, args any) (responseInfo any, err error))
}

func ActionParseArgs[T any, F any](ctx iris.Context, model *SchemaModel[any]) (*ActionPostPart[F], []T, ISchemaAction, error) {
	// 必须为post
	part := new(ActionPostPart[F])
	err := ctx.ReadBody(&part)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "解构action参数包失败")
	}

	var action ISchemaAction
	for _, ac := range model.Actions {
		if ac.GetBase().Name == part.Name {
			action = ac
			break
		}
	}

	if action == nil {
		return nil, nil, nil, errors.Wrap(err, "未找到对应action")
	}

	rows := make([]T, 0, len(part.Rows))
	if len(part.Rows) >= 1 {
		// 去获取出最新的这一批数据
		err = model.GetCollection().Find(ctx, bson.M{ut.DefaultUidTag: bson.M{"$in": part.Rows}}).All(&rows)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "获取对应行列表失败")
		}
	}
	return part, rows, action, nil
}

func ActionRun[T any, F any](ctx iris.Context, model *SchemaModel[any], user *SimpleUserModel) {
	part, rows, action, err := ActionParseArgs[T, F](ctx, model)
	args := new(ActionPostArgs[T, F])
	args.Rows = rows
	args.FormData = part.FormData
	args.User = user
	args.Model = model

	result, err := action.Execute(ctx, args)
	if err != nil {
		IrisRespErr("", err, ctx)
		return
	}

	if result != nil {
		_ = ctx.JSON(result)
		return
	}
	_, _ = ctx.WriteString("ok")

}

// SchemaModelAction 模型action T是模型数据 F是表单数据
type SchemaModelAction[T, F any] struct {
	SchemaActionBase
	call func(ctx iris.Context, args any) (responseInfo any, err error)
}

func (s *SchemaModelAction[T, F]) GetBase() *SchemaActionBase {
	return &s.SchemaActionBase
}
func (s *SchemaModelAction[T, F]) Execute(ctx iris.Context, args any) (responseInfo any, err error) {
	if s.call == nil {
		return nil, errors.New("action未定义默认执行函数")
	}
	return s.call(ctx, args)
}
func (s *SchemaModelAction[T, F]) SetCall(call func(ctx iris.Context, args any) (responseInfo any, err error)) {
	s.call = call
}

func NewSchemaModelAction[T, F any]() *SchemaModelAction[T, F] {
	inst := new(SchemaModelAction[T, F])
	return inst
}

func NewRowAction[T any, F any](name string, form F) ISchemaAction {
	inst := NewSchemaModelAction[T, F]()
	inst.Types = []uint{1}
	inst.Name = name

	val := reflect.ValueOf(form)
	if val.Kind() == reflect.Ptr {
		if !val.IsNil() {
			inst.SetForm(form)
		}
	} else if val.IsValid() && !val.IsZero() {
		inst.SetForm(form)
	}
	inst.Conditions = make([]ut.Kov, 0)
	return inst
}
func NewTableAction[T any, F any](name string, form F) ISchemaAction {
	inst := NewRowAction[T, F](name, form)
	inst.GetBase().SetType([]uint{0})
	return inst
}

// NewAction action的名称必填 form没有可传nil
func NewAction[T any, F any](name string, form F) ISchemaAction {
	inst := NewRowAction[T, F](name, form)
	inst.GetBase().SetType([]uint{0, 1})
	return inst
}

type SchemaBase struct {
	Group    string `json:"group,omitempty"`    // 组名
	Priority int    `json:"priority,omitempty"` // 在组下显示的优先级 越大越优先
	EngName  string `json:"eng_name,omitempty"` // 英文名 表名
	RawName  string `json:"raw_name,omitempty"` // 原始名称
	Alias    string `json:"alias,omitempty"`    // 别名 中文名
}

type SchemaModel[T any] struct {
	SchemaBase
	db      *qmgo.Database
	raw     T
	Schema  *jsonschema.Schema `json:"schema,omitempty"`
	Actions []ISchemaAction    `json:"actions,omitempty"` // 各类操作
	// 每个查询都注入的内容 从context中去获取 可用于获取用户id等操作
	queryInjects []ContextValueInject
	WriteInsert  bool     `json:"write_insert,omitempty"`   // 是否把注入内容写入新增体
	PostMustKeys []string `json:"post_must_keys,omitempty"` // 新增时候必须存在的key
	// 过滤参数能否通过 这里能注入和修改过滤参数和判断参数是否缺失 返回错误则抛出错误
	filterCanPass func(ctx iris.Context, query *ut.QueryFull) error
}

func (s *SchemaModel[T]) AddQueryInject(q ContextValueInject) {
	s.queryInjects = append(s.queryInjects, q)
}

func (s *SchemaModel[T]) GetContextUser(ctx iris.Context) *SimpleUserModel {
	var user *SimpleUserModel = nil
	if ctx.Values().Exists(UserContextKey) {
		user = ctx.Values().Get(UserContextKey).(*SimpleUserModel)
	}
	return user
}

func NewSchemaModel[T any](raw T, db *qmgo.Database) *SchemaModel[T] {
	var r = &SchemaModel[T]{
		db:      db,
		Actions: make([]ISchemaAction, 0),
	}
	r.SetRaw(raw)
	return r
}

func ToJsonSchema[T any](origin T, omitFields ...string) *jsonschema.Schema {
	schema := new(jsonschema.Reflector)
	// 只要为标记为omitempty的都会进入required
	//schema.RequiredFromJSONSchemaTags = true
	// 用真实的[]uint8 别去mock去一个 string base64出来
	schema.DoNotBase64 = true
	// 为true 则会写入Properties 对于object会写入$defs 生成$ref引用
	schema.ExpandedStruct = true
	schema.Mapper = func(r reflect.Type) *jsonschema.Schema {
		switch r {
		case reflect.TypeOf(primitive.ObjectID{}):
			return &jsonschema.Schema{
				Type:   "string",
				Format: "objectId",
			}
		case reflect.TypeOf(time.Time{}):
			return &jsonschema.Schema{
				Type:   "string",
				Format: "date-time",
			}
		}

		return nil
	}

	// 映射comment为Title
	schema.AddTagSetMapper("comment", "Title")

	// 还应该跳过_id 和uid 不让改
	schema.Intercept = func(field reflect.StructField) bool {
		if field.Name == "Id" && field.Type == reflect.TypeOf(primitive.ObjectID{}) {
			return false
		}
		if field.Name == "Uid" && field.Type == reflect.TypeOf("") {
			return false
		}
		for _, omitField := range omitFields {
			if field.Name == omitField {
				return false
			}
		}
		return true
	}

	ref := schema.Reflect(origin)
	return ref
}

func (s *SchemaModel[T]) SetRaw(raw T) {

	s.raw = raw
	s.Schema = ToJsonSchema(raw)
	// 通过反射获取struct的名称 不包含包名

	typ := reflect.TypeOf(raw)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	name := typ.Name()
	s.RawName = name

	s.EngName = name
}

func (s *SchemaModel[T]) AddAction(action ISchemaAction) {
	s.Actions = append(s.Actions, action)
}

func (s *SchemaModel[T]) SetFilterCanPass(filterCanPass func(ctx iris.Context, query *ut.QueryFull) error) {
	s.filterCanPass = filterCanPass
}

func (s *SchemaModel[T]) ParseInject(ctx iris.Context) ([]*ut.Kov, error) {
	result := make([]*ut.Kov, 0, len(s.queryInjects))
	for _, inject := range s.queryInjects {
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

func (s *SchemaModel[T]) HaveUserKey() bool {
	_, ok := s.Schema.Properties.Get(UserIdFieldName)
	return ok
}

func (s *SchemaModel[T]) ToAny() *SchemaModel[any] {
	var b = new(SchemaModel[any])
	b.Group = s.Group
	b.Alias = s.Alias
	b.Actions = s.Actions
	b.EngName = s.EngName
	b.Schema = s.Schema
	b.Priority = s.Priority
	b.queryInjects = s.queryInjects
	b.WriteInsert = s.WriteInsert
	b.filterCanPass = s.filterCanPass
	b.raw = s.raw
	b.db = s.db
	return b
}
func (s *SchemaModel[T]) GetAction(name string) (ISchemaAction, bool) {
	for _, ac := range s.Actions {
		if ac.GetBase().Name == name {
			return ac, true
		}
	}
	return nil, false
}

func (s *SchemaModel[T]) MarshalJSON() ([]byte, error) {
	inline := struct {
		Info    SchemaBase         `json:"info"`
		Actions []ISchemaAction    `json:"actions"`
		Schema  *jsonschema.Schema `json:"schema"`
	}{
		Info:    s.SchemaBase,
		Actions: s.Actions,
		Schema:  s.Schema,
	}
	inline.Schema.Title = s.SchemaBase.Alias
	return json.Marshal(inline)
}

func (s *SchemaModel[T]) GetCollection() *qmgo.Collection {
	return s.db.Collection(s.EngName)
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
		if resp.Result.SortAsc == nil && resp.Result.SortDesc == nil {
			resp.Result.SortDesc = append(resp.Result.SortDesc, "update_at")
		}
		if len(resp.Result.SortAsc) < 1 && len(resp.Result.SortDesc) < 1 {
			resp.Result.SortDesc = append(resp.Result.SortDesc, "update_at")
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
		if getParams.Single {
			return dataResp.Err
		}
		if dataResp.Err != qmgo.ErrNoSuchDocuments {
			return dataResp.Err
		}
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
	result.Sorts = resp.Result.BaseSort
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

	user := s.GetContextUser(ctx)
	uid := insertResult.Result[ut.DefaultUidTag]
	uidStr, _ := uid.(string)
	MustOpLog(ctx, s.db.Collection("operation_log"), "post", user, s.EngName, "新增一行", uidStr, nil)

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

	user := s.GetContextUser(ctx)
	var fields = make([]ut.Kov, 0, len(resp.Result))
	for k, v := range resp.Result {
		fields = append(fields, ut.Kov{
			Key:   k,
			Value: v,
		})
	}
	MustOpLog(ctx, s.db.Collection("operation_log"), "put", user, s.EngName, "修改行", params.RowId, fields)

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

	user := s.GetContextUser(ctx)
	MustOpLog(ctx, s.db.Collection("operation_log"), "del", user, s.EngName, "删除行", params.RowId, nil)

	return nil
}

func (s *SchemaModel[T]) getUid(ctx iris.Context) (string, error) {
	uid := ctx.Params().Get(ut.DefaultUidTag)
	if len(uid) < 1 {
		return "", errors.New("获取行id失败")
	}
	if len(uid) < 1 {
		return "", errors.New("行id为空")
	}
	return uid, nil
}

func (s *SchemaModel[T]) ActionEntry(ctx iris.Context) {
	ActionRun[T, map[string]any](ctx, s.ToAny(), nil)
	return
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
			"info":    s.SchemaBase,
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
