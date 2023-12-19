package ut

import (
	"context"
	"errors"
	"github.com/23233/ggg/logger"
	"github.com/redis/rueidis/rueidiscompat"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Work[T any] struct {
	db                rueidiscompat.Cmdable
	PlatformName      string                                  // 平台名
	PlatformEng       string                                  // 平台英文名
	Name              string                                  // 任务名称
	BulkIds           []string                                // id列表
	BulkStart         int64                                   // id起始值
	BulkEnd           int64                                   // id截止值
	DefaultStartCount int64                                   // 默认启动数
	DefaultEndCount   int64                                   // 默认截止数值
	NowCount          atomic.Int64                            // 当前已执行个数
	FileName          string                                  // 保存文件名
	Concurrency       uint8                                   // 线程数量
	Results           []T                                     // 结果保存
	Running           bool                                    // 当前执行状态
	FetchCall         func(id string) ([]T, error)            // 获取单个
	FetchError        func(id string, err error) ([]T, error) // 获取错误
	ForceStop         bool                                    // 是否被强制停止
	RangeReverse      bool                                    // 范围旋转 默认从小到大 如果设置为true 则是从大到小
	InjectData        map[string]any                          //
	OnSuccess         func(self *Work[T]) error               // 当成功时执行
	OnStartup         func(self *Work[T]) error               // 启动时
	OnRangeStart      func(scopes []string, self *Work[T]) []string
	OnRangSuccess     func(self *Work[T], results []T) error

	// 内部态
	startTime   time.Time // 开始时间
	resultsChan *ThreadSafeArray[T]
	rangeTime   time.Time
	redisKey    string
}

func (c *Work[T]) StartTime() time.Time {
	return c.startTime
}

func (c *Work[T]) SetOnStartup(OnStartup func(self *Work[T]) error) {
	c.OnStartup = OnStartup
}

func (c *Work[T]) GetNowCount() int64 {
	return c.NowCount.Load()
}

func (c *Work[T]) GetLast() string {
	if len(c.BulkIds) >= 1 {
		return strconv.Itoa(len(c.BulkIds))
	}
	return "0"
}

func (c *Work[T]) GetFirst() string {
	if len(c.BulkIds) >= 1 {
		return c.BulkIds[0]
	}
	return "0"
}

func (c *Work[T]) SetOnSuccess(OnSuccess func(self *Work[T]) error) {
	c.OnSuccess = OnSuccess
}

func (c *Work[T]) getConcurrency() int {
	if c.Concurrency < 1 {
		return 1
	}
	return int(c.Concurrency)
}

func (c *Work[T]) Run() error {
	defer func() {
		if err := recover(); err != nil {
			logger.J.Errorf("work崩溃了 %v", err)
		}
	}()
	if c.Running {
		return errors.New("当前任务进行中")
	}
	c.runPreset()
	if len(c.BulkIds) >= 1 {
		return c.runBulk()
	}
	return c.runRange(c.BulkStart, c.BulkEnd)

}

func (c *Work[T]) RunRedisSetRange(redisKey string) error {
	c.redisKey = redisKey
	return c.runRedisRange()
}
func (c *Work[T]) RunRedisItem(redisKey string) error {
	c.redisKey = redisKey
	return c.runRedisItem()
}

func (c *Work[T]) runBulk() error {

	c.resultsChan = new(ThreadSafeArray[T])
	if c.OnStartup != nil {
		err := c.OnStartup(c)
		if err != nil {
			logger.J.ErrorE(err, "[%s] 任务启动时失败", c.Name)
			return err
		}
	}

	var wg sync.WaitGroup

	bulks := SplitArrayByFixedSize(c.BulkIds, c.getConcurrency())

	for _, d := range bulks {
		wg.Add(1)
		dataList := d
		go func() {
			defer wg.Done()

			for _, id := range dataList {
				if c.ForceStop {
					break
				}
				idStr := strings.TrimSpace(id)
				c.NowCount.Add(1)
				result, err := c.FetchCall(idStr)
				if err != nil {
					if c.FetchError != nil {
						result, err = c.FetchError(idStr, err)
						if err != nil {
							continue
						}
					} else {
						continue
					}
					continue
				}

				c.resultsChan.Append(result...)
			}
		}()
	}

	go func() {
		defer c.clear()

		wg.Wait()

		duration := time.Since(c.startTime)
		c.Results = c.resultsChan.GetAll()
		logger.J.Infof("[%s]任务执行结束 数据:%d条 预期:%d条 差额:%d 执行时间:%s", c.Name, len(c.Results), len(c.BulkIds), len(c.BulkIds)-len(c.Results), duration)

		//if len(c.Results) >= 1 {
		//	c.ToDisk()
		//}

		if c.OnSuccess != nil {
			err := c.OnSuccess(c)
			if err != nil {
				logger.J.ErrorE(err, "[%s]任务结束onSuccess返回错误", c.Name)
			}
		}

	}()

	return nil
}

func (c *Work[T]) runPreset() {
	c.Running = true
	c.ForceStop = false
	c.startTime = time.Now()
}

func (c *Work[T]) clear() {
	c.Running = false
	c.resultsChan = nil
	c.NowCount.Store(0)
}

func (c *Work[T]) runRange(start int64, end int64) error {
	if start > end {
		return errors.New("起始值大于结束值")
	}
	c.BulkStart = start
	c.BulkEnd = end

	if c.OnStartup != nil {
		err := c.OnStartup(c)
		if err != nil {
			logger.J.ErrorE(err, "[%s] 任务启动时失败", c.Name)
			return err
		}
	}

	// 将 rangeSize 设置为定值 15000
	rangeSize := int64(15000)
	// 计算任务的总数
	taskCount := int(math.Ceil(float64(end-start+1) / float64(rangeSize)))
	// 创建一个有缓冲的通道，用于存储任务
	tasks := make(chan struct {
		start int64
		end   int64
	}, taskCount)

	if c.RangeReverse {
		for i := 0; i < taskCount; i++ {
			taskEnd := end - int64(i)*rangeSize
			taskStart := taskEnd - rangeSize + 1

			if taskStart < start {
				taskStart = start
			}

			tasks <- struct {
				start int64
				end   int64
			}{taskStart, taskEnd}
		}
	} else {
		// 将任务分割，并将它们放入通道中
		for i := int64(0); i*rangeSize <= end-start; i++ {
			taskStart := start + i*rangeSize
			taskEnd := taskStart + rangeSize - 1

			if taskEnd > end {
				taskEnd = end
			}
			tasks <- struct {
				start int64
				end   int64
			}{taskStart, taskEnd}
		}
	}

	close(tasks)
	threading := c.getConcurrency()

	// 循环获取任务进行处理
	go func() {
		defer c.clear()
		for task := range tasks {
			if c.ForceStop {
				break
			}
			startTime := time.Now()
			threadStart := task.start
			threadEnd := task.end
			bulkTask := make(chan string, rangeSize)
			for ii := threadStart; ii <= threadEnd; ii++ {
				idStr := strconv.Itoa(int(ii))
				bulkTask <- idStr
			}
			close(bulkTask)

			bulkData := new(ThreadSafeArray[T])

			// 创建一个WaitGroup，用于等待所有goroutine完成
			var wg sync.WaitGroup
			for i := 0; i < threading; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for bt := range bulkTask {
						if c.ForceStop {
							break
						}
						c.NowCount.Add(1)
						result, err := c.FetchCall(bt)
						if err != nil {
							continue
						}
						bulkData.Append(result...)
					}
				}()

			}

			wg.Wait()
			logger.J.Infof("%s %d/%d %d条结果 耗时%s", c.Name, threadStart, threadEnd, bulkData.Count(), time.Since(startTime))

			if c.OnRangSuccess != nil {
				err := c.OnRangSuccess(c, bulkData.GetAll())
				if err != nil {
					logger.J.ErrorE(err, "%s 区间%d/%d任务返回错误", c.Name, threadStart, threadEnd)
				}
			}
		}

		duration := time.Since(c.startTime)
		logger.J.Infof("[%s]任务执行结束 执行时间:%s", c.Name, duration)

		if c.OnSuccess != nil {
			err := c.OnSuccess(c)
			if err != nil {
				logger.J.ErrorE(err, "[%s]任务结束onSuccess返回错误", c.Name)
			}
		}

	}()

	return nil
}

