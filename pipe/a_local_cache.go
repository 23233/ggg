package pipe

import (
	"github.com/bluele/gcache"
	"github.com/kataras/iris/v12"
	"time"
)

// LocalCachePipe 使用 https://github.com/bluele/gcache
type LocalCachePipe struct {
	ExpireDuration time.Duration  `json:"expire_duration,omitempty"` // 过期时间 默认1分钟
	KeyGen         *StrExpand     `json:"key_gen,omitempty"`         // key的来源
	EmptyRaise     bool           `json:"empty_raise,omitempty"`     // 获取时 为空报错
	Values         map[string]any `json:"values,omitempty"`
	DisWriteHeader bool           `json:"dis_write_header,omitempty"` // 不写入header
}

func (c *LocalCachePipe) GetExpire() time.Duration {
	if c.ExpireDuration < 1 {
		return time.Minute
	}
	return c.ExpireDuration
}

// 使用LRU 可在本地空间使用缓存

var (
	LocalCacheGet = &RunnerContext[any, *LocalCachePipe, gcache.Cache, any]{
		Name: "本地缓存获取",
		Key:  "local_cache_get",
		call: func(ctx iris.Context, origin any, params *LocalCachePipe, db gcache.Cache, more ...any) *RunResp[any] {

			if params == nil {
				return newPipeErr[any](PipePackParamsError)
			}
			gc := db

			k, err := params.KeyGen.Build()
			if err != nil {
				return newPipeErr[any](err)
			}
			v, err := gc.Get(k)
			if err != nil {
				if err != gcache.KeyNotFoundError {
					return newPipeErr[any](err)
				}
				if params.EmptyRaise {
					return newPipeErr[any](err)
				}
			} else {
				if !params.DisWriteHeader {
					ctx.Header("X-Cache", "1")
				}
			}

			return newPipeResult(v)
		},
	}
	pipeLocalCacheSet = &RunnerContext[any, *LocalCachePipe, gcache.Cache, any]{
		Name: "本地缓存设置",
		Key:  "local_cache_set",
		call: func(ctx iris.Context, origin any, params *LocalCachePipe, db gcache.Cache, more ...any) *RunResp[any] {
			if params == nil {
				return newPipeErr[any](PipePackParamsError)
			}

			k, err := params.KeyGen.Build()
			if err != nil {
				return newPipeErr[any](err)
			}

			err = db.SetWithExpire(k, params.Values, params.GetExpire())
			if err != nil {
				return newPipeErr[any](err)
			}
			return newPipeResult[any](params.Values)
		},
	}
)
