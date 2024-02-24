package ut

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewPruneCtxQuery(t *testing.T) {
	query := NewPruneCtxQuery()

	t.Run("测试时间格式", func(t *testing.T) {
		query.params = map[string]string{
			"update_at_gt":  "2024-02-24T00:00:00Z",
			"update_at_gte": "2024-02-24 00:00:00",
			"update_at_lt":  "2023-04-02T21:34:55",
		}
		and, or, err := query.PruneParseUrlParams()
		assert.NotEqual(t, nil, err)
		assert.Equal(t, 0, len(or))
		assert.Equal(t, 3, len(and))
		for _, kov := range and {
			if _, ok := kov.Value.(time.Time); !ok {
				t.Error(fmt.Sprintf("key:%s op:%s 应该是time.Time但现在不是", kov.Key, kov.Op))
			}
		}

	})

}
