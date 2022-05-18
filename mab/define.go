package mab

import (
	"github.com/23233/ggg/ut"
	tollerr "github.com/didip/tollbooth/v6/errors"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/importcjj/sensitive"
	"github.com/kataras/iris/v12"
	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"strings"
	"time"
)

// SingleModel 单个模型实例
// CustomModel 主要用来动态配置项 比如说context变更 pk变更等灵活使用 e.g:
//	func(ctx iris.Context, Model *SingleModel) *SingleModel {
//		newModel:= new(SingleModel)
//      *newModel = *Model
//      // 修改配置文件
//      return newModel
//	}
//
type SingleModel struct {
	Prefix                string                                                                                              // 路由前缀
	Suffix                string                                                                                              // 路由后缀
	Model                 any                                                                                                 // xorm Model
	ShowCount             bool                                                                                                // 显示搜索数量
	ShowDocCount          bool                                                                                                // 显示文档数量
	AllowGetInfo          bool                                                                                                // 允许获取表结构信息
	uriPath               string                                                                                              // 实际匹配的uri
	CustomModel           func(ctx iris.Context, model *SingleModel) *SingleModel                                             // 记住要进行指针值传递 返回一个新对象 否则会修改原始配置文件
	Pk                    func() []bson.D                                                                                     // 外键
	info                  ModelInfo                                                                                           //
	DisablePrivate        bool                                                                                                // 禁用私密参数
	DisablePrivateMap     map[string]bool                                                                                     // 禁用私密参数定制项 key为方法 value为启用与否 前提条件必须启用后才能禁用 而不能禁用后启用
	private               bool                                                                                                // 当有context key 以及col name时为true
	PrivateContextKey     string                                                                                              // 上下文key string int uint
	PrivateColName        string                                                                                              // 数据库字段名 MapName or ColName is ok
	privateStructName     string                                                                                              // 根据colName 找到真实的struct name
	privateIndex          int                                                                                                 //
	AllowMethods          []string                                                                                            // allow methods first
	DisableMethods        []string                                                                                            // get(all) get(single) post put delete
	MustSearch            bool                                                                                                // 必须搜索模式 会忽略下面的搜索设置强制开启搜索 (非slice|struct|primitive.ObjectID)
	AllowSearchFields     []string                                                                                            // 搜索的字段 struct名称
	searchFields          []StructInfo                                                                                        // allow search col names
	InjectParams          func(ctx iris.Context) map[string]string                                                            // 注入params 用于get请求参数的自定义
	GetAllResponseFunc    func(ctx iris.Context, result iris.Map, dataList []bson.M) iris.Map                                 // 返回内容替换的方法
	GetAllExtraFilters    func(ctx iris.Context) map[string]interface{}                                                       // 额外的固定过滤 key(数据库列名) 和 value 若与请求过滤重复则覆盖 优先级最高
	GetAllMustFilters     map[string]string                                                                                   // 获取全部必须拥有筛选
	GetSingleResponseFunc func(ctx iris.Context, item bson.M) bson.M                                                          // 获取单个返回内容替换的方法
	GetSingleExtraFilters func(ctx iris.Context) map[string]interface{}                                                       // 额外的固定过滤 key(数据库列名) 和 value 若与请求过滤重复则覆盖 优先级最高
	GetSingleMustFilters  map[string]string                                                                                   // 获取单个必须拥有筛选
	PostValidator         interface{}                                                                                         // 新增自定义验证器
	PostMustFilters       map[string]string                                                                                   // 新增必须存在的参数
	PostResponseFunc      func(ctx iris.Context, mid string, item interface{}) interface{}                                    //
	PostDataParse         func(ctx iris.Context, raw interface{}) interface{}                                                 //
	PutValidator          interface{}                                                                                         // 修改验证器
	PutDataParse          func(ctx iris.Context, mid string, diff bson.M) bson.M                                              //
	PutQueryParse         func(ctx iris.Context, mid string, query bson.M, data interface{}, privateValue interface{}) bson.M // 修改的时候query可以自定义修改
	PutResponseFunc       func(ctx iris.Context, mid string) iris.Map                                                         // 在修改之前还可以变更一下数据
	PutMustFilters        map[string]string                                                                                   //
	DeleteValidator       interface{}                                                                                         // 删除验证器
	DeleteResponseFunc    func(ctx iris.Context, mid string, item bson.M, result iris.Map) iris.Map                           //
	SensitiveFields       []string                                                                                            // 使用struct name 或者mapname 均可(map对象为bson:)
	sensitiveField        []string                                                                                            // post传入的key
	CacheTime             time.Duration                                                                                       // full cache time
	GetAllCacheTime       time.Duration                                                                                       // get all cache time
	GetSingleCacheTime    time.Duration                                                                                       // get single cache time
	DelayDeleteTime       time.Duration                                                                                       // 延迟多久双删 default 500ms
	MaxPageSize           int64                                                                                               // max page size limit
	MaxPageCount          int64                                                                                               // max page count limit
	RateErrorFunc         func(*tollerr.HTTPError, iris.Context)                                                              //
	Rate                  *limiter.Limiter                                                                                    // all
	GetAllRate            *limiter.Limiter                                                                                    //
	GetSingleRate         *limiter.Limiter                                                                                    //
	AddRate               *limiter.Limiter                                                                                    //
	PutRate               *limiter.Limiter                                                                                    //
	DeleteRate            *limiter.Limiter                                                                                    //
}

