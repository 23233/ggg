package idgenerator

// IDGenerator 定义了 ID 生成器的接口
type IDGenerator interface {
	GetNextID() (int64, error)
	GetCurrentNextID() (int64, error)
}
