package ut

import (
	"context"
	"errors"
	"github.com/23233/ggg/logger"
	"github.com/colduction/randomizer"
	"github.com/redis/rueidis/rueidiscompat"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 使用redis进行一些分布式任务 支持[]string{id}(数组切分) id起点_id终点(范围切分) redisKey zset消耗切分

type RedisWork[T any] struct {
	db                rueidiscompat.Cmdable
	PlatformName      string                                  // 平台名
	PlatformEng       string                                  // 平台英文名
	Name              string                                  // 任务名称
	Description       string                                  // 更多描述
	BulkIds           []string                                // id列表
	BulkStart         int64                                   // id起始值
	BulkEnd           int64                                   // id截止值
	DefaultStartCount int64                                   // 默认启动数
	DefaultEndCount   int64                                   // 默认截止数值
	NowCount          atomic.Int64                            // 当前已执行个数
	FileName          string                                  // 保存文件名
	Concurrency       uint                                    // 线程数量
	Results           []T                                     // 结果保存
	Running           bool                                    // 当前执行状态
	FetchCall         func(id string) ([]T, error)            // 获取单个
	FetchError        func(id string, err error) ([]T, error) // 获取错误
	ForceStop         bool                                    // 是否被强制停止
	RangeReverse      bool                                    // 范围旋转 默认从小到大 如果设置为true 则是从大到小
	InjectData        map[string]any                          //
	OnSuccess         func(self *RedisWork[T]) error          // 当成功时执行
	OnStartup         func(self *RedisWork[T]) error          // 启动时
	OnRangeStart      func(scopes []string, self *RedisWork[T]) []string
	OnRangSuccess     func(self *RedisWork[T], results []T, resultMap map[string][]T) error
	OnSingleSuccess   func(self *RedisWork[T], results []T, target string) error
	ItemDelayStart    time.Duration // 每获取一个的间隔时间起始点
	ItemDelayEnd      time.Duration // 每获取一个的间隔时间终止点
	RangeDelayStart   time.Duration
	RangeDelayEnd     time.Duration

	// 内部态
	startTime   time.Time // 开始时间
	resultsChan *ThreadSafeArray[T]
	rangeTime   time.Time
	redisKey    string
}

func (c *RedisWork[T]) Db() rueidiscompat.Cmdable {
	return c.db
}

func (c *RedisWork[T]) SetDb(db rueidiscompat.Cmdable) {
	c.db = db
}

func (c *RedisWork[T]) SetRedisKey(redisKey string) {
	c.redisKey = redisKey
}

func (c *RedisWork[T]) RedisKey() string {
	return c.redisKey
}

func (c *RedisWork[T]) GetItemDelay(start time.Duration, end time.Duration) time.Duration {
	if end == 0 || end < start {
		return start
	}
	return time.Duration(randomizer.RandInt64(int64(start), int64(end)))
}

func (c *RedisWork[T]) RunItemDelay() {
	delay := c.GetItemDelay(c.ItemDelayStart, c.ItemDelayEnd)
	if delay > 0 {
		time.Sleep(delay)
	}
}
func (c *RedisWork[T]) RunRangeDelay() {
	delay := c.GetItemDelay(c.RangeDelayStart, c.RangeDelayEnd)
	if delay > 0 {
		time.Sleep(delay)
	}
}

func (c *RedisWork[T]) StartTime() time.Time {
	return c.startTime
}

func (c *RedisWork[T]) SetOnStartup(OnStartup func(self *RedisWork[T]) error) {
	c.OnStartup = OnStartup
}

func (c *RedisWork[T]) GetNowCount() int64 {
	return c.NowCount.Load()
}

func (c *RedisWork[T]) GetLast() string {
	if len(c.BulkIds) >= 1 {
		return strconv.Itoa(len(c.BulkIds))
	}
	if len(c.redisKey) >= 1 {
		return strconv.FormatInt(c.DefaultEndCount, 10)
	}
	if c.BulkEnd >= 1 {
		return strconv.FormatInt(c.BulkEnd, 10)
	}
	return "0"
}

func (c *RedisWork[T]) GetFirst() string {
	if len(c.BulkIds) >= 1 {
		return c.BulkIds[0]
	}
	if len(c.redisKey) >= 1 {
		return strconv.FormatInt(c.DefaultStartCount, 10)
	}
	if c.BulkStart >= 1 {
		return strconv.FormatInt(c.BulkStart, 10)
	}
	return "0"
}

func (c *RedisWork[T]) SetOnSuccess(OnSuccess func(self *RedisWork[T]) error) {
	c.OnSuccess = OnSuccess
}

func (c *RedisWork[T]) getConcurrency() int {
	if c.Concurrency < 1 {
		return 1
	}
	return int(c.Concurrency)
}

func (c *RedisWork[T]) Run() error {
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

// RunRedisSetRange 按范围批次处理 会等待批次结束 性能若
func (c *RedisWork[T]) RunRedisSetRange(redisKey string) error {
	c.redisKey = redisKey
	return c.runRedisRange()
}

// RunRedisItem 按批次处理单个 会等待批次结束 性能稍弱
func (c *RedisWork[T]) RunRedisItem(redisKey string) error {
	c.redisKey = redisKey
	return c.runRedisItem()
}

// RunRedisItemParallel 多线程并行版 性能高
func (c *RedisWork[T]) RunRedisItemParallel(redisKey string) error {
	c.redisKey = redisKey
	return c.runRedisItemParallel()
}

func (c *RedisWork[T]) runBulk() error {

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

func (c *RedisWork[T]) runPreset() {
	c.Running = true
	c.ForceStop = false
	c.startTime = time.Now()
}

func (c *RedisWork[T]) clear() {
	c.Running = false
	c.resultsChan = nil
	c.NowCount.Store(0)
}

func (c *RedisWork[T]) runRange(start int64, end int64) error {
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
			bulkMap := NewConcurrentMap[string, []T]()

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
						if c.OnRangSuccess != nil {

							bulkMap.Set(bt, result)
						}
						if c.OnSingleSuccess != nil {
							err = c.OnSingleSuccess(c, result, bt)
							if err != nil {
								logger.J.ErrorE(err, "%s 单个%s任务成功回调返回错误", c.Name, bt)
							}
						}
						c.RunItemDelay()

					}
				}()

			}
			wg.Wait()
			logger.J.Infof("%s %d/%d %d条结果 耗时%s", c.Name, threadStart, threadEnd, bulkData.Count(), time.Since(startTime))

			if c.OnRangSuccess != nil {
				err := c.OnRangSuccess(c, bulkData.GetAll(), bulkMap.GetMap())
				if err != nil {
					logger.J.ErrorE(err, "%s 区间%d/%d任务返回错误", c.Name, threadStart, threadEnd)
				}
			}
			c.RunRangeDelay()
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

func (c *RedisWork[T]) runRedisRange() error {
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
			// https://github.com/redis/rueidis/issues/431 暂时不能设置为1
			scopes, err := c.db.ZPopMax(context.TODO(), c.redisKey).Result()
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
			bulkMap := NewConcurrentMap[string, []T]()

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
						if c.OnRangSuccess != nil {
							bulkMap.Set(bt, result)
						}
						if c.OnSingleSuccess != nil {
							err = c.OnSingleSuccess(c, result, bt)
							if err != nil {
								logger.J.ErrorE(err, "%s 单个%s任务成功回调返回错误", c.Name, bt)
							}
						}
						c.RunItemDelay()

					}
				}()

			}
			wg.Wait()
			logger.J.Infof("%s %d/%d %d条结果 耗时%s", c.Name, threadStart, threadEnd, bulkData.Count(), time.Since(startTime))

			if c.OnRangSuccess != nil {
				err = c.OnRangSuccess(c, bulkData.GetAll(), bulkMap.GetMap())
				if err != nil {
					logger.J.ErrorE(err, "%s 区间%d/%d任务返回错误", c.Name, threadStart, threadEnd)
				}
			}
			c.RunRangeDelay()

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

