package mab

import (
	"github.com/23233/ggg/ut"
	"github.com/bluele/gcache"
	"strings"
	"time"
)

var restCache gcache.Cache

// genCacheKey key尽量的短 所以使用 xxhash后进行base58
// 生成key需要的参数为 所有请求参数与额外参数 额外参数可以为用户id等
func genCacheKey(ReqParams string, otherInfo ...string) string {
	d := make([]string, 0, 1+len(otherInfo))
	d = append(d, ReqParams)
	d = append(d, otherInfo...)
	origin := strings.Join(d, "")
	return ut.StrToB58(origin)
}

// saveToCache 响应体保存到redis当中
func (rest *RestApi) saveToCache(keyName string, data interface{}, expireTime time.Duration) error {
	return restCache.SetWithExpire(keyName, data, expireTime)
}

// 删除key
func (rest *RestApi) deleteAtCache(keyName string) bool {
	return restCache.Remove(keyName)
}

func init() {
	restCache = gcache.New(50000).LRU().Build()
}