func (sm *SingleModel) init(cfg *Config) {
	model := sm.Model
	ct := reflect.TypeOf(model)
	if ct.Kind() == reflect.Ptr {
		ct = ct.Elem()
	}
	apiName := ut.STN(ct.Name())
	uriList := make([]string, 0, 6)
	uriList = append(uriList, "/")
	if len(sm.Prefix) > 0 {
		prefix := sm.Prefix
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		if strings.HasPrefix(prefix, "/") {
			uriList = append(uriList, prefix[1:])
		} else {
			uriList = append(uriList, prefix)
		}
	}
	uriList = append(uriList, apiName)
	uriList = append(uriList, sm.Suffix)
	// 拼接效率高
	uriPath := strings.Join(uriList, "")
	sm.uriPath = uriPath
	fields := TableNameReflectFieldsAndTypes(model, cfg.StructDelimiter)

	relPath := cfg.Party.GetRelPath()

	fullPath := strings.Join([]string{relPath, uriPath}, "")
	if relPath == "/" {
		fullPath = uriPath
	}

	info := ModelInfo{
		MapName:    apiName,
		FieldList:  fields,
		FullPath:   fullPath,
		FlatFields: flatField(fields),
	}

	if processor, ok := model.(AliasProcess); ok {
		info.Alias = processor.Alias()
	} else {
		if processor, ok := model.(SpAliasProcess); ok {
			info.Alias = processor.SpAlias()
		}
	}
	if len(info.Alias) >= 1 {
		// 以_开头则表示有群组 eg: _组名_表名
		if strings.HasPrefix(info.Alias, "_") && strings.Count(info.Alias, "_") == 2 {
			splitGroup := strings.Split(info.Alias, "_")
			info.Alias = splitGroup[2]
			info.Group = splitGroup[1]
		}
	}

	sm.info = info
	// 解析敏感词字段信息
	if len(sm.SensitiveFields) > 0 {
		sensitiveFullList := make([]string, 0, len(sm.SensitiveFields))
		for _, f := range sm.SensitiveFields {
			for _, field := range fields {
				if field.Name == f || field.MapName == f {
					sensitiveFullList = append(sensitiveFullList, field.MapName)
					break
				}
			}
		}
		sm.sensitiveField = sensitiveFullList
	}

	if sm.MustSearch {
		var f []StructInfo
		// 遍历字段信息
		for _, field := range info.FieldList {
			// 时间与主键
			if field.IsUpdated || field.IsCreated || field.IsDeleted || field.IsTime {
				f = append(f, field)
			}
			if field.IsDefaultWrap {
				for _, child := range field.Children {
					if child.IsUpdated || child.IsCreated || child.IsDeleted || field.IsTime {
						f = append(f, child)
					}
				}
				continue
			}
			// 跳过slice和struct 还有ObjId 这是获取不了的
			if field.Kind == "slice" || field.Kind == "struct" || field.IsObjId {
				continue
			}
			f = append(f, field)
		}
		sm.searchFields = f
	} else if len(sm.AllowSearchFields) >= 1 {
		var b []StructInfo

		for _, f := range sm.AllowSearchFields {
			for _, field := range info.FlatFields {
				if field.Name == f || field.MapName == f {
					b = append(b, field)
				}
			}
		}
		sm.searchFields = b
	}
	// 生成private信息
	sm.genPrivate(cfg.PrivateContextKey, cfg.PrivateColName)
}