func (c *Work[T]) runRedisRange() error {
	if c.Running {
		return errors.New("当前任务进行中")
	}
	c.runPreset()

	if c.OnStartup != nil {
		err := c.OnStartup(c)
		if err != nil {
			logger.J.ErrorE(err, "[%s] 任务启动时失败", c.Name)
			return err
		}
	}

	// 循环获取任务进行处理
	go func() {
		defer c.clear()

		for {
			if c.ForceStop {
				break
			}
			// 从redis key最顶层获取一个区间
			scopes, err := c.db.ZPopMax(context.TODO(), c.redisKey, 1).Result()
			if err != nil {
				break
			}
			first := scopes[0]
			taskList := strings.Split(first.Member, "-")
			threadStart, _ := strconv.Atoi(taskList[0])
			threadEnd, _ := strconv.Atoi(taskList[1])
			startTime := time.Now()
			bulkTask := make(chan string, threadEnd-threadStart+1)
			for ii := threadStart; ii <= threadEnd; ii++ {
				idStr := strconv.Itoa(ii)
				bulkTask <- idStr
			}
			close(bulkTask)

			bulkData := new(ThreadSafeArray[T])

			// 创建一个WaitGroup，用于等待所有goroutine完成
			var wg sync.WaitGroup
			for i := 0; i < c.getConcurrency(); i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for bt := range bulkTask {
						if c.ForceStop {
							break
						}
						c.NowCount.Add(1)
						result, err := c.FetchCall(bt)
						if err != nil {
							continue
						}
						bulkData.Append(result...)
					}
				}()

			}

			wg.Wait()
			logger.J.Infof("%s %d/%d %d条结果 耗时%s", c.Name, threadStart, threadEnd, bulkData.Count(), time.Since(startTime))

			if c.OnRangSuccess != nil {
				err = c.OnRangSuccess(c, bulkData.GetAll())
				if err != nil {
					logger.J.ErrorE(err, "%s 区间%d/%d任务返回错误", c.Name, threadStart, threadEnd)
				}
			}
		}

		duration := time.Since(c.startTime)
		logger.J.Infof("[%s]任务执行结束 执行时间:%s", c.Name, duration)

		if c.OnSuccess != nil {
			err := c.OnSuccess(c)
			if err != nil {
				logger.J.ErrorE(err, "[%s]任务结束onSuccess返回错误", c.Name)
			}
		}

	}()

	return nil
}

