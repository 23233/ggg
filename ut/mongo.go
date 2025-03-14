package ut

import (
	"context"
	"reflect"

	"github.com/iancoleman/strcase"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 这个库主要是跟mongodb有关的操作

// strToTableName 模型名称转换成表名
func strToTableName(modelName string) string {
	return strcase.ToSnake(modelName)
}

// instanceToTableName 模型转表名
func instanceToTableName(instance interface{}) string {
	ct := reflect.TypeOf(instance)
	if ct.Kind() == reflect.Ptr {
		ct = ct.Elem()
	}
	return strcase.ToSnake(ct.Name())
}

// STN 模型字符串转表名 使用 strToTableName 方法
func STN(modelName string) string {
	return strToTableName(modelName)
}

// MTN 模型获取表名 使用 instanceToTableName 方法
func MTN(instance interface{}) string {
	return instanceToTableName(instance)
}

// MGenUnique mongo 生成唯一索引
func MGenUnique(k string, sparse bool) mongo.IndexModel {
	m := mongo.IndexModel{
		Keys:    bson.M{k: 1},
		Options: new(options.IndexOptions),
	}
	m.Options.SetUnique(true)
	if sparse {
		m.Options.SetSparse(true)
	}
	return m
}

// MGenNormal mongo 生成普通索引
func MGenNormal(k string) mongo.IndexModel {
	return mongo.IndexModel{
		Keys: bson.M{k: 1},
	}
}

// MGenNormals 复合索引 左匹配原则 请把最常用的字段放前面
func MGenNormals(keys ...string) mongo.IndexModel {
	var ops = bson.D{}
	for _, k := range keys {
		ops = append(ops, bson.E{Key: k, Value: 1})
	}
	return mongo.IndexModel{
		Keys: ops,
	}
}

// MGen2dSphere mongo 生成2dSphere索引
func MGen2dSphere(k string) mongo.IndexModel {
	return mongo.IndexModel{
		Keys: bson.M{k: "2dsphere"},
	}
}

// supportedLanguages 支持的文本索引语言映射
var supportedLanguages = map[string]bool{
	"da": true, "danish": true,
	"nl": true, "dutch": true,
	"en": true, "english": true,
	"fi": true, "finnish": true,
	"fr": true, "french": true,
	"de": true, "german": true,
	"hu": true, "hungarian": true,
	"it": true, "italian": true,
	"nb": true, "norwegian": true,
	"pt": true, "portuguese": true,
	"ro": true, "romanian": true,
	"ru": true, "russian": true,
	"es": true, "spanish": true,
	"sv": true, "swedish": true,
	"tr": true, "turkish": true,
}

// MGenText mongo 生成text索引
// https://www.mongodb.com/zh-cn/docs/manual/reference/text-search-languages/#std-label-text-search-languages
func MGenText(keys []string, language string) mongo.IndexModel {
	textIndex := bson.D{}
	for _, key := range keys {
		textIndex = append(textIndex, bson.E{Key: key, Value: "text"})
	}

	// 验证语言是否支持，不支持则使用 "none"
	if _, ok := supportedLanguages[language]; !ok {
		language = "none"
	}

	return mongo.IndexModel{
		Keys:    textIndex,
		Options: options.Index().SetDefaultLanguage(language),
	}
}

// MCreateIndex 快捷创建索引
func MCreateIndex(ctx context.Context, db *mongo.Collection, indexs ...mongo.IndexModel) error {
	_, err := db.Indexes().CreateMany(ctx, indexs)
	return err
}
