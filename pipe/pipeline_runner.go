package pipe

import (
	"github.com/kataras/iris/v12"
)

// RunnerContext 运行器上下文
// T 是原始传入值 运行的默认值
// P 是执行传入的参数 运行的配置
// D 是db 运行的依赖
// R 执行结果类型
type RunnerContext[T any, P any, D any, R any] struct {
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
	Desc string `json:"desc,omitempty"`
	call func(ctx iris.Context, origin T, params P, db D, more ...any) *RunResp[R]
}

func (c *RunnerContext[T, P, D, R]) SetKey(key string) *RunnerContext[T, P, D, R] {
	c.Key = key
	return c
}
func (c *RunnerContext[T, P, D, R]) SetName(name string) *RunnerContext[T, P, D, R] {
	c.Name = name
	return c
}
func (c *RunnerContext[T, P, D, R]) SetDesc(desc string) *RunnerContext[T, P, D, R] {
	c.Desc = desc
	return c
}
func (c *RunnerContext[T, P, D, R]) NewPipeErr(err error) *RunResp[R] {
	return newPipeErr[R](err)
}
func (c *RunnerContext[T, P, D, R]) Run(ctx iris.Context, origin T, params P, db D, more ...any) *RunResp[R] {
	return c.call(ctx, origin, params, db, more...)
}
