package idgenerator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

// setupTestRedis 创建并返回一个 miniredis 服务器实例和一个连接到它的 redis.Client
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, redis.Cmdable) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Ping the server to ensure connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		s.Close()
		t.Fatalf("Failed to ping miniredis: %v", err)
	}

	return s, client
}

func TestNewRedisIDGenerator(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close() // Type assert to *redis.Client to access Close()

	t.Run("successful creation", func(t *testing.T) {
		key := "test_redis_new_success"
		initialValue := int64(100)
		gen, err := NewRedisIDGenerator(rdb, key, initialValue)
		if err != nil {
			t.Errorf("NewRedisIDGenerator() error = %v, wantErr nil", err)
			return
		}
		if gen == nil {
			t.Errorf("NewRedisIDGenerator() gen = nil, want non-nil")
			return
		}
		// 检查是否初始化了计数器 (通过 GetCurrentNextID 间接检查)
		currentID, err := gen.GetCurrentNextID()
		if err != nil {
			t.Errorf("GetCurrentNextID() after NewRedisIDGenerator error = %v", err)
		}
		if currentID != initialValue {
			t.Errorf("GetCurrentNextID() after NewRedisIDGenerator = %d, want %d", currentID, initialValue)
		}
	})

	t.Run("nil client", func(t *testing.T) {
		_, err := NewRedisIDGenerator(nil, "test_redis_nil_client", 1)
		if err == nil {
			t.Errorf("NewRedisIDGenerator() with nil client, error = nil, wantErr non-nil")
		}
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := NewRedisIDGenerator(rdb, "", 1)
		if err == nil {
			t.Errorf("NewRedisIDGenerator() with empty key, error = nil, wantErr non-nil")
		}
	})

	t.Run("negative initial value", func(t *testing.T) {
		_, err := NewRedisIDGenerator(rdb, "test_redis_neg_initial", -1)
		if err == nil {
			t.Errorf("NewRedisIDGenerator() with negative initial value, error = nil, wantErr non-nil")
		}
	})

	t.Run("lua script load fail", func(t *testing.T) {
		mr.Close()                         // Close the server to make ScriptLoad fail
		newMr, newRdb := setupTestRedis(t) // Setup a new one for other tests
		defer newMr.Close()
		defer newRdb.(*redis.Client).Close()

		TempRdbClient := redis.NewClient(&redis.Options{Addr: "localhost:12345"}) // Non-existent server
		defer TempRdbClient.Close()

		_, err := NewRedisIDGenerator(TempRdbClient, "test_lua_fail", 1)
		if err == nil {
			t.Errorf("NewRedisIDGenerator() with failing ScriptLoad, error = nil, wantErr non-nil")
		}
	})
}

func TestRedisGetNextID_Sequential(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()

	key := "test_redis_sequential"
	initialValue := int64(1)
	gen, err := NewRedisIDGenerator(rdb, key, initialValue)
	if err != nil {
		t.Fatalf("NewRedisIDGenerator() error = %v", err)
	}

	for i := 0; i < 10; i++ {
		var expectedID int64
		if i == 0 {
			expectedID = initialValue // First ID is the initial value itself
		} else {
			expectedID = initialValue + int64(i)
		}

		id, errGet := gen.GetNextID()
		if errGet != nil {
			t.Errorf("GetNextID() iteration %d: error = %v", i, errGet)
			return
		}
		if id != expectedID {
			t.Errorf("GetNextID() iteration %d: got %d, want %d", i, id, expectedID)
		}

		currentNext, errCurr := gen.GetCurrentNextID()
		if errCurr != nil {
			t.Errorf("GetCurrentNextID() iteration %d error = %v", i, errCurr)
		}
		if currentNext != expectedID+1 {
			t.Errorf("GetCurrentNextID() iteration %d: got %d, want %d", i, currentNext, expectedID+1)
		}
	}
}