func (c *Work[T]) runRedisItem() error {
	if c.Running {
		return errors.New("当前任务进行中")
	}
	c.runPreset()

	if c.OnStartup != nil {
		err := c.OnStartup(c)
		if err != nil {
			logger.J.ErrorE(err, "[%s] 任务启动时失败", c.Name)
			return err
		}
	}
	// 循环获取任务进行处理
	go func() {
		defer c.clear()

		for {
			if c.ForceStop {
				break
			}
			// 从redis key最顶层获取一个区间
			bulkSize := c.getConcurrency() * 5
			scopes, err := c.db.ZPopMin(context.TODO(), c.redisKey, int64(bulkSize)).Result()
			if err != nil {
				break
			}

			if len(scopes) < 1 {
				break
			}

			ids := make([]string, 0, len(scopes))
			for _, scope := range scopes {
				ids = append(ids, scope.Member)
			}

			if c.OnRangeStart != nil {
				ids = c.OnRangeStart(ids, c)
			}

			if ids == nil || len(ids) < 1 {
				c.NowCount.Add(int64(len(scopes)))
				logger.J.Infof("%s %d条id 已存在db %s 跳过", c.Name, len(scopes), c.InjectData["collName"].(string))
				continue
			}

			startTime := time.Now()

			bulkTask := make(chan string, bulkSize)
			for _, scope := range ids {
				bulkTask <- scope
			}
			close(bulkTask)

			bulkData := new(ThreadSafeArray[T])

			// 创建一个WaitGroup，用于等待所有goroutine完成
			var wg sync.WaitGroup
			for i := 0; i < c.getConcurrency(); i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for bt := range bulkTask {
						if c.ForceStop {
							break
						}
						c.NowCount.Add(1)
						result, err := c.FetchCall(bt)
						if err != nil {
							if c.FetchError != nil {
								result, err = c.FetchError(bt, err)
								if err != nil {
									continue
								}
							} else {
								continue
							}
						}
						bulkData.Append(result...)
					}
				}()

			}

			wg.Wait()
			logger.J.Infof("%s %d条结果 耗时%s", c.Name, bulkData.Count(), time.Since(startTime))

			if c.OnRangSuccess != nil {
				err = c.OnRangSuccess(c, bulkData.GetAll())
				if err != nil {
					logger.J.ErrorE(err, "%s 任务返回错误", c.Name)
				}
			}
		}

		duration := time.Since(c.startTime)
		logger.J.Infof("[%s]任务执行结束 执行时间:%s", c.Name, duration)

		if c.OnSuccess != nil {
			err := c.OnSuccess(c)
			if err != nil {
				logger.J.ErrorE(err, "[%s]任务结束onSuccess返回错误", c.Name)
			}
		}

	}()

	return nil
}

func (c *Work[T]) SetCall(call func(id string) ([]T, error)) {
	c.FetchCall = call
}

func NewWork[T any](name string, redisClient rueidiscompat.Cmdable) *Work[T] {
	return &Work[T]{
		db:         redisClient,
		Name:       name,
		BulkIds:    make([]string, 0),
		InjectData: make(map[string]any),
		Results:    make([]T, 0),
	}
}
