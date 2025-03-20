package pipe

import (
	"context"
	"github.com/qiniu/qmgo"
)

// GetModelAllCount 使用 metadata 统计文档数量
// https://www.mongodb.com/docs/drivers/go/current/fundamentals/crud/read-operations/count/
func GetModelAllCount(ctx context.Context, db *qmgo.Database, modelId string) (int64, error) {
	cli, err := db.Collection(modelId).CloneCollection()
	if err != nil {
		return 0, err
	}
	return cli.EstimatedDocumentCount(ctx)
}

// MMN 获取模型名
func MMN[T *IMongoModel](input IMongoModel) string {
	return input.GetCollName()
}