func TestRedisGetNextID_Concurrent(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()

	key := "test_redis_concurrent"
	initialValue := int64(1000)
	gen, err := NewRedisIDGenerator(rdb, key, initialValue)
	if err != nil {
		t.Fatalf("NewRedisIDGenerator() error = %v", err)
	}

	numGoroutines := 20
	idsPerGoroutine := 10
	totalIDs := numGoroutines * idsPerGoroutine

	var wg sync.WaitGroup
	generatedIDs := make(map[int64]struct{})
	var mu sync.Mutex // Protects generatedIDs map

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, getErr := gen.GetNextID()
				if getErr != nil {
					t.Errorf("Goroutine %d, iteration %d: GetNextID() error = %v", routineID, j, getErr)
					return
				}
				mu.Lock()
				if _, exists := generatedIDs[id]; exists {
					t.Errorf("Goroutine %d, iteration %d: Duplicate ID generated: %d", routineID, j, id)
				}
				generatedIDs[id] = struct{}{}
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	if len(generatedIDs) != totalIDs {
		t.Errorf("Expected %d unique IDs, but got %d", totalIDs, len(generatedIDs))
	}

	// Check if all IDs from initialValue to initialValue + totalIDs - 1 are present
	// The Lua script for GetNextID returns initialValue on the first call if new,
	// and then subsequent INCRs return initialValue+1, initialValue+2, etc.
	for i := 0; i < totalIDs; i++ {
		expectedID := initialValue + int64(i)
		if _, exists := generatedIDs[expectedID]; !exists {
			t.Errorf("Expected ID %d to be generated, but it was not", expectedID)
		}
	}

	currentNext, errCurr := gen.GetCurrentNextID()
	if errCurr != nil {
		t.Errorf("GetCurrentNextID() after concurrent test error = %v", errCurr)
	}
	expectedCurrentNext := initialValue + int64(totalIDs)
	if currentNext != expectedCurrentNext {
		t.Errorf("GetCurrentNextID() after concurrent test: got %d, want %d", currentNext, expectedCurrentNext)
	}
}

func TestRedisGetCurrentNextID_NotInitialized(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()

	key := "test_redis_current_not_init"
	initialValue := int64(500)
	gen, err := NewRedisIDGenerator(rdb, key, initialValue)
	if err != nil {
		t.Fatalf("NewRedisIDGenerator() error = %v", err)
	}

	// Key should not exist in Redis yet, so GetCurrentNextID should return initialValue
	currentID, err := gen.GetCurrentNextID()
	if err != nil {
		t.Errorf("GetCurrentNextID() for new key error = %v", err)
	}
	if currentID != initialValue {
		t.Errorf("GetCurrentNextID() for new key: got %d, want %d", currentID, initialValue)
	}

	// Call GetNextID once, which should initialize the key in Redis to initialValue
	firstID, err := gen.GetNextID()
	if err != nil {
		t.Fatalf("GetNextID() failed: %v", err)
	}
	if firstID != initialValue {
		t.Fatalf("First GetNextID() got %d, want %d", firstID, initialValue)
	}

	// Now GetCurrentNextID should return initialValue + 1
	currentIDAfterGet, err := gen.GetCurrentNextID()
	if err != nil {
		t.Errorf("GetCurrentNextID() after one GetNextID error = %v", err)
	}
	if currentIDAfterGet != initialValue+1 {
		t.Errorf("GetCurrentNextID() after one GetNextID: got %d, want %d", currentIDAfterGet, initialValue+1)
	}
}

func TestRedisGetNextID_LuaScriptReload(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()

	key := "test_redis_lua_reload"
	initialValue := int64(777)
	gen, err := NewRedisIDGenerator(rdb, key, initialValue)
	if err != nil {
		t.Fatalf("NewRedisIDGenerator() error = %v", err)
	}

	// 1. Get an ID normally, this should work and the script SHA is known.
	firstID, err := gen.GetNextID()
	if err != nil {
		t.Fatalf("First GetNextID() failed: %v", err)
	}
	if firstID != initialValue {
		t.Fatalf("First GetNextID() got %d, want %d", firstID, initialValue)
	}

	// 2. Simulate Redis flushing scripts (miniredis specific command)
	mr.FlushAll() // This will clear scripts known by miniredis by their SHA.
	// Note: For a real Redis, this would be SCRIPT FLUSH. Miniredis FlushAll clears all data & scripts.
	// So we need to re-set up the initial value expectation for the script.

	// 3. Try to get the next ID. This should trigger NOSCRIPT, reload, and succeed.
	// The Lua script will re-initialize because FlushAll cleared the key.
	secondID, err := gen.GetNextID()
	if err != nil {
		t.Fatalf("Second GetNextID() after script flush failed: %v", err)
	}
	// Since FlushAll also deleted the key, the Lua script will initialize it to initialValue again.
	if secondID != initialValue {
		t.Fatalf("Second GetNextID() after script flush got %d, want %d (re-initialized)", secondID, initialValue)
	}

	// 4. Get another ID to ensure it increments correctly after reload
	thirdID, err := gen.GetNextID()
	if err != nil {
		t.Fatalf("Third GetNextID() after script reload failed: %v", err)
	}
	if thirdID != initialValue+1 {
		t.Fatalf("Third GetNextID() after script reload got %d, want %d", thirdID, initialValue+1)
	}

	// Check that the SHA was indeed reloaded (it might be the same or different, just check it's not empty)
	redisGen := gen.(*RedisIDGenerator) // Type assertion
	if redisGen.luaScriptSHA == "" {
		t.Errorf("luaScriptSHA is empty after supposed reload")
	}
}
