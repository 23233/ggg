package pipe

import (
	"context"
	"fmt"
	"github.com/23233/ggg/logger"
	"github.com/23233/ggg/ut"
	"github.com/pkg/errors"
	"github.com/qiniu/qmgo"
	opta "github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 封装在mongodb中常见操作方法

func MongoFilters[T any](ctx context.Context, db *qmgo.Collection, filters bson.M) ([]T, error) {
	var result = make([]T, 0)
	err := db.Find(ctx, filters).All(&result)
	if err != nil {
		return nil, err
	}
	return result, err
}

// MongoGetOne 传入实例 返回new之后的指针
func MongoGetOne[T any](ctx context.Context, db *qmgo.Collection, uid string) (*T, error) {
	var result = new(T)
	err := db.Find(ctx, bson.M{ut.DefaultUidTag: uid}).One(result)
	if err != nil {
		return result, err
	}
	return result, err
}
func MongoRandom[T any](ctx context.Context, db *qmgo.Collection, filters bson.D, count int) ([]T, error) {
	pipeline := mongo.Pipeline{
		{{"$match", filters}},
		{{"$sample", bson.D{{"size", count}}}},
	}
	var accounts = make([]T, 0)
	err := db.Aggregate(ctx, pipeline).All(&accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}
func MongoUpdateOne(ctx context.Context, db *qmgo.Collection, uid string, pack bson.M) error {
	return db.UpdateOne(ctx, bson.M{ut.DefaultUidTag: uid}, bson.M{"$set": pack})
}
func MongoIterateByBatch[T IMongoBase](ctx context.Context, db *qmgo.Collection, batchSize int64, processFunc func([]T) error) error {
	lastID := primitive.NilObjectID

	for {
		var accounts = make([]T, 0)

		err := db.Find(ctx, bson.M{"_id": bson.M{"$gt": lastID}}).Sort("_id").Limit(batchSize).All(&accounts)
		if err != nil {
			return fmt.Errorf("error fetching records: %v", err)
		}

		if len(accounts) == 0 {
			break
		}

		// 使用提供的处理函数处理这批数据
		err = processFunc(accounts)
		if err != nil {
			return err
		}

		// 更新 lastID 以供下一次迭代使用
		lastID = accounts[len(accounts)-1].GetBase().Id
	}

	return nil
}
func MongoBulkInsert[T any](ctx context.Context, db *qmgo.Collection, accounts ...T) error {
	_, err := mongoBulkInsert(ctx, db, accounts...)
	return err
}
func mongoBulkInsert[T any](ctx context.Context, db *qmgo.Collection, accounts ...T) (*qmgo.InsertManyResult, error) {
	// 批量新增
	opts := opta.InsertManyOptions{}
	// order默认为true 则一个报错后面都停 所以设置为false
	opts.InsertManyOptions = options.InsertMany().SetOrdered(false)

	result, err := db.InsertMany(ctx, accounts, opts)
	if err != nil {
		var bulkErr mongo.BulkWriteException
		if ok := errors.As(err, &bulkErr); !ok {
			logger.J.ErrorE(err, "批量插入发生非bulkWrite异常")
			return result, err
		}
		if len(result.InsertedIDs) < 1 {
			return result, PipeBulkEmptySuccessError
		}
	}
	logger.J.Infof("批量预期传入 %d条 插入成功 %d 条", len(accounts), len(result.InsertedIDs))
	return result, nil
}
func MongoBulkInsertCount[T any](ctx context.Context, db *qmgo.Collection, accounts ...T) (int, error) {
	result, err := mongoBulkInsert(ctx, db, accounts...)
	if err != nil {
		if err != PipeBulkEmptySuccessError {
			return 0, err
		}
	}
	return len(result.InsertedIDs), err
}
