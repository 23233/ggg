package idgenerator

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedisForRoomManager is a helper to create a miniredis instance for tests.
func setupTestRedisForRoomManager(t *testing.T) (*miniredis.Miniredis, redis.Cmdable) {
	s, err := miniredis.Run()
	require.NoError(t, err, "Failed to start miniredis")

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Ping(ctx).Err(), "Failed to ping miniredis")

	// Clean up global state before each test setup
	getAndMarkUsedScriptSHA = ""
	releaseScriptSHA = ""

	return s, client
}

// setupManager is a helper to create a RoomIDManager with a specific strategy.
func setupManager(t *testing.T, rdb redis.Cmdable, baseKey string, strategy GenerationStrategy) *RoomIDManager {
	manager, err := NewRoomIDManager(rdb, baseKey, WithGenerationStrategy(strategy))
	require.NoError(t, err)
	err = manager.LoadScripts(context.Background())
	require.NoError(t, err)
	return manager
}

func TestNewRoomIDManager_WithOptions(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()

	t.Run("creation with default options", func(t *testing.T) {
		manager, err := NewRoomIDManager(rdb, "default_opts")
		require.NoError(t, err)
		assert.Equal(t, PreferReleased, manager.strategy)
		assert.Equal(t, int64(defaultMaxReleasedIDsToStore), manager.maxReleasedIDsToStore)
		assert.Equal(t, defaultRandomGenerationAttempts, manager.randomAttempts)
	})

	t.Run("creation with custom options", func(t *testing.T) {
		manager, err := NewRoomIDManager(rdb, "custom_opts",
			WithGenerationStrategy(PreferNew),
			WithMaxReleasedIDsToStore(50),
			WithRandomGenerationAttempts(10),
		)
		require.NoError(t, err)
		assert.Equal(t, PreferNew, manager.strategy)
		assert.Equal(t, int64(50), manager.maxReleasedIDsToStore)
		assert.Equal(t, 10, manager.randomAttempts)
	})

	t.Run("nil client", func(t *testing.T) {
		_, err := NewRoomIDManager(nil, "test_nil_client")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "redis client cannot be nil")
	})

	t.Run("empty baseKey", func(t *testing.T) {
		_, err := NewRoomIDManager(rdb, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "baseKey cannot be empty")
	})
}

func TestRoomIDManager_GetNextAvailableRoomID_Strategy(t *testing.T) {
	// This test covers the core logic of getting an ID based on strategy.
	t.Run("Strategy: PreferReleased", func(t *testing.T) {
		mr, rdb := setupTestRedisForRoomManager(t)
		defer mr.Close()
		defer rdb.(*redis.Client).Close()
		ctx := context.Background()

		baseKey := "test_prefer_released"
		manager := setupManager(t, rdb, baseKey, PreferReleased)
		defer manager.cleanupRedisKeys(ctx)

		// 1. Get a new ID and release it
		id1, err := manager.GetNextAvailableRoomID(ctx)
		require.NoError(t, err)
		err = manager.ReleaseRoomID(ctx, id1)
		require.NoError(t, err)

		// 2. Get next ID - should be the released one
		id2, err := manager.GetNextAvailableRoomID(ctx)
		require.NoError(t, err)
		assert.Equal(t, id1, id2, "Should reuse the released ID with PreferReleased strategy")
	})

	t.Run("Strategy: PreferNew", func(t *testing.T) {
		mr, rdb := setupTestRedisForRoomManager(t)
		defer mr.Close()
		defer rdb.(*redis.Client).Close()
		ctx := context.Background()

		baseKey := "test_prefer_new"
		manager := setupManager(t, rdb, baseKey, PreferNew)
		defer manager.cleanupRedisKeys(ctx)

		// 1. Get a new ID and release it
		id1, err := manager.GetNextAvailableRoomID(ctx)
		require.NoError(t, err)
		err = manager.ReleaseRoomID(ctx, id1)
		require.NoError(t, err)

		// 2. Get next ID - should be a new one, not the released one
		id2, err := manager.GetNextAvailableRoomID(ctx)
		require.NoError(t, err)
		assert.NotEqual(t, id1, id2, "Should get a new ID with PreferNew strategy, not the released one")
	})
}

