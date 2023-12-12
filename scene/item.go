package scene

import (
	"github.com/23233/ggg/pipe"
	"github.com/23233/ggg/ut"
	"github.com/23233/jsonschema"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"reflect"
)

type ItemHandler func(ctx iris.Context, self ISceneModelItem) *pipe.RunResp[any]

type ItemExtra struct {
	GetCount   bool
	MustLastId bool
}

type Item struct {
	raw       any
	db        *qmgo.Database
	tableName string
	schema    *jsonschema.Schema
	mapper    *RequestData
	extra     *ItemExtra

	// 上下文注入到filter
	queryContextFilterInjects []ContextValueInject
	queryFilterInjects        []*ut.Kov // 过滤参数注入

	handler ItemHandler
}

type ISceneModelItem interface {
	SetRaw(raw any)
	GetRaw() any
	GetRawNew() any

	GetDb() *qmgo.Database
	SetDb(newDb *qmgo.Database)

	GetTableName() string
	SetTableName(newTableName string)

	SetSchema(newStruct any)
	GetSchema() *jsonschema.Schema

	GetMapper() *RequestData
	SetMapper(mapper *RequestData)

	Clone() ISceneModelItem

	GetExtra() *ItemExtra
	SetExtra(newExtra *ItemExtra)

	SetContextInjectFilter(injects []ContextValueInject)
	AddContextInjectFilter(injects ...ContextValueInject)
	GetContextInjectFilter() []ContextValueInject

	SetFilterInjects(injects []*ut.Kov)
	GetFilterInjects() []*ut.Kov
	AddFilterInject(injects ...*ut.Kov)

	SetHandler(ItemHandler)
	GetHandler() ItemHandler
}

func (i *Item) GetTableName() string {
	return i.tableName
}

func (i *Item) GetDb() *qmgo.Database {
	return i.db
}

func (i *Item) SetDb(newDb *qmgo.Database) {
	i.db = newDb
}

func (i *Item) SetTableName(newTableName string) {
	i.tableName = newTableName
}

func (i *Item) SetSchema(newStruct any) {
	i.schema = pipe.ToJsonSchema(newStruct)
}

func (i *Item) GetSchema() *jsonschema.Schema {
	return i.schema
}

func (i *Item) GetMapper() *RequestData {
	return i.mapper
}

func (i *Item) SetMapper(mapper *RequestData) {
	i.mapper = mapper
}

func (i *Item) SetContextInjectFilter(injects []ContextValueInject) {
	i.queryContextFilterInjects = injects
}
func (i *Item) AddContextInjectFilter(injects ...ContextValueInject) {
	i.queryContextFilterInjects = append(i.queryContextFilterInjects, injects...)
}

func (i *Item) GetContextInjectFilter() []ContextValueInject {
	return i.queryContextFilterInjects
}

func (i *Item) SetFilterInjects(injects []*ut.Kov) {
	i.queryFilterInjects = injects
}

func (i *Item) GetFilterInjects() []*ut.Kov {
	return i.queryFilterInjects
}
func (i *Item) AddFilterInject(injects ...*ut.Kov) {
	i.queryFilterInjects = append(i.queryFilterInjects, injects...)
}

func (i *Item) SetHandler(f ItemHandler) {
	i.handler = f
}

func (i *Item) GetHandler() ItemHandler {
	return i.handler
}

func (i *Item) Clone() ISceneModelItem {
	var a = new(Item)
	*a = *i
	return a
}

func (i *Item) GetExtra() *ItemExtra {
	return i.extra
}
func (i *Item) SetExtra(newExtra *ItemExtra) {
	i.extra = newExtra
}

func (i *Item) SetRaw(raw any) {
	i.raw = raw
	i.SetSchema(raw)
}
func (i *Item) GetRaw() any {
	return i.raw
}
func (i *Item) GetRawNew() any {
	typ := reflect.TypeOf(i.raw)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	newV := reflect.New(typ).Interface()
	return newV
}