func (sm *SingleModel) reset(v any) {
	p := reflect.ValueOf(v).Elem()
	p.Set(reflect.Zero(p.Type()))
}

// getMethods 初始化请求方法 返回数组
func (sm *SingleModel) getMethods() []string {
	if len(sm.AllowMethods) >= 1 {
		return sm.AllowMethods
	}
	m := sm.initMethods()
	if len(sm.DisableMethods) >= 1 {
		for _, method := range sm.DisableMethods {
			if _, ok := m[method]; ok {
				delete(m, method)
				continue
			}
		}
	}
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// initMethods 初始化请求方法 返回map
func (sm *SingleModel) initMethods() map[string]string {
	// get(all) get(single) post put delete
	return map[string]string{
		"get(all)":    "get(all)",
		"get(single)": "get(single)",
		"post":        "post",
		"put":         "put",
		"delete":      "delete",
	}
}

// genPrivate 生成私密参数信息
func (sm *SingleModel) genPrivate(globalPrivateContextKey, globalPrivateColName string) {
	// 当前不禁用私密参数时
	if sm.DisablePrivate == false {
		sm.private = (len(sm.PrivateContextKey) >= 1 && len(sm.PrivateColName) >= 1) || (len(globalPrivateContextKey) >= 1 && len(globalPrivateColName) >= 1)
	}
	if sm.private {
		// 局部大于全局
		var structName string
		var colName string
		if len(sm.PrivateColName) >= 1 {
			structName = sm.PrivateColName
		} else {
			structName = globalPrivateColName
		}
		for _, field := range sm.info.FieldList {
			if field.Name == sm.PrivateColName || field.MapName == sm.PrivateColName || field.Name == globalPrivateColName || field.MapName == globalPrivateColName {
				colName = field.MapName
				structName = field.Name
				break
			}
		}
		sm.privateStructName = structName
		sm.PrivateColName = colName
		if len(sm.PrivateContextKey) < 1 {
			sm.PrivateContextKey = globalPrivateContextKey
		}

		// 找到private index
		for _, field := range sm.info.FlatFields {
			if field.Name == structName || field.MapName == colName {
				sm.privateIndex = field.Index
				break
			}
		}
	}

}

// getPage 获取最大限制的页码和每页数量
func (sm *SingleModel) getPage() (int64, int64) {
	maxPageCount := sm.MaxPageCount
	if maxPageCount < 1 {
		maxPageCount = 100
	}
	maxPageSize := sm.MaxPageSize
	if maxPageSize < 1 {
		maxPageSize = 100
	}
	return maxPageCount, maxPageSize
}

// getDelayDeleteTime 获取延迟删除时间
func (sm *SingleModel) getDelayDeleteTime() time.Duration {
	if sm.DelayDeleteTime >= 1 {
		return sm.DelayDeleteTime
	}
	return 500 * time.Millisecond
}

// getAllListCacheTime 获取列表缓存时间
func (sm *SingleModel) getAllListCacheTime() time.Duration {
	if sm.GetAllCacheTime >= 1 {
		return sm.GetAllCacheTime
	}
	return sm.CacheTime
}

// getSingleCacheTime 获取单条缓存时间
func (sm *SingleModel) getSingleCacheTime() time.Duration {
	if sm.GetSingleCacheTime >= 1 {
		return sm.GetSingleCacheTime
	}
	return sm.CacheTime
}

// getAllRate get(all) rate
func (sm *SingleModel) getAllRate() *limiter.Limiter {
	if sm.GetAllRate != nil {
		return sm.GetAllRate
	}
	return sm.Rate
}

// getSingleRate get(single) rate
func (sm *SingleModel) getSingleRate() *limiter.Limiter {
	if sm.GetSingleRate != nil {
		return sm.GetSingleRate
	}
	return sm.Rate
}

// getAddRate get post rate
func (sm *SingleModel) getAddRate() *limiter.Limiter {
	if sm.AddRate != nil {
		return sm.AddRate
	}
	return sm.Rate
}

// getEditRate get put rate
func (sm *SingleModel) getEditRate() *limiter.Limiter {
	if sm.PutRate != nil {
		return sm.PutRate
	}
	return sm.Rate
}

// getDeleteRate get delete rate
func (sm *SingleModel) getDeleteRate() *limiter.Limiter {
	if sm.DeleteRate != nil {
		return sm.DeleteRate
	}
	return sm.Rate
}

// Config 配置文件
// 敏感词 用于国内的审核 可在model配置文件中配置需要验证的字段 每次数据新增/修改时会进行验证
type Config struct {
	Party             iris.Party
	Mdb               *qmgo.Database
	Models            []*SingleModel
	ErrorTrace        func(err error, event, from, router string) // error trace func
	Generator         bool                                        // 生成器模式 若启用则只有一个入口
	PrivateContextKey string
	PrivateColName    string
	StructDelimiter   string   // 内联struct之间的分隔符 默认__ 因为.号会被转义 不要使用. _
	SensitiveUri      []string // 敏感词库
	SensitiveWords    []string // 敏感词列表
}

// ModelInfo 模型信息
type ModelInfo struct {
	MapName    string       `json:"map_name"`
	FullPath   string       `json:"full_path"`
	Alias      string       `json:"alias"`
	Group      string       `json:"group"`
	FieldList  []StructInfo `json:"field_list"`
	FlatFields []StructInfo `json:"flat_fields"`
}

type RestApi struct {
	Cfg               *Config
	sensitiveInstance *sensitive.Filter
}

// StructInfo 模型字段信息
type StructInfo struct {
	Name          string       `json:"name"`                   // 字段名 struct name
	MapName       string       `json:"map_name"`               // 转snake格式的名称
	FullName      string       `json:"full_name"`              // [parentStructName][structDelimiter]][structName]
	FullMapName   string       `json:"full_map_name"`          // [parentSnakeName][structDelimiter][snakeName]
	ParamsKey     string       `json:"params_key"`             // post form key name
	CustomTag     string       `json:"custom_tag"`             // 自定义标签信息 mab:
	ValidateTag   string       `json:"validate_tag,omitempty"` // 验证器标签信息
	Comment       string       `json:"comment,omitempty"`
	Level         string       `json:"level"` // parentIndex - .... - self index
	Kind          string       `json:"kind"`
	Bson          []string     `json:"bson"`     // bson tag
	JsonTag       []string     `json:"json_tag"` // json tag
	Types         string       `json:"types"`
	Index         int          `json:"index,omitempty"`
	IsDefaultWrap bool         `json:"is_default_wrap,omitempty"`
	IsTime        bool         `json:"is_time,omitempty"`
	IsPk          bool         `json:"is_pk,omitempty"`
	IsObjId       bool         `json:"is_obj_id,omitempty"`
	IsCreated     bool         `json:"is_created,omitempty"`
	IsUpdated     bool         `json:"is_updated,omitempty"`
	IsDeleted     bool         `json:"is_deleted,omitempty"`
	IsGeo         bool         `json:"is_geo,omitempty"`
	IsInline      bool         `json:"is_inline,omitempty"`
	Children      []StructInfo `json:"children,omitempty"`
	ChildrenKind  string       `json:"children_kind,omitempty"`
}

type AliasProcess interface {
	Alias() string
}

type SpAliasProcess interface {
	SpAlias() string
}
