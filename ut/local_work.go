package ut

import (
	"errors"
	"github.com/23233/ggg/logger"
	"sync"
	"sync/atomic"
	"time"
)

// 本地并发封装

// LocalWork 本地执行work T是原始数据 R是执行结果
type LocalWork[T any, R any] struct {
	Name string `json:"name"` // 任务名称
	Tid  string `json:"tid"`  // 任务id 需唯一

	NowCount     atomic.Int64 // 当前已执行个数
	FailCount    atomic.Int64 // 失败次数
	Running      bool         // 当前执行状态
	Concurrency  uint         // 线程数量
	ForceStop    bool         // 是否被强制停止
	PrintProcess bool         // 是否打印进度 默认开启

	SuccessStopLimit uint // 成功多少条后停止

	ChanData  []T // 待处理的数据
	call      func(item T) (R, error)
	Results   ThreadSafeArray[R]
	OnSuccess func(work *LocalWork[T, R], results []R) error

	startTime time.Time // 开始时间
}

func (c *LocalWork[T, R]) SetPrintProcess(PrintProcess bool) *LocalWork[T, R] {
	c.PrintProcess = PrintProcess
	return c
}

func (c *LocalWork[T, R]) SetCall(call func(item T) (R, error)) *LocalWork[T, R] {
	c.call = call
	return c
}

func (c *LocalWork[T, R]) SetOnSuccess(OnSuccess func(work *LocalWork[T, R], results []R) error) *LocalWork[T, R] {
	c.OnSuccess = OnSuccess
	return c
}

func (c *LocalWork[T, R]) SetChanData(ChanData []T) *LocalWork[T, R] {
	c.ChanData = ChanData
	return c
}

func (c *LocalWork[T, R]) GetConcurrency() uint {
	if c.Concurrency < 1 {
		return 1
	}
	return c.Concurrency
}

func (c *LocalWork[T, R]) SetConcurrency(Concurrency uint) *LocalWork[T, R] {
	c.Concurrency = Concurrency
	return c
}

func (c *LocalWork[T, R]) Reset() {
	c.Running = false
	c.Results = ThreadSafeArray[R]{}
	c.FailCount.Store(0)
	c.NowCount.Store(0)
}

func (c *LocalWork[T, R]) run() {
	defer func() {
		c.Running = false
	}()

	chanData := make(chan T, len(c.ChanData))
	for _, data := range c.ChanData {
		chanData <- data
	}
	close(chanData)

	var wg sync.WaitGroup
	stop := make(chan bool, 1) // 创建一个缓冲区为1的停止信号通道

	for i := 0; i < int(c.GetConcurrency()); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for data := range chanData {
				select {
				case <-stop:
					// 如果接收到停止信号，则退出当前 goroutine
					return
				default:
					// 正常执行任务
					r, err := c.call(data)
					if err != nil {
						c.FailCount.Add(1)
					} else {
						c.Results.Append(r)
					}
					c.NowCount.Add(1)

					if c.PrintProcess {
						logger.J.Infof("[%s]%s 进度%d/%d fail:%d", c.Tid, c.Name, c.NowCount.Load(), len(c.ChanData), c.FailCount.Load())
					}

					// 计算成功的数量
					successCount := c.NowCount.Load() - c.FailCount.Load()
					if c.SuccessStopLimit > 0 && uint(successCount) >= c.SuccessStopLimit {
						logger.J.Infof("[%s]%s 达到成功停止限制(%d)，发送停止信号", c.Tid, c.Name, c.SuccessStopLimit)
						stop <- true // 发送停止信号
						break
					}
				}
			}
		}()
	}
	wg.Wait()
	if c.SuccessStopLimit > 0 {
		logger.J.Infof("[%s]%s 执行完成 需求%d/%d条 失败%d条 成功%d条 耗时%s", c.Tid, c.Name, c.SuccessStopLimit, len(c.ChanData), c.FailCount.Load(), int64(len(c.ChanData))-c.FailCount.Load(), time.Since(c.startTime))
	} else {
		logger.J.Infof("[%s]%s 执行完成 总数%d条 失败%d条 成功%d条 耗时%s", c.Tid, c.Name, len(c.ChanData), c.FailCount.Load(), int64(len(c.ChanData))-c.FailCount.Load(), time.Since(c.startTime))
	}

	if c.OnSuccess != nil {
		err := c.OnSuccess(c, c.Results.GetAll())
		if err != nil {
			logger.J.ErrorE(err, "%s_%s 执行success方法失败", c.Name, c.Tid)
		}
	}
}

func (c *LocalWork[T, R]) Run(runSync bool) error {
	if c.Running {
		return errors.New("当前任务进行中")
	}
	c.Reset()
	c.Running = true
	c.ForceStop = false
	c.startTime = time.Now()

	if len(c.ChanData) < 1 {
		return errors.New("chanData为空 请填充内容后重试")
	}

	if runSync {
		c.run()
	} else {
		go c.run()
	}

	return nil
}

func NewLocalWorkFull[T any, R any](name, tid string, concurrency uint, todoDatas []T, call func(item T) (R, error)) *LocalWork[T, R] {
	inst := NewLocalWork[T, R](name, tid)
	inst.SetConcurrency(concurrency).SetCall(call).SetChanData(todoDatas)
	return inst
}

func NewLocalWork[T any, R any](name, tid string) *LocalWork[T, R] {
	inst := &LocalWork[T, R]{
		Name:         name,
		Tid:          tid,
		PrintProcess: true,
	}
	return inst
}