func TestRoomIDManager_Fallback_PreferReleased(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()
	ctx := context.Background()

	baseKey := "test_released_fallback"
	manager := setupManager(t, rdb, baseKey, PreferReleased)
	defer manager.cleanupRedisKeys(ctx)

	// Released list is empty initially. It should fall back to getting a new ID.
	id, err := manager.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	assert.True(t, id >= minRoomIDValue && id <= maxRoomIDValue, "Should generate a new ID when released list is empty")

	count, err := manager.CountUsedRoomIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRoomIDManager_Concurrent_GetNext(t *testing.T) {
	runConcurrencyTest := func(t *testing.T, strategy GenerationStrategy) {
		mr, rdb := setupTestRedisForRoomManager(t)
		defer mr.Close()
		defer rdb.(*redis.Client).Close()
		ctx := context.Background()

		baseKey := fmt.Sprintf("test_concurrent_%v", strategy)
		manager := setupManager(t, rdb, baseKey, strategy)
		defer manager.cleanupRedisKeys(ctx)

		numGoroutines := 20
		idsPerGoroutine := 5
		totalIDsToGet := numGoroutines * idsPerGoroutine

		var wg sync.WaitGroup
		generatedIDs := make(map[int64]struct{})
		var mu sync.Mutex
		errors := make(chan error, totalIDsToGet)
		idsChan := make(chan int64, totalIDsToGet)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < idsPerGoroutine; j++ {
					id, getErr := manager.GetNextAvailableRoomID(ctx)
					if getErr != nil {
						errors <- getErr
						return
					}
					idsChan <- id
				}
			}()
		}
		wg.Wait()
		close(errors)
		close(idsChan)

		for err := range errors {
			require.NoError(t, err, "Concurrent GetNextAvailableRoomID resulted in an error")
		}

		for id := range idsChan {
			mu.Lock()
			_, exists := generatedIDs[id]
			assert.False(t, exists, "Duplicate ID generated concurrently: %d", id)
			generatedIDs[id] = struct{}{}
			mu.Unlock()
			assert.True(t, id >= minRoomIDValue && id <= maxRoomIDValue, "ID %d out of range", id)
		}

		assert.Equal(t, totalIDsToGet, len(generatedIDs), "Incorrect number of unique IDs generated")

		usedCount, err := manager.CountUsedRoomIDs(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(totalIDsToGet), usedCount, "Final used count mismatch")
	}

	t.Run("Strategy: PreferReleased", func(t *testing.T) {
		runConcurrencyTest(t, PreferReleased)
	})

	t.Run("Strategy: PreferNew", func(t *testing.T) {
		runConcurrencyTest(t, PreferNew)
	})
}

func TestRoomIDManager_Concurrent_GetAndRelease(t *testing.T) {
	runGetAndReleaseTest := func(t *testing.T, strategy GenerationStrategy) {
		mr, rdb := setupTestRedisForRoomManager(t)
		defer mr.Close()
		defer rdb.(*redis.Client).Close()
		ctx := context.Background()

		baseKey := fmt.Sprintf("test_concurrent_get_release_%v", strategy)
		manager := setupManager(t, rdb, baseKey, strategy)
		defer manager.cleanupRedisKeys(ctx)

		numCycles := 50
		var wg sync.WaitGroup
		errChan := make(chan error, numCycles*2)

		initialIDs := make([]int64, 0, numCycles/2)
		for i := 0; i < numCycles/2; i++ {
			id, getErr := manager.GetNextAvailableRoomID(ctx)
			require.NoError(t, getErr, "Failed to get initial ID for release")
			initialIDs = append(initialIDs, id)
		}
		initialIDCount, _ := manager.CountUsedRoomIDs(ctx)
		assert.Equal(t, int64(len(initialIDs)), initialIDCount)

		for i := 0; i < numCycles; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				_, getErr := manager.GetNextAvailableRoomID(ctx)
				if getErr != nil {
					errChan <- fmt.Errorf("getter error: %w", getErr)
				}
			}()

			go func(idx int) {
				defer wg.Done()
				if idx < len(initialIDs) {
					releaseErr := manager.ReleaseRoomID(ctx, initialIDs[idx])
					if releaseErr != nil {
						errChan <- fmt.Errorf("releaser error for ID %d: %w", initialIDs[idx], releaseErr)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		for e := range errChan {
			require.NoError(t, e, "Error during concurrent get/release cycle")
		}

		finalUsedCount, err := manager.CountUsedRoomIDs(ctx)
		require.NoError(t, err)

		// Expected count: initial gets (numCycles/2) + new gets (numCycles) - releases (len(initialIDs), which is numCycles/2) = numCycles
		expectedCount := int64(numCycles)
		assert.Equal(t, expectedCount, finalUsedCount, "The final count of used IDs should be consistent")

		fmt.Printf("Strategy %v - Final used count: %d, Released queue size: %d\n",
			strategy,
			finalUsedCount,
			must(manager.CountReleasedRoomIDs(ctx)).(int64))
	}

	t.Run("Strategy: PreferReleased", func(t *testing.T) {
		runGetAndReleaseTest(t, PreferReleased)
	})

	t.Run("Strategy: PreferNew", func(t *testing.T) {
		runGetAndReleaseTest(t, PreferNew)
	})
}

// --- Other existing tests can remain largely the same, as they test mechanics not dependent on strategy ---
func TestRoomIDManager_ReleaseNonExistentID(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()
	ctx := context.Background()

	baseKey := "test_release_non_existent"
	manager := setupManager(t, rdb, baseKey, PreferReleased) // Strategy doesn't matter here
	defer manager.cleanupRedisKeys(ctx)

	nonExistentID := int64(100001)
	err := manager.ReleaseRoomID(ctx, nonExistentID)
	require.NoError(t, err, "Releasing a non-existent ID should not error")

	usedCount, _ := manager.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(0), usedCount)

	releasedCount, _ := manager.CountReleasedRoomIDs(ctx)
	assert.Equal(t, int64(1), releasedCount)

	// With PreferReleased, we should get this ID back
	id, err := manager.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	assert.Equal(t, nonExistentID, id)
}

func TestRoomIDManager_ReleasedListTrimming(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()
	ctx := context.Background()

	baseKey := "test_trimming"
	// Use custom options for this test
	manager, err := NewRoomIDManager(rdb, baseKey, WithMaxReleasedIDsToStore(3))
	require.NoError(t, err)
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	defer manager.cleanupRedisKeys(ctx)

	idsToRelease := make([]int64, 5)
	expectedInList := make([]string, 0, 3)

	for i := 0; i < 5; i++ {
		id := int64(minRoomIDValue + i)
		idsToRelease[i] = id
		// Manually add to used set to simulate they were in use
		err := rdb.SAdd(ctx, manager.usedRoomIDsSetKey, formatRoomID(id)).Err()
		require.NoError(t, err)
	}

	for _, id := range idsToRelease {
		err := manager.ReleaseRoomID(ctx, id)
		require.NoError(t, err)
	}

	// The list should contain the last 3 released IDs: 100002, 100003, 100004
	expectedInList = append(expectedInList, formatRoomID(100002), formatRoomID(100003), formatRoomID(100004))

	releasedCount, err := manager.CountReleasedRoomIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), releasedCount, "Released list should be trimmed to 3")

	actualList, _ := rdb.LRange(ctx, manager.releasedRoomIDsListKey, 0, -1).Result()
	assert.ElementsMatch(t, expectedInList, actualList, "Trimmed list should contain the last 3 released IDs")
}

