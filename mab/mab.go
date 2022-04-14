package mab

import (
	"github.com/23233/ggg/sv"
	"github.com/importcjj/sensitive"
	"github.com/kataras/iris/v12/context"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"reflect"
	"strings"
)

func New(cfg *Config) *RestApi {
	instance := new(RestApi)
	instance.Cfg = cfg
	if len(cfg.StructDelimiter) < 1 {
		cfg.StructDelimiter = "__"
	}
	instance.checkConfig()
	instance.initSensitive()
	instance.initModel()
	instance.run()
	return instance
}

func (rest *RestApi) run() {
	if rest.Cfg.Generator {
		rest.Cfg.Party.Any("/{Model:path}", rest.generatorMiddleware)
	} else {
		for _, item := range rest.Cfg.Models {
			api := rest.Cfg.Party.Party(item.uriPath)
			// 获取所有方法
			methods := item.getMethods()
			if len(methods) >= 1 {

				// 获取全部列表
				if isContain(methods, "get(all)") {
					h := rest.GetAllFunc
					r := api.Handle("GET", "/", h)
					// rate
					if item.getAllRate() != nil {
						r.Use(LimitHandler(item.getAllRate(), item.RateErrorFunc))
					}
					// cache
					if item.getAllListCacheTime() > 0 {
						r.Use(rest.getCacheMiddleware("list"))
					}
				}

				// 获取单条
				if isContain(methods, "get(single)") {
					var h context.Handler
					h = rest.GetSingle
					r := api.Handle("GET", "/{mid:string range(1,32)}", h)
					// rate
					if item.getSingleRate() != nil {
						r.Use(LimitHandler(item.getSingleRate(), item.RateErrorFunc))
					}
					if item.getSingleCacheTime() > 0 {
						r.Use(rest.getCacheMiddleware("single"))
					}
				}

				// 新增
				if isContain(methods, "post") {
					var h context.Handler
					h = rest.AddData
					route := api.Handle("POST", "/", h)

					// 判断是否有自定义验证器
					if item.PostValidator != nil {
						route.Use(sv.Run(item.PostValidator))
					} else {
						route.Use(sv.Run(item.Model, "json"))
					}

					// rate
					if item.getAddRate() != nil {
						route.Use(LimitHandler(item.getAddRate(), item.RateErrorFunc))
					}
				}

				// 修改
				if isContain(methods, "put") {
					var h context.Handler
					h = rest.EditData
					route := api.Handle("PUT", "/{mid:string range(1,32)}", h)

					// 判断是否有自定义验证器
					if item.PutValidator != nil {
						route.Use(sv.Run(item.PutValidator))
					} else {
						route.Use(sv.Run(item.Model, "json"))
					}
					// rate
					if item.getEditRate() != nil {
						route.Use(LimitHandler(item.getEditRate(), item.RateErrorFunc))
					}
				}

				// 删除
				if isContain(methods, "delete") {
					var h context.Handler
					h = rest.DeleteData
					route := api.Handle("DELETE", "/{mid:string range(1,32)}", h)
					// rate
					if item.getDeleteRate() != nil {
						route.Use(LimitHandler(item.getDeleteRate(), item.RateErrorFunc))
					}
					// 判断是否有自定义验证器
					if item.DeleteValidator != nil {
						route.Use(sv.Run(item.DeleteValidator))
					}
				}

			}
		}
	}
	rest.Cfg.Party.Get("/model_info/{modelName:string}", rest.GetModelInfo)
}

// initModel 初始化模型信息
func (rest *RestApi) initModel() {
	for _, item := range rest.Cfg.Models {
		item.init(rest.Cfg)
	}
}

// GetModelInfoList 获取详细的模型列表
func (rest *RestApi) GetModelInfoList() []ModelInfo {
	var result = make([]ModelInfo, 0, len(rest.Cfg.Models))
	for _, model := range rest.Cfg.Models {
		result = append(result, model.info)
	}
	return result
}

// 获取关键词模型
func (rest *RestApi) getSensitive() *sensitive.Filter {
	return rest.sensitiveInstance
}

// PathGetModel 通过路径获取对应的模型信息
func (rest *RestApi) PathGetModel(pathUri string) *SingleModel {
	uri := pathUri
	if rest.Cfg.Generator {
		uri, _ = rest.UriGetMid(pathUri)
	}
	for _, m := range rest.Cfg.Models {
		if m.info.FullPath == uri || strings.HasPrefix(uri, m.info.FullPath+"/") {
			return m
		}
	}
	return new(SingleModel)
}

// NameGetModel 根据名称获取model
func (rest *RestApi) NameGetModel(pathname string) (*SingleModel, error) {
	for _, m := range rest.Cfg.Models {
		if m.info.MapName == pathname {
			return m, nil
		} else {
			if len(m.uriPath) > 0 {
				if m.uriPath[1:] == pathname {
					return m, nil
				}
			}
		}
	}
	return nil, errors.New("模型信息匹配失败")
}

// PathGetMid 根据path提取出mid
func (rest *RestApi) PathGetMid(method string, uri string) (string, string) {
	switch method {
	case "GET", "PUT", "DELETE":
		return rest.UriGetMid(uri)
	}
	return uri, ""
}

// UriGetMid 根据uri获取mid
func (rest *RestApi) UriGetMid(uri string) (string, string) {
	lastList := strings.Split(uri, "/")
	if len(lastList) >= 2 {
		last := lastList[len(lastList)-1]
		if len(last) > 16 {
			_, err := primitive.ObjectIDFromHex(last)
			if err == nil {
				return strings.Join(lastList[0:len(lastList)-1], "/"), last
			}
		}
	}
	return uri, ""
}

// 通过模型名获取模型信息
func (rest *RestApi) tableNameGetModelInfo(tableName string) (*SingleModel, error) {
	for _, l := range rest.Cfg.Models {
		if l.info.MapName == tableName {
			return l, nil
		}
	}
	return new(SingleModel), errors.New("未找到模型")
}

// 模型反射一个新模型
func (rest *RestApi) newModel(routerName string) interface{} {
	cb, _ := rest.tableNameGetModelInfo(routerName)
	return rest.newInterface(cb.Model)
}

// 反射一个新数据
func (rest *RestApi) newInterface(input interface{}) interface{} {
	return rest.newType(input).Interface()
}

// 反射一个新类型
func (rest *RestApi) newType(input interface{}) reflect.Value {
	t := reflect.TypeOf(input)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return reflect.New(t)
}
