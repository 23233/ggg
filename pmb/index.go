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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"regexp"
	"strings"
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

type SchemaBase struct {
	Group     string `json:"group,omitempty"`      // 组名
	Priority  int    `json:"priority,omitempty"`   // 在组下显示的优先级 越大越优先
	TableName string `json:"table_name,omitempty"` // 表名
	UniqueId  string `json:"unique_id,omitempty"`  // 唯一ID 默认生成sonyflakeId
	PathId    string `json:"path_id,omitempty"`    // 路径ID 默认取UniqueId 在加入backend时会自动修改
	RawName   string `json:"raw_name,omitempty"`   // 原始名称
	Alias     string `json:"alias,omitempty"`      // 别名 中文名
}

type SchemaTableDisable struct {
	Read    bool `json:"read,omitempty"`
	Edit    bool `json:"edit,omitempty"`
	Del     bool `json:"del,omitempty"`
	Create  bool `json:"create,omitempty"`
	Actions bool `json:"actions,omitempty"`
}

type SchemaRole struct {
	RoleGroup []string `json:"role_group"` // staff root
	NameGroup []string `json:"name_group"` // 自定义名称组
}

type SchemaHooks[T any] struct {
	CustomAddHandler func(ctx iris.Context, params ut.ModelCtxMapperPack, model *SchemaModel[T]) error
	OnAddBefore      func(ctx iris.Context, args *pipe.RunResp[any], model *SchemaModel[T]) error                                      // 在新增之前
	OnAddAfter       func(ctx iris.Context, args *pipe.RunResp[map[string]any], model *SchemaModel[T]) error                           // 在新增之后
	OnGetBefore      func(ctx iris.Context, args *pipe.RunResp[*ut.QueryFull], params *pipe.ModelGetData, model *SchemaModel[T]) error // 在获取之前
	OnGetAfter       func(ctx iris.Context, args *pipe.RunResp[*ut.MongoFacetResult], model *SchemaModel[T]) error                     // 在获取之后
	OnEditBefore     func(ctx iris.Context)                                                                                            // 在修改之前
	OnEditAfter      func(ctx iris.Context)                                                                                            // 在修改之后
	OnDelBefore      func(ctx iris.Context)                                                                                            // 在删除之前
	OnDelAfter       func(ctx iris.Context)                                                                                            // 在删除之后
}

type SchemaIframe struct {
	Url string `json:"url,omitempty"`
}

type ManySchema struct {
	Table  *jsonschema.Schema `json:"table"`
	Edit   *jsonschema.Schema `json:"edit"`
	Add    *jsonschema.Schema `json:"add"`
	Delete *jsonschema.Schema `json:"delete"`
}

type SchemaAllowMethods struct {
	GetAll    bool `json:"get_all"`
	GetSingle bool `json:"get_single"`
	Post      bool `json:"post"`
	Put       bool `json:"put"`
	Delete    bool `json:"delete"`
}

func (c *SchemaAllowMethods) ChangeGetAll(status bool) {
	c.GetAll = status
}
func (c *SchemaAllowMethods) ChangeGetSingle(status bool) {
	c.GetSingle = status
}
func (c *SchemaAllowMethods) ChangePost(status bool) {
	c.Post = status
}
func (c *SchemaAllowMethods) ChangePut(status bool) {
	c.Put = status
}
func (c *SchemaAllowMethods) ChangeDelete(status bool) {
	c.Delete = status
}
func (c *SchemaAllowMethods) OnlyGet(status bool) {
	c.GetAll = status
	c.GetSingle = status
	c.Post = !status
	c.Put = !status
	c.Delete = !status
}