func (c *RedisWork[T]) runRedisItem() error {
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
			scopes, err := c.db.ZPopMax(context.TODO(), c.redisKey, int64(bulkSize)).Result()
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
			bulkMap := NewConcurrentMap[string, []T]()

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
						if c.OnRangSuccess != nil {
							bulkMap.Set(bt, result)
						}
						if c.OnSingleSuccess != nil {
							err = c.OnSingleSuccess(c, result, bt)
							if err != nil {
								logger.J.ErrorE(err, "%s 单个%s任务成功回调返回错误", c.Name, bt)
							}
						}
						c.RunItemDelay()

					}
				}()

			}

			wg.Wait()
			logger.J.Infof("%s %d条结果 耗时%s", c.Name, bulkData.Count(), time.Since(startTime))

			if c.OnRangSuccess != nil {
				err = c.OnRangSuccess(c, bulkData.GetAll(), bulkMap.GetMap())
				if err != nil {
					logger.J.ErrorE(err, "%s 任务返回错误", c.Name)
				}
			}
			c.RunRangeDelay()

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

func (c *RedisWork[T]) runRedisItemParallel() error {
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

	// 创建一个WaitGroup，用于等待所有goroutine完成
	var wg sync.WaitGroup

	go func() {
		defer c.clear()

		// 为每个goroutine生成任务并开始执行
		for i := 0; i < c.getConcurrency(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for {
					if c.ForceStop {
						break
					}

					// 每个goroutine自己获取任务
					scopes, err := c.db.ZPopMax(context.TODO(), c.redisKey, 10).Result()
					if err != nil || len(scopes) == 0 {
						break
					}

					ids := make([]string, len(scopes))
					for i, scope := range scopes {
						ids[i] = scope.Member
					}

					for _, id := range ids {
						if c.ForceStop {
							return
						}
						c.NowCount.Add(1)
						result, err := c.FetchCall(id)
						if err != nil && c.FetchError != nil {
							result, err = c.FetchError(id, err)
							if err != nil {
								continue
							}
						}

						if c.OnSingleSuccess != nil {
							err = c.OnSingleSuccess(c, result, id)
							if err != nil {
								logger.J.ErrorE(err, "%s 单个%s任务成功回调返回错误", c.Name, id)
							}
						}
						c.RunItemDelay()
					}
				}
			}()
		}
		wg.Wait()

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

func (c *RedisWork[T]) SetCall(call func(id string) ([]T, error)) {
	c.FetchCall = call
}

func NewRedisWork[T any](name string, redisClient rueidiscompat.Cmdable) *RedisWork[T] {
	return &RedisWork[T]{
		db:         redisClient,
		Name:       name,
		BulkIds:    make([]string, 0),
		InjectData: make(map[string]any),
		Results:    make([]T, 0),
	}
}