func TestRoomIDManager_ScriptReload(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()
	ctx := context.Background()

	baseKey := "test_script_reload"
	manager := setupManager(t, rdb, baseKey, PreferReleased) // Strategy doesn't matter
	defer manager.cleanupRedisKeys(ctx)

	// First call should load the scripts
	id1, err := manager.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, getAndMarkUsedScriptSHA)

	// Release call ensures the other script is loaded
	err = manager.ReleaseRoomID(ctx, id1)
	require.NoError(t, err)
	require.NotEmpty(t, releaseScriptSHA)

	// Flush Redis, which removes scripts
	mr.FlushAll()

	// The manager should detect the script is gone and reload it
	id2, err := manager.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2) // Should be a new ID

	// And again for the release script
	err = manager.ReleaseRoomID(ctx, id2)
	require.NoError(t, err)
}

func TestRoomIDManager_MultipleManagers_DifferentBaseKeys(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	defer rdb.(*redis.Client).Close()
	ctx := context.Background()

	manager1 := setupManager(t, rdb, "game_A", PreferReleased)
	defer manager1.cleanupRedisKeys(ctx)

	manager2 := setupManager(t, rdb, "game_B", PreferReleased)
	defer manager2.cleanupRedisKeys(ctx)

	// Get ID from manager 1
	id_A1, err := manager1.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)

	// Check counts
	count_A, _ := manager1.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), count_A)
	count_B, _ := manager2.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(0), count_B)

	// Get ID from manager 2
	// FIX: Use blank identifier '_' because id_B1 is not used later
	_, err = manager2.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)

	// Check counts again
	count_A, _ = manager1.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), count_A)
	count_B, _ = manager2.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), count_B)

	// Release from manager 1 and reuse
	err = manager1.ReleaseRoomID(ctx, id_A1)
	require.NoError(t, err)
	id_A2, err := manager1.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	assert.Equal(t, id_A1, id_A2)

	// Ensure manager 2's state is unaffected
	count_B, _ = manager2.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), count_B)
	released_B, _ := manager2.CountReleasedRoomIDs(ctx)
	assert.Equal(t, int64(0), released_B)
}

func must(val interface{}, err error) interface{} {
	if err != nil {
		panic(fmt.Sprintf("must failed: %v", err))
	}
	return val
}

// Sorting helper for comparing slices of strings when order doesn't matter
func sortStrings(s []string) []string {
	sort.Strings(s)
	return s
}