type SchemaModel[T any] struct {
	SchemaBase
	db            *qmgo.Database
	raw           any
	Roles         *SchemaRole         `json:"roles,omitempty"`
	Schemas       ManySchema          `json:"schemas"`
	Iframe        *SchemaIframe       `json:"iframe,omitempty"`
	TableDisable  *SchemaTableDisable `json:"table_disable,omitempty"`
	Actions       []ISchemaAction     `json:"actions,omitempty"`        // 各类操作
	DynamicFields []*DynamicField     `json:"dynamic_fields,omitempty"` // 动态字段
	AllowMethods  *SchemaAllowMethods `json:"allow_methods"`

	GetHandlerConfig    pipe.QueryParseConfig
	PostHandlerConfig   ut.ModelCtxMapperPack
	PutHandlerConfig    pipe.ModelPutConfig
	DeleteHandlerConfig pipe.ModelDelConfig

	// 每个查询都注入的内容 从context中去获取 可用于获取用户id等操作
	queryContextInjects []ContextValueInject
	queryFilterInject   []*ut.Kov // 过滤参数注入
	WriteInsert         bool      `json:"write_insert,omitempty"`   // 是否把注入内容写入新增体
	PostMustKeys        []string  `json:"post_must_keys,omitempty"` // 新增时候必须存在的key
	// 过滤参数能否通过 这里能注入和修改过滤参数和判断参数是否缺失 返回错误则抛出错误
	filterCanPass func(ctx iris.Context, self *SchemaModel[T], query *ut.QueryFull) error
	Hooks         SchemaHooks[T]
}

func (s *SchemaModel[T]) AddQueryContextInject(q ContextValueInject) {
	s.queryContextInjects = append(s.queryContextInjects, q)
}

func (s *SchemaModel[T]) GetContextUser(ctx iris.Context) *SimpleUserModel {
	var user *SimpleUserModel = nil
	if ctx.Values().Exists(UserContextKey) {
		user = ctx.Values().Get(UserContextKey).(*SimpleUserModel)
	}
	return user
}

func (s *SchemaModel[T]) GetSelf() *SchemaModel[T] {
	return s
}
func (s *SchemaModel[T]) GetBase() SchemaBase {
	return s.SchemaBase
}
func (s *SchemaModel[T]) GetRoles() *SchemaRole {
	return s.Roles
}

func NewSchemaModel[T any](raw T, db *qmgo.Database) *SchemaModel[T] {
	var r = &SchemaModel[T]{
		db:      db,
		Actions: make([]ISchemaAction, 0),
		AllowMethods: &SchemaAllowMethods{
			GetAll:    true,
			GetSingle: true,
			Post:      true,
			Put:       true,
			Delete:    true,
		},
	}
	r.SetRaw(raw)
	return r
}

// ToJsonSchema 把一个struct转换为jsonschema 在后台的前端ui-schema中不支持patternProperties
// 也就是说不支持map[string]any的写法 omitFields 是需要跳过的字段名 是struct的字段名
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
func (s *SchemaModel[T]) cleanTypeName(typeName string) string {
	// 移除所有空的花括号
	reBraces := regexp.MustCompile(`\{\s*\}`)
	typeName = reBraces.ReplaceAllString(typeName, "")

	// 替换第一个出现的 [ 为 _
	typeName = strings.Replace(typeName, "[", "_", 1)

	// 移除所有剩余的 [ 和 ]
	reRemainingBrackets := regexp.MustCompile(`[\[\]]`)
	typeName = reRemainingBrackets.ReplaceAllString(typeName, "")

	// 去除字符串末尾的空格
	typeName = strings.TrimRight(typeName, " ")

	return typeName
}

func (s *SchemaModel[T]) SetRaw(raw any) {

	s.raw = raw
	schema := ToJsonSchema(raw)
	s.Schemas = ManySchema{
		Table:  schema,
		Edit:   schema,
		Add:    schema,
		Delete: schema,
	}
	// 通过反射获取struct的名称 不包含包名

	typ := reflect.TypeOf(raw)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	name := typ.Name()
	s.RawName = name

	s.TableName = s.cleanTypeName(name)
	// 唯一Id 的生成规则?
	s.UniqueId = pipe.SfNextId()
}

func (s *SchemaModel[T]) AddAction(action ISchemaAction) {
	s.Actions = append(s.Actions, action)
}

func (s *SchemaModel[T]) SetFilterCanPass(filterCanPass func(ctx iris.Context, self *SchemaModel[T], query *ut.QueryFull) error) {
	s.filterCanPass = filterCanPass
}

