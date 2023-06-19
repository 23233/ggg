package pipe

import (
	"github.com/kataras/iris/v12"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"net/http"
	"strconv"
)

type RateLimitPipe struct {
	// <limit>-<period>
	// * "S": second
	// * "M": minute
	// * "H": hour
	// * "D": day
	//
	// Examples:
	//
	// * 5 reqs/second: "5-S"
	// * 10 reqs/minute: "10-M"
	// * 1000 reqs/hour: "1000-H"
	// * 2000 reqs/day: "2000-D"
	RatePeriod  string     `json:"rate_period,omitempty"`
	Store       string     `json:"store,omitempty"`        // 存储 暂时 memory
	KeyGen      *StrExpand `json:"key_gen,omitempty"`      // 判断key的来源
	WriteHeader bool       `json:"write_header,omitempty"` // 是否把rate状态写入到请求header中
}

func (c *RateLimitPipe) genClient() (*limiter.Limiter, error) {
	rate, err := limiter.NewRateFromFormatted(c.RatePeriod)
	if err != nil {
		return nil, err
	}
	store := memory.NewStore()
	return limiter.New(store, rate), nil
}

var (
	// RequestRate 限速器 使用 https://github.com/ulule/limiter 实现
	RequestRate = &RunnerContext[any, *RateLimitPipe, any, any]{
		Name: "请求限速",
		Key:  "request_rate",
		call: func(ctx iris.Context, origin any, params *RateLimitPipe, db any, more ...any) *RunResp[any] {

			if params == nil {
				return newPipeErr[any](PipePackParamsError)
			}
			k, err := params.KeyGen.Build()
			if err != nil {
				return newPipeErr[any](err)
			}

			client, err := params.genClient()
			if err != nil {
				return newPipeErr[any](err)
			}

			// 过程报错了 并不是说达到限速了
			rateCtx, err := client.Get(ctx, k)
			if err != nil {
				return newPipeErr[any](err)
			}

			if params.WriteHeader {
				ctx.Header("X-RateLimit-Limit", strconv.FormatInt(rateCtx.Limit, 10))
				ctx.Header("X-RateLimit-Remaining", strconv.FormatInt(rateCtx.Remaining, 10))
				ctx.Header("X-RateLimit-Reset", strconv.FormatInt(rateCtx.Reset, 10))
			}

			// 超出限速了
			if rateCtx.Reached {
				// 抛出错误
				return newPipeErr[any](PipeRatedError).SetReqCode(http.StatusTooManyRequests)
			}
			return newPipeResult[any](nil)
		},
	}
)
