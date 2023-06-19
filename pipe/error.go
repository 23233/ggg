package pipe

import "github.com/pkg/errors"

var (
	PipeSelectorParamsError  = errors.New("操作序列选择器参数错误")
	PipeDepNotFound          = errors.New("操作序列依赖项未找到")
	PipeSelectorNotFoundData = errors.New("操作序列获取内容为空")
	PipePackError            = errors.New("操作序列包参数错误")
	PipeDepError             = errors.New("操作序列依赖错误")
	PipeDbError              = errors.New("操作序列db报错")
	PipeOriginError          = errors.New("操作序列原始数据获取失败")
	PipePackParamsError      = errors.New("操作序列包内容参数错误")
	PipeParamsError          = errors.New("操作序列参数错误")
	PipeBreakError           = errors.New("操作序列主动跳出错误")
	PipeCacheHasError        = errors.New("有缓存主动返回")
	PipeRunAfterError        = errors.New("操作序列包执行错误")
	PipeRunAfterEmptyError   = errors.New("操作序列包解析后为空错误")
	PipeRatedError           = errors.New("太快了 请慢一点~")
)
