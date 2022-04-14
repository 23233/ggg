package hotrank

import "github.com/pkg/errors"

var (
	EmptyError    = errors.New("数量为空")
	LikeExists    = errors.New("用户已点赞")
	LikeNotExists = errors.New("用户未点过赞")
)
