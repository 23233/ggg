package ut

import (
	"context"
	"github.com/iancoleman/strcase"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
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

// MGen2dSphere mongo 生成2dSphere索引
func MGen2dSphere(k string) mongo.IndexModel {
	return mongo.IndexModel{
		Keys: bson.M{k: "2dsphere"},
	}
}

// MCreateIndex 快捷创建索引
func MCreateIndex(ctx context.Context, db *mongo.Collection, indexs ...mongo.IndexModel) error {
	_, err := db.Indexes().CreateMany(ctx, indexs)
	return err
}
