package pipe

import (
	"github.com/go-redis/redis/v8"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"github.com/redis/rueidis"
	"strings"
)

type RedisOperates struct {
	Records []*RedisOperate `json:"records,omitempty" bson:"records,omitempty"`
}

type RedisKv struct {
	Key   string `json:"key,omitempty"`   // key 必须为字符串
	Value string `json:"value,omitempty"` // value 必须为字符串
}

func (c *RedisKv) Get(uid string) ([]string, error) {
	var st = make([]string, 0, 2)

	if len(c.Key) > 0 {
		st = append(st, uid+":"+c.Key)
	}

	if len(c.Value) > 0 {
		st = append(st, c.Value)
	}

	return st, nil
}

// RedisOperate redis操作 必传uid 一定要注意attach命令需要区分大小写 所有的key都会自动加上 reqUid:key
type RedisOperate struct {
	// 最终组合成{Name}: {COMMAND} [{mid:key} {value}] {runAttach:{key | KEY} {value} 模板替换} {EXPIRY} = {ResultType}
	Uid        string     `json:"mid,omitempty"`         // 用户id 自动解析到key上面去
	Type       string     `json:"type,omitempty"`        // 类型 string, list, set, zSet, hash, stream, json? 暂定 可不填
	Command    string     `json:"command,omitempty"`     // 命令 https://redis.io/commands/ 对应上面Type类型 必填
	Kvs        []RedisKv  `json:"kvs,omitempty"`         // key value 对应关系
	Attach     *StrExpand `json:"runAttach,omitempty"`   // 模板key 区分大小写
	Expiry     string     `json:"expiry,omitempty"`      // NX XX GT LT
	ResultType string     `json:"result_type,omitempty"` // 结果类型 推荐直接为空 (string|[]) (int 8 16 32 64|[]|{}) bool (number float 32 64|[]) {}string 为空则any直接输出
	AllowNil   bool       `json:"allow_nil,omitempty"`   // 允许结果为空
}

type redisAb struct {
	Command string
	Keys    [][]string
	Args    []string
}

func (c *RedisOperate) Build() (*redisAb, error) {
	if len(c.Uid) < 1 {
		return nil, errors.New("redis Uid 参数错误 ")
	}
	if len(c.Command) < 1 {
		return nil, errors.New("redis 命令为空")
	}
	if len(c.Kvs) < 1 {
		return nil, errors.New("redis kv为空")
	}

	var ab = new(redisAb)
	ab.Command = strings.ToUpper(c.Command)

	for _, kv := range c.Kvs {
		tl, err := kv.Get(c.Uid)
		if err != nil {
			return nil, err
		}
		if len(tl) >= 1 {
			ab.Keys = append(ab.Keys, tl)
		}
	}

	if c.Attach != nil {
		attach, err := c.Attach.Build()
		if err != nil {
			return nil, err
		}
		ab.Args = append(ab.Args, strings.Split(attach, " ")...)
	}

	if len(c.Expiry) > 0 {
		ab.Args = append(ab.Args, strings.ToUpper(c.Expiry))
	}

	return ab, nil

}
func (c *RedisOperate) RespParse(resp rueidis.RedisResult) (result any, err error) {
	// 返回类型转换
	switch strings.ToLower(c.ResultType) {
	case "string":
		result, err = resp.ToString()
		break
	case "[]string":
		result, err = resp.AsStrSlice()
		break
	case "int64":
	case "int32":
	case "int16":
	case "int8":
	case "int":
		result, err = resp.AsInt64()
		break
	case "[]int64":
	case "[]int32":
	case "[]int16":
	case "[]int8":
	case "[]int":
		result, err = resp.AsIntSlice()
		break
	case "bool":
		result, err = resp.AsBool()
		break
	case "number":
	case "float":
	case "float32":
	case "float64":
		result, err = resp.AsFloat64()
		break
	case "[]float":
	case "[]float32":
	case "[]float64":
	case "[]number":
		result, err = resp.AsFloatSlice()
		break
	case "[]":
	case "array":
		result, err = resp.ToArray()
		break
	case "map":
		result, err = resp.ToMap()
		break
	case "{}string":
		result, err = resp.AsStrMap()
		break
	case "{}int":
	case "{}int8":
	case "{}int16":
	case "{}int32":
	case "{}int64":
		result, err = resp.AsIntMap()
		break
	default:
		result, err = resp.ToAny()
		break
	}
	return
}

// redis操作
// https://redis.io/commands/
// redis是没有一致性事务 虽然有事务方法 但是已被执行的命令无法撤回

var (
	RedisCommand = &RunnerContext[any, *RedisOperate, rueidis.Client, any]{
		Name: "redis单条操作",
		Key:  "redis_command",
		call: func(ctx iris.Context, origin any, params *RedisOperate, db rueidis.Client, more ...any) *RunResp[any] {

			if params == nil {
				return newPipeErr[any](PipePackParamsError)
			}
			ab, err := params.Build()
			if err != nil {
				return newPipeErr[any](err)
			}
			rdb := db

			// Arbitrary 是支持原始命令的 看这里 https://github.com/redis/rueidis#arbitrary-command
			redisCmd := rdb.B().Arbitrary(ab.Command)
			for _, kv := range ab.Keys {
				redisCmd.Keys(kv[0])
				if len(kv) >= 2 {
					redisCmd.Args(kv[1])
				}
			}
			redisCmd.Args(ab.Args...)
			resp := rdb.Do(ctx, redisCmd.Build())
			if resp.Error() != nil {
				if resp.Error() == redis.Nil {
					if !params.AllowNil {
						return newPipeErr[any](resp.Error())
					}
					return newPipeErr[any](nil)
				}
				return newPipeErr[any](resp.Error())
			}

			result, err := params.RespParse(resp)
			return newPipeResultErr(result, err)
		},
	}
	RedisCommands = &RunnerContext[any, *RedisOperates, rueidis.Client, []any]{
		Name: "多条redis操作",
		Key:  "redis_commands",
		call: func(ctx iris.Context, origin any, params *RedisOperates, db rueidis.Client, more ...any) *RunResp[[]any] {
			rdb := db
			cmdList := make(rueidis.Commands, 0, len(params.Records))
			for _, rc := range params.Records {
				ab, err := rc.Build()
				if err != nil {
					return newPipeErr[[]any](err)
				}
				redisCmd := rdb.B().Arbitrary(ab.Command)
				for _, kv := range ab.Keys {
					redisCmd.Keys(kv[0])
					if len(kv) >= 2 {
						redisCmd.Args(kv[1])
					}
				}
				redisCmd.Args(ab.Args...)
				cmdList = append(cmdList, redisCmd.Build())
			}

			if len(cmdList) < 1 {
				return newPipeErr[[]any](errors.New("redis命令组为空"))

			}

			var result = make([]any, 0, len(cmdList))

			for index, resp := range rdb.DoMulti(ctx, cmdList...) {
				if resp.Error() != nil {
					if resp.Error() == redis.Nil {
						if !params.Records[index].AllowNil {
							return newPipeErr[[]any](resp.Error())
						}
						result = append(result, nil)
						continue
					}
					return newPipeErr[[]any](resp.Error())
				}

				r, err := params.Records[index].RespParse(resp)
				if err != nil {
					return newPipeErr[[]any](err)
				}
				result = append(result, r)
			}

			return newPipeResult(result)
		},
	}
)