func (s *SchemaModel[T]) AddDynamicField(f *DynamicField) {
	s.DynamicFields = append(s.DynamicFields, f)
}
func (s *SchemaModel[T]) AddDF(id, name string, call DynamicCall) {
	s.DynamicFields = append(s.DynamicFields, &DynamicField{
		Id:   id,
		Name: name,
		call: call,
	})
}
func (s *SchemaModel[T]) DfIdFind(id string) *DynamicField {
	for _, field := range s.DynamicFields {
		if field.Id == id {
			return field
		}
	}
	return nil
}

func (s *SchemaModel[T]) AddQueryFilterInject(fl *ut.Kov) {
	if s.queryFilterInject == nil {
		s.queryFilterInject = make([]*ut.Kov, 0)
	}
	s.queryFilterInject = append(s.queryFilterInject, fl)
}

func (s *SchemaModel[T]) ParseInject(ctx iris.Context) ([]*ut.Kov, error) {
	result := make([]*ut.Kov, 0, len(s.queryContextInjects))
	for _, inject := range s.queryContextInjects {
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

func (s *SchemaModel[T]) HaveUserKey(schema *jsonschema.Schema) bool {
	_, ok := schema.Properties.Get(UserIdFieldName)
	return ok
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
		Info          SchemaBase          `json:"info"`
		Actions       []ISchemaAction     `json:"actions"`
		DynamicFields []*DynamicField     `json:"dynamic_fields"`
		Schemas       ManySchema          `json:"schemas"`
		TableDisable  *SchemaTableDisable `json:"table_disable"`
		Iframe        *SchemaIframe       `json:"iframe"`
	}{
		Info:          s.SchemaBase,
		Actions:       s.Actions,
		Schemas:       s.Schemas,
		TableDisable:  s.TableDisable,
		Iframe:        s.Iframe,
		DynamicFields: s.DynamicFields,
	}
	inline.Schemas.Table.Title = s.SchemaBase.Alias
	return json.Marshal(inline)
}

func (s *SchemaModel[T]) GetCollection() *qmgo.Collection {
	return s.db.Collection(s.TableName)
}
func (s *SchemaModel[T]) GetDb() *qmgo.Database {
	return s.db
}
func (s *SchemaModel[T]) GetTableName() string {
	return s.TableName
}
func (s *SchemaModel[T]) GetAllAction() []ISchemaAction {
	return s.Actions
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

	if s.queryFilterInject != nil && len(s.queryFilterInject) >= 1 {
		queryParams.InjectAnd = append(queryParams.InjectAnd, s.queryFilterInject...)
	}

	// 解析query
	resp := pipe.QueryParse.Run(ctx, nil, &queryParams, nil)
	if resp.Err != nil {
		return resp.Err
	}
	// 过滤参数 外键什么的可以在这里注入
	if s.filterCanPass != nil {
		err = s.filterCanPass(ctx, s, resp.Result)
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

	if s.Hooks.OnGetBefore != nil {
		err := s.Hooks.OnGetBefore(ctx, resp, &getParams, s)
		if err != nil {
			return err
		}
	}

	// 获取数据
	dataResp := pipe.QueryGetData.Run(ctx,
		&pipe.ModelGetDataDep{
			ModelId: s.GetTableName(),
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

	if s.Hooks.OnGetAfter != nil {
		err := s.Hooks.OnGetAfter(ctx, dataResp, s)
		if err != nil {
			return err
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
func (s *SchemaModel[T]) PostHandler(ctx iris.Context, params ut.ModelCtxMapperPack) error {

	if s.Hooks.CustomAddHandler != nil {
		return s.Hooks.CustomAddHandler(ctx, params, s)
	}

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
	err := ctx.ReadBody(newV)
	if err != nil {
		return err
	}

	err = params.Process(newV)
	if err != nil {
		return err
	}

	// 必须出现在body中的字段名
	if s.PostMustKeys != nil {
		err := checkKeys(s.PostMustKeys, newV)
		if err != nil {
			return err
		}
	}

	if s.Hooks.OnAddBefore != nil {
		err := s.Hooks.OnAddBefore(ctx, pipe.NewPipeResult(newV), s)
		if err != nil {
			return err
		}
	}

	// 进行新增
	insertResult := pipe.ModelAdd.Run(ctx, newV, &pipe.ModelCtxAddConfig{ModelId: s.GetTableName()}, s.db)
	if insertResult.Err != nil {
		return insertResult.Err
	}

	if s.Hooks.OnAddAfter != nil {
		err := s.Hooks.OnAddAfter(ctx, insertResult, s)
		if err != nil {
			return err
		}
	}

	ctx.JSON(insertResult.Result)

	user := s.GetContextUser(ctx)
	uid := insertResult.Result[ut.DefaultUidTag]
	uidStr, _ := uid.(string)
	MustOpLog(ctx, s.db.Collection("operation_log"), "post", user, s.GetTableName(), "新增一行", uidStr, nil)

	return nil
}

func (s *SchemaModel[T]) SetPathId(newId string) {
	s.PathId = newId
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
	if params.QueryFilter.QueryParse == nil {
		params.QueryFilter.QueryParse = new(ut.QueryParse)
	}

	params.QueryFilter.QueryParse.InsertOrReplaces("and", injectQuery...)

	params.ModelId = s.GetTableName()

	newV := s.newRaw()
	err = ctx.ReadBody(&newV)
	if err != nil {
		return err
	}

	bodyMap := make(map[string]any)
	err = ctx.ReadBody(&bodyMap)
	if err != nil {
		return err
	}

	params.BodyMap = bodyMap

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
	MustOpLog(ctx, s.db.Collection("operation_log"), "put", user, s.GetTableName(), "修改行", params.RowId, fields)

	return nil
}

func (s *SchemaModel[T]) DelHandler(ctx iris.Context, params pipe.ModelDelConfig) error {
	params.ModelId = s.GetTableName()

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
	MustOpLog(ctx, s.db.Collection("operation_log"), "del", user, s.GetTableName(), "删除行", params.RowId, nil)

	return nil
}

func (s *SchemaModel[T]) getUid(ctx iris.Context) (string, error) {
	uid := ctx.Params().Get(ut.DefaultUidTag)
	if len(uid) < 1 {
		return "", errors.New("获取行id失败")
	}
	return uid, nil
}

func (s *SchemaModel[T]) ActionEntry(ctx iris.Context) {
	var user *SimpleUserModel
	if ctx.Values().Exists(UserContextKey) {
		user = ctx.Values().Get(UserContextKey).(*SimpleUserModel)
	}
	ActionRun(ctx, s, user)
	return
}

func (s *SchemaModel[T]) DynamicFieldsEntry(ctx iris.Context) {
	var user *SimpleUserModel
	if ctx.Values().Exists(UserContextKey) {
		user = ctx.Values().Get(UserContextKey).(*SimpleUserModel)
	}
	DynamicRun(ctx, s, user)
	return
}

func (s *SchemaModel[T]) Registry(parentParty iris.Party) {
	p := s.GeneratorParty(parentParty)
	s.RegistryConfigAction(p)
	s.RegistryCrud(p)
}
func (s *SchemaModel[T]) GeneratorParty(parentParty iris.Party, contextHandlers ...iris.Handler) iris.Party {
	p := parentParty.Party("/"+s.UniqueId, contextHandlers...)
	return p
}
func (s *SchemaModel[T]) RegistryConfigAction(p iris.Party) {
	// 获取配置文件
	p.Get("/config", func(ctx iris.Context) {
		_ = ctx.JSON(s)
		return
	})
	// action
	p.Post("/action", s.ActionEntry)
	// dynamic
	p.Post("/dynamic", s.DynamicFieldsEntry)
}
func (s *SchemaModel[T]) RegistryCrud(p iris.Party) {
	if s.AllowMethods.GetAll {
		p.Get("/", s.CrudHandler)
	}
	if s.AllowMethods.GetSingle {
		p.Get("/{uid:string}", s.CrudHandler)
	}
	if s.AllowMethods.Post {
		p.Post("/", s.CrudHandler)
	}
	if s.AllowMethods.Put {
		p.Put("/{uid:string}", s.CrudHandler)
	}
	if s.AllowMethods.Delete {
		p.Delete("/{uid:string}", s.CrudHandler)
	}
}

func (s *SchemaModel[T]) CrudHandler(ctx iris.Context) {
	method := strings.ToLower(ctx.Method())
	var err error
	uid := ctx.Params().Get(ut.DefaultUidTag)
	uidHas := len(uid) >= 1
	// get 但是没有uid 则是获取全部
	switch method {
	case "get":
		err = s.GetHandler(ctx, s.GetHandlerConfig, pipe.ModelGetData{
			Single:        uidHas,
			GetQueryCount: !uidHas,
		}, "")
		break
	case "post":
		if s.Hooks.CustomAddHandler != nil {
			err = s.Hooks.CustomAddHandler(ctx, s.PostHandlerConfig, s)
		} else {
			err = s.PostHandler(ctx, s.PostHandlerConfig)
		}
		break
	case "put":
		if !uidHas {
			break
		}
		s.PutHandlerConfig.UpdateTime = true
		err = s.PutHandler(ctx, s.PutHandlerConfig)
		break
	case "delete":
		if !uidHas {
			break
		}
		err = s.DelHandler(ctx, s.DeleteHandlerConfig)
		break
	default:
		err = errors.New("未被支持的方法")
	}

	if err != nil {
		IrisRespErr("", err, ctx)
		return
	}

}
func (s *SchemaModel[T]) GetSchema(mode ISchemaMode) *jsonschema.Schema {
	switch mode {
	case SchemaModeAdd:
		return s.Schemas.Add
	case SchemaModeEdit:
		return s.Schemas.Edit
	case SchemaModeTable:
		return s.Schemas.Table
	case SchemaModeDelete:
		return s.Schemas.Delete
	default:
		return s.Schemas.Table
	}
}
func (s *SchemaModel[T]) SetSchemaRaw(mode ISchemaMode, raw any) {
	schema := ToJsonSchema(raw)
	s.SetSchema(mode, schema)
}
func (s *SchemaModel[T]) SetSchema(mode ISchemaMode, schema *jsonschema.Schema) {
	switch mode {
	case SchemaModeAdd:
		s.Schemas.Add = schema
	case SchemaModeEdit:
		s.Schemas.Edit = schema
	case SchemaModeTable:
		s.Schemas.Table = schema
	case SchemaModeDelete:
		s.Schemas.Delete = schema
	default:
		s.Schemas.Table = schema
	}
}

type ISchemaMode int

const (
	SchemaModeAdd ISchemaMode = iota
	SchemaModeEdit
	SchemaModeTable
	SchemaModeDelete
)

type IModelItem interface {
	GetBase() SchemaBase
	SetRaw(raw any)
	AddAction(action ISchemaAction)
	GetAction(name string) (ISchemaAction, bool)
	AddQueryFilterInject(fl *ut.Kov)
	ParseInject(ctx iris.Context) ([]*ut.Kov, error)
	GetCollection() *qmgo.Collection
	GetDb() *qmgo.Database
	GetTableName() string
	GetHandler(ctx iris.Context, queryParams pipe.QueryParseConfig, getParams pipe.ModelGetData, uid string) error
	PostHandler(ctx iris.Context, params ut.ModelCtxMapperPack) error
	PutHandler(ctx iris.Context, params pipe.ModelPutConfig) error
	DelHandler(ctx iris.Context, params pipe.ModelDelConfig) error
	ActionEntry(ctx iris.Context)
	DynamicFieldsEntry(ctx iris.Context)
	Registry(part iris.Party)
	GeneratorParty(part iris.Party, contextHandlers ...iris.Handler) iris.Party
	RegistryConfigAction(p iris.Party)
	RegistryCrud(p iris.Party)
	CrudHandler(ctx iris.Context)
	AddQueryContextInject(q ContextValueInject)
	GetContextUser(ctx iris.Context) *SimpleUserModel
	GetAllAction() []ISchemaAction
	SetPathId(newId string)
	GetRoles() *SchemaRole
	HaveUserKey(schema *jsonschema.Schema) bool
	GetSchema(mode ISchemaMode) *jsonschema.Schema
	SetSchemaRaw(mode ISchemaMode, raw any)
	SetSchema(mode ISchemaMode, schema *jsonschema.Schema)
	AddDynamicField(f *DynamicField)
	AddDF(id, name string, call DynamicCall)
	DfIdFind(id string) *DynamicField
}
