package stats

import (
	"context"
	"github.com/23233/ggg/ut"
	"github.com/go-redis/redis/v8"
	"os"
	"strconv"
	"testing"
	"time"
)

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func initRdb() *redis.Client {
	db, _ := strconv.Atoi(getEnv("REDISDB", "2"))
	return redis.NewClient(&redis.Options{
		Addr:     getEnv("REDISHOST", "127.0.0.1:6379"),
		Password: getEnv("REDISPD", ""),
		DB:       db,
		PoolSize: 100,
	})
}

func TestNewStats(t *testing.T) {
	rdb := initRdb()
	ctx := context.Background()
	m := NewStats("article", rdb)
	m.MustAdd(ctx, "今天1")
	m.MustAdd(ctx, "今天2")
	m.MustAdd(ctx, "今天3")
	m.MustAdd(ctx, "今天4")
	oKey := m.GenerateKey(time.Now().AddDate(0, 0, -1))
	m.MustAddAny(ctx, oKey, "昨天1")
	m.MustAddAny(ctx, oKey, "昨天2")
	m.MustAddAny(ctx, oKey, "昨天3")
	m.MustAddAny(ctx, oKey, "昨天4")
	// 获取上个月第一天
	mt := ut.GetFirstDateOfMonth().AddDate(0, -1, 0)
	mKey := m.GenerateKey(mt)
	m.MustAddAny(ctx, mKey, "上月1")
	m.MustAddAny(ctx, mKey, "上月2")
	m.MustAddAny(ctx, mKey, "上月3")
	m.MustAddAny(ctx, mKey, "上月4")
	m.MustAddAny(ctx, mKey, "上月5")

	// 测试汇总
	count, err := m.NowCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("今日总数 应该4 实际:%d", count)
	// 测试本月总数
	mc, err := m.GetNowWeekCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("本月总数 应该>=8 实际:%d", mc)
	// 测试上月总数
	sm, err := m.TimeRangerCount(ctx, mt, ut.GetFirstDateOfMonth())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("上月总数 应该>=5 实际:%d", sm)

}

func BenchmarkNewStats(b *testing.B) {
	rdb := initRdb()
	ctx := context.Background()
	m := NewStats("article", rdb)

	for i := 0; i < b.N; i++ {
		m.MustAdd(ctx, "随机111")
	}
}
