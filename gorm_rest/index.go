package gorm_rest

import (
	"encoding/json"
	"fmt"
	"github.com/23233/ggg/logger"
	"github.com/23233/ggg/ut"
	"github.com/23233/jsonschema"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var (
	// GormIdKey 模型定义中的id字段的json标签名
	GormIdKey = "id"
)

type ContextValueInject struct {
	FromKey    string `json:"from_key,omitempty"`
	ToKey      string `json:"to_key,omitempty"`
	AllowEmpty bool   `json:"allow_empty,omitempty"`
}

type SchemaBase struct {
	Group     string `json:"group,omitempty"`      // 组名
	Priority  int    `json:"priority,omitempty"`   // 在组下显示的优先级 越大越优先
	TableName string `json:"table_name,omitempty"` // 表名 虽然没用到
	UniqueId  string `json:"unique_id,omitempty"`  // 唯一ID 默认生成sonyflakeId
	PathId    string `json:"path_id,omitempty"`    // 路径ID 默认取party的最后一个路径
	RawName   string `json:"raw_name,omitempty"`   // 原始名称
	Alias     string `json:"alias,omitempty"`      // 别名 中文名
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

type GormQueryParseConfig struct {
	SearchFields []string          `json:"search_fields,omitempty"`
	InjectAnd    []*ut.Kov         `json:"inject_and,omitempty"`
	InjectOr     []*ut.Kov         `json:"inject_or,omitempty"`
	UrlParams    map[string]string `json:"url_params,omitempty"`
}

type GormModelPutConfig struct {
	QueryFilter *ut.QueryFull  `json:"query_filter,omitempty"`
	DropKeys    []string       `json:"drop_keys,omitempty"` // 最后的diff还需要丢弃的key
	RowId       string         `json:"row_id,omitempty"`    // 字符串也没事 mysql会自动把id转换为int
	UpdateTime  bool           `json:"update_time,omitempty"`
	UpdateForce bool           `json:"update_force,omitempty"` // 强行覆盖
	BodyMap     map[string]any `json:"body_map,omitempty"`
}

type GormModelDelConfig struct {
	QueryFilter *ut.QueryFull `json:"query_filter,omitempty"`
	RowId       string        `json:"row_id,omitempty"`
}

type GormSchemaRest[T any] struct {
	SchemaBase
	db           *gorm.DB
	raw          any
	Schemas      ManySchema          `json:"schemas"`
	AllowMethods *SchemaAllowMethods `json:"allow_methods"`

	GetHandlerConfig    GormQueryParseConfig
	PostHandlerConfig   ut.ModelCtxMapperPack
	PutHandlerConfig    GormModelPutConfig
	DeleteHandlerConfig GormModelDelConfig
	RaiseRawError       bool

	// 每个查询都注入的内容 从context中去获取 可用于获取用户id等操作
	queryContextInjects []ContextValueInject
	queryFilterInject   []*ut.Kov // 过滤参数注入
	WriteInsert         bool      `json:"write_insert,omitempty"`   // 是否把注入内容写入新增体
	PostMustKeys        []string  `json:"post_must_keys,omitempty"` // 新增时候必须存在的key 以json tag为准
	// 过滤参数能否通过 这里能注入和修改过滤参数和判断参数是否缺失 返回错误则抛出错误
	filterCanPass func(ctx iris.Context, self *GormSchemaRest[T], query *ut.QueryFull) error
}

func (s *GormSchemaRest[T]) AddQueryContextInject(q ContextValueInject) {
	s.queryContextInjects = append(s.queryContextInjects, q)
}

func (s *GormSchemaRest[T]) GetSelf() *GormSchemaRest[T] {
	return s
}
func (s *GormSchemaRest[T]) GetBase() SchemaBase {
	return s.SchemaBase
}

func NewGormSchemaRest[T any](raw T, db *gorm.DB) *GormSchemaRest[T] {
	var r = &GormSchemaRest[T]{
		db: db,
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
func (s *GormSchemaRest[T]) cleanTypeName(typeName string) string {
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

func (s *GormSchemaRest[T]) SetRaw(raw any) {

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

	// 获取gorm的表名
	stmt := GetSchema(new(T), s.db)
	if stmt != nil {
		s.TableName = stmt.Table
	} else {
		s.TableName = s.cleanTypeName(name)
	}
}

func (s *GormSchemaRest[T]) SetFilterCanPass(filterCanPass func(ctx iris.Context, self *GormSchemaRest[T], query *ut.QueryFull) error) {
	s.filterCanPass = filterCanPass
}

func (s *GormSchemaRest[T]) AddQueryFilterInject(fl *ut.Kov) {
	if s.queryFilterInject == nil {
		s.queryFilterInject = make([]*ut.Kov, 0)
	}
	s.queryFilterInject = append(s.queryFilterInject, fl)
}

func (s *GormSchemaRest[T]) ParseInject(ctx iris.Context) ([]*ut.Kov, error) {
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

func (s *GormSchemaRest[T]) newRaw() any {
	typ := reflect.TypeOf(s.raw)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	newV := reflect.New(typ).Interface()
	return newV
}

// getHandler 仅获取数据
func (s *GormSchemaRest[T]) getHandler(ctx iris.Context, queryParams GormQueryParseConfig, singe bool) error {
	injectQuery, err := s.ParseInject(ctx)
	if err != nil {
		return err
	}
	var uid string
	if singe {
		uid, err = s.getUid(ctx)
		if err != nil {
			return err
		}
		if queryParams.InjectAnd == nil {
			queryParams.InjectAnd = make([]*ut.Kov, 0)
		}
		queryParams.InjectAnd = append(queryParams.InjectAnd, &ut.Kov{
			Key:   GormIdKey,
			Op:    "eq",
			Value: uid,
		})
	}

	queryParams.InjectAnd = append(queryParams.InjectAnd, injectQuery...)

	if s.queryFilterInject != nil && len(s.queryFilterInject) >= 1 {
		queryParams.InjectAnd = append(queryParams.InjectAnd, s.queryFilterInject...)
	}

	if queryParams.UrlParams == nil {
		queryParams.UrlParams = ctx.URLParams()
	}

	qs := ut.NewPruneCtxQuery()
	mapper, err := qs.PruneParse(queryParams.UrlParams, queryParams.SearchFields, "")
	if err != nil {
		return err
	}
	if queryParams.InjectAnd != nil {
		mapper.QueryParse.InsertOrReplaces("and", queryParams.InjectAnd...)
	}
	if queryParams.InjectOr != nil {
		mapper.QueryParse.InsertOrReplaces("or", queryParams.InjectOr...)
	}

	// 过滤参数 外键什么的可以在这里注入
	if s.filterCanPass != nil {
		err = s.filterCanPass(ctx, s, mapper)
		if err != nil {
			return err
		}
	}

	if !singe {

		if mapper.Page < 1 {
			mapper.Page = 1
		}
		if mapper.Page > 100 {
			mapper.Page = 100
		}
		if mapper.PageSize <= 0 {
			mapper.PageSize = 10
		}
		if mapper.PageSize > 100 {
			mapper.PageSize = 100
		}
		// 默认按照更新时间倒序
		if mapper.SortAsc == nil && mapper.SortDesc == nil {
			mapper.SortDesc = append(mapper.SortDesc, "updated_at")
		}
		if len(mapper.SortAsc) < 1 && len(mapper.SortDesc) < 1 {
			mapper.SortDesc = append(mapper.SortDesc, "updated_at")
		}

	}

	queryResult, err := ut.RunGormQuery[T](mapper, s.db)
	if err != nil {
		return err
	}

	if singe {
		if queryResult.Datas == nil || len(queryResult.Datas) < 1 {
			return errors.New("获取单条数据失败")
		}
		ctx.JSON(queryResult.Datas[0])
		return nil
	}

	ctx.JSON(iris.Map{
		"page":      mapper.Page,
		"page_size": mapper.PageSize,
		"filters":   mapper.QueryParse,
		"sorts":     mapper.BaseSort,
		"count":     queryResult.Count,
		"data":      queryResult.Datas,
	})

	return nil

}

// postHandler 新增数据
func (s *GormSchemaRest[T]) postHandler(ctx iris.Context, params ut.ModelCtxMapperPack) error {

	if s.WriteInsert {
		injectQuery, err := s.ParseInject(ctx)
		if err != nil {
			return err
		}
		if params.InjectData == nil {
			params.InjectData = make(map[string]any)
		}
		for _, kov := range injectQuery {
			params.InjectData[kov.Key] = kov.Value
		}
	}

	// 通过模型去获取上下文
	newV := s.newRaw()
	err := ctx.ReadBody(newV)
	if err != nil {
		return err
	}

	// 进行内容注入和过滤
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

	// 进行创建
	err = s.db.Create(newV).Error
	if err != nil {
		return err
	}

	ctx.JSON(newV)

	return nil
}

// putHandler 更新数据 不校验 提交什么字段则覆盖什么字段
func (s *GormSchemaRest[T]) putHandler(ctx iris.Context, params GormModelPutConfig) error {
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
	params.QueryFilter.QueryParse.InsertOrReplaces("and", &ut.Kov{
		Key:   GormIdKey,
		Op:    "eq",
		Value: params.RowId,
	})
	// 先获取出原本那一条
	queryResult, err := ut.RunGormQuery[T](params.QueryFilter, s.db)
	if err != nil {
		return err
	}
	if queryResult.Datas == nil || len(queryResult.Datas) < 1 {
		return errors.New("未找到原始条目")
	}

	// 传什么改什么 使用map
	bodyMap := make(map[string]any)
	err = ctx.ReadBody(&bodyMap)
	if err != nil {
		return err
	}
	params.DropKeys = append(params.DropKeys, GormIdKey)
	for _, key := range params.DropKeys {
		if _, ok := bodyMap[key]; ok {
			delete(bodyMap, key)
		}
	}
	if len(bodyMap) < 1 {
		return errors.New("未找到变更项")
	}
	if params.UpdateTime {
		_, ok := bodyMap["updated_at"]
		if params.UpdateForce || !ok {
			bodyMap["updated_at"] = time.Now()
		}
	}
	err = s.db.Model(new(T)).Where("id = ?", params.RowId).Updates(bodyMap).Error
	if err != nil {
		return err
	}
	ctx.JSON(bodyMap)

	return nil
}

func (s *GormSchemaRest[T]) delHandler(ctx iris.Context, params GormModelDelConfig) error {
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
	params.QueryFilter.QueryParse.InsertOrReplaces("and", &ut.Kov{
		Key:   GormIdKey,
		Op:    "eq",
		Value: params.RowId,
	})

	// 先获取出原本那一条
	queryResult, err := ut.RunGormQuery[T](params.QueryFilter, s.db)
	if err != nil {
		return err
	}
	if queryResult.Datas == nil || len(queryResult.Datas) < 1 {
		return errors.New("未找到原始条目")
	}

	// 进行删除
	err = s.db.Delete(new(T), params.RowId).Error
	if err != nil {
		return err
	}

	_, _ = ctx.WriteString("ok")

	return nil
}

func (s *GormSchemaRest[T]) getUid(ctx iris.Context) (string, error) {
	uid := ctx.Params().Get(ut.DefaultUidTag)
	if len(uid) < 1 {
		return "", errors.New("获取行id失败")
	}
	return uid, nil
}

func (s *GormSchemaRest[T]) Registry(p iris.Party) {
	matchs := strings.Split(strings.TrimRight(p.GetRelPath(), "/"), "/")
	lastMatch := matchs[len(matchs)-1]
	s.PathId = lastMatch

	if s.AllowMethods.GetAll {
		p.Get("/", s.crudHandler)
	}
	if s.AllowMethods.GetSingle {
		p.Get("/{uid:string}", s.crudHandler)
	}
	if s.AllowMethods.Post {
		p.Post("/", s.crudHandler)
	}
	if s.AllowMethods.Put {
		p.Put("/{uid:string}", s.crudHandler)
	}
	if s.AllowMethods.Delete {
		p.Delete("/{uid:string}", s.crudHandler)
	}
	p.Get("/config", func(ctx iris.Context) {
		_ = ctx.JSON(s)
		return
	})
}

func (s *GormSchemaRest[T]) MarshalJSON() ([]byte, error) {
	inline := struct {
		Info    SchemaBase `json:"info"`
		Schemas ManySchema `json:"schemas"`
	}{
		Info:    s.SchemaBase,
		Schemas: s.Schemas,
	}
	inline.Schemas.Table.Title = s.SchemaBase.Alias
	return json.Marshal(inline)
}

func (s *GormSchemaRest[T]) crudHandler(ctx iris.Context) {
	method := strings.ToLower(ctx.Method())
	var err error
	uid := ctx.Params().Get(ut.DefaultUidTag)
	uidHas := len(uid) >= 1
	// get 但是没有uid 则是获取全部
	switch method {
	case "get":
		err = s.getHandler(ctx, s.GetHandlerConfig, uidHas)
		break
	case "post":
		err = s.postHandler(ctx, s.PostHandlerConfig)
		break
	case "put":
		if !uidHas {
			break
		}
		s.PutHandlerConfig.UpdateTime = true
		err = s.putHandler(ctx, s.PutHandlerConfig)
		break
	case "delete":
		if !uidHas {
			break
		}
		err = s.delHandler(ctx, s.DeleteHandlerConfig)
		break
	default:
		err = errors.New("未被支持的方法")
	}

	if err != nil {
		if s.RaiseRawError {
			logger.J.ErrorE(err, "执行出错")
			ut.IrisErr(ctx, err)
		} else {
			ut.IrisErrLog(ctx, err, "执行方法出错")
		}
		return
	}

}
func (s *GormSchemaRest[T]) SetSchemaRaw(mode ISchemaMode, raw any) {
	schema := ToJsonSchema(raw)
	s.SetSchema(mode, schema)
}
func (s *GormSchemaRest[T]) SetSchema(mode ISchemaMode, schema *jsonschema.Schema) {
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

func GetSchema(table any, db *gorm.DB) *schema.Schema {
	stmt := &gorm.Statement{DB: db}
	_ = stmt.Parse(table)
	return stmt.Schema
}
