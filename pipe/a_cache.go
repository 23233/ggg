package pipe

import (
	"github.com/23233/ggg/ut"
	jsoniter "github.com/json-iterator/go"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"strings"
	"time"
)

var (
	// RequestCacheGet 请求缓存获取 必传params和db
	RequestCacheGet = &RunnerContext[any, *RequestCachePipe, rueidis.Client, *ParseResponse]{
		call: func(ctx iris.Context, origin any, params *RequestCachePipe, db rueidis.Client, more ...any) *RunResp[*ParseResponse] {
			if params == nil {
				params = new(RequestCachePipe)
			}
			cacheKey, err := params.GetCacheKey(ctx)
			if err != nil {
				return newPipeErr[*ParseResponse](err)
			}

			resp := db.Do(ctx, db.B().Get().Key(cacheKey).Build())
			if resp.Error() != nil {
				if resp.Error() == rueidis.Nil {
					// 如果缓存为空 则直接跳出到下一步
					return newPipeErr[*ParseResponse](nil)
				}
				return newPipeErr[*ParseResponse](resp.Error())
			}
			raw, err := resp.ToString()
			if err != nil {
				return newPipeErr[*ParseResponse](err)
			}
			var response *ParseResponse
			err = jsoniter.Unmarshal([]byte(raw), &response)
			if err != nil {
				return newPipeErr[*ParseResponse](err)
			}
			// 主动抛出跳出的错误
			return newPipeResultErr[*ParseResponse](response, PipeCacheHasError).SetBreak(true)
		},
		Name: "请求缓存获取",
		Key:  "request_cache_get",
	}
	// RequestCacheSet 请求缓存设置
	RequestCacheSet = &RunnerContext[*ParseResponse, *RequestCachePipe, rueidis.Client, *ParseResponse]{
		call: func(ctx iris.Context, origin *ParseResponse, params *RequestCachePipe, db rueidis.Client, more ...any) *RunResp[*ParseResponse] {

			cacheKey, err := params.GetCacheKey(ctx)
			if err != nil {
				return newPipeErr[*ParseResponse](err)
			}

			if _, ok := db.(rueidis.Client); !ok {
				return newPipeErr[*ParseResponse](errors.New("获取rdb失败"))
			}
			rdb := db.(rueidis.Client)

			mpByte, _ := jsoniter.Marshal(origin)
			rdbResp := rdb.Do(ctx, rdb.B().Set().Key(cacheKey).Value(string(mpByte)).ExSeconds(int64(params.GetCacheTime().Seconds())).Build())
			if rdbResp.Error() != nil {
				return newPipeErr[*ParseResponse](rdbResp.Error())
			}
			return newPipeResult[*ParseResponse](nil)
		},
		Name: "请求缓存设置",
		Key:  "request_cache_set",
	}
)

type RequestCachePipe struct {
	CacheTime  time.Duration     `json:"cache_time,omitempty"` // 不传 默认1分钟
	AttachData map[string]string `json:"attach_data,omitempty"`
}

func (c *RequestCachePipe) GetCacheTime() time.Duration {
	if c.CacheTime < 1 {
		return time.Minute
	}
	return c.CacheTime
}

func (c *RequestCachePipe) GetCacheKey(ctx iris.Context) (string, error) {
	cacheKey := ctx.Values().GetString("cache_key")
	if len(cacheKey) < 1 {
		// 生成缓存key
		var keys strings.Builder
		keys.WriteString(ctx.Request().RequestURI)
		if c.AttachData != nil {
			for k, v := range c.AttachData {
				keys.WriteString(k)
				keys.WriteString(v)
			}
		}

		cacheKey = ut.StrToB58(keys.String())
	}
	return cacheKey, nil
}
