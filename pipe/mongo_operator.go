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
	"time"
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
	return MongoFilterGetOne[T](ctx, db, bson.M{ut.DefaultUidTag: uid})
}
func MongoFilterGetOne[T any](ctx context.Context, db *qmgo.Collection, filters bson.M) (*T, error) {
	var result = new(T)
	err := db.Find(ctx, filters).One(result)
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
func MongoIterateByBatch[T IMongoBase](ctx context.Context, db *qmgo.Collection, filter bson.M, selects bson.M, batchSize int64, processFunc func([]T) error) error {
	return MongoIterateByBatchUseStart[T](ctx, db, filter, selects, batchSize, primitive.NilObjectID, processFunc)
}

func MongoIterateByBatchUseStart[T IMongoBase](ctx context.Context, db *qmgo.Collection, filter bson.M, selects bson.M, batchSize int64, startId primitive.ObjectID, processFunc func([]T) error) error {
	lastID := startId

	for {
		var datas = make([]T, 0)
		ft := bson.M{
			"_id": bson.M{"$gt": lastID},
		}
		if filter != nil {
			for k, v := range filter {
				ft[k] = v
			}
		}
		baseQ := db.Find(ctx, ft).Sort("_id").Limit(batchSize)
		if selects != nil && len(selects) >= 1 {
			baseQ.Select(selects)
		}
		err := baseQ.All(&datas)
		if err != nil {
			return fmt.Errorf("error fetching records: %v", err)
		}

		if len(datas) == 0 {
			break
		}

		// 使用提供的处理函数处理这批数据
		err = processFunc(datas)
		if err != nil {
			return err
		}

		// 更新 lastID 以供下一次迭代使用
		lastID = datas[len(datas)-1].GetBase().Id
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
func MongoBulkInsertCountRetry[T any](ctx context.Context, db *qmgo.Collection, retryCount uint, retryInterval time.Duration, accounts ...T) (int, error) {
	result, err := mongoBulkInsert(ctx, db, accounts...)
	if err != nil {
		if err != PipeBulkEmptySuccessError {
			retryErr := ut.RetryFunc(func() error {
				result, err = mongoBulkInsert(ctx, db, accounts...)
				return err
			}, retryCount, retryInterval)
			if retryErr != nil {
				return 0, retryErr
			}
		}
	}
	return len(result.InsertedIDs), err

}
func MongoBulkInsertCountMustRetry[T any](ctx context.Context, db *qmgo.Collection, accounts ...T) (int, error) {
	return MongoBulkInsertCountRetry[T](ctx, db, 5, 1*time.Second, accounts...)
}
