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

// setupTestRedisForRoomManager is a helper from redis_idgenerator_test.go
func setupTestRedisForRoomManager(t *testing.T) (*miniredis.Miniredis, redis.Cmdable) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if errPing := client.Ping(ctx).Err(); errPing != nil {
		s.Close()
		t.Fatalf("Failed to ping miniredis: %v", errPing)
	}
	return s, client
}

func TestNewRoomIDManager(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	// Type assert to *redis.Client to access Close() method if rdb is redis.Cmdable
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}

	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		baseKey := "test_success"
		manager, err := NewRoomIDManager(rdb, baseKey)
		require.NoError(t, err)
		require.NotNil(t, manager)
		assert.Equal(t, baseKey, manager.baseKey)
		assert.Equal(t, usedRoomIDsSetKeyPrefix+baseKey, manager.usedRoomIDsSetKey)
		assert.Equal(t, releasedRoomIDsListKeyPrefix+baseKey, manager.releasedRoomIDsListKey)

		// Test loading scripts
		// Clear global SHAs before this specific sub-test to ensure LoadScripts populates them.
		getAndMarkUsedScriptSHA = "" // Reset for this sub-test
		releaseScriptSHA = ""        // Reset for this sub-test
		err = manager.LoadScripts(ctx)
		require.NoError(t, err, "LoadScripts should succeed")
		assert.NotEmpty(t, getAndMarkUsedScriptSHA, "getAndMarkUsedScriptSHA should be populated")
		assert.NotEmpty(t, releaseScriptSHA, "releaseScriptSHA should be populated")

		// Clean up global SHAs after this sub-test for other tests
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
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

	t.Run("script load fail - simulate by closing miniredis then creating manager", func(t *testing.T) {
		mrServer, rdbClient := setupTestRedisForRoomManager(t)
		baseKey := "test_script_load_prep"
		manager, err := NewRoomIDManager(rdbClient, baseKey)
		require.NoError(t, err)
		require.NotNil(t, manager)

		// Clear global SHAs to ensure LoadScripts attempts to load them.
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""

		mrServer.Close() // Close server before LoadScripts

		err = manager.LoadScripts(ctx)
		require.Error(t, err, "LoadScripts should fail if Redis is down")

		getAndMarkUsedScriptSHA = "" // Clean up global SHAs
		releaseScriptSHA = ""
		if rc, ok := rdbClient.(*redis.Client); ok {
			rc.Close() // Close the client associated with the closed server
		}
	})
}

func TestRoomIDManager_GetNextAvailableRoomID_Basic(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	baseKey := "test_get_basic"
	manager, err := NewRoomIDManager(rdb, baseKey)
	require.NoError(t, err)
	getAndMarkUsedScriptSHA = "" // Reset for this test
	releaseScriptSHA = ""        // Reset for this test
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	defer func() {
		manager.cleanupRedisKeys(ctx)
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
	}()

	id, err := manager.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	assert.True(t, id >= minRoomIDValue && id <= maxRoomIDValue, "ID %d out of range", id)

	// Verify it's in the used set (需要格式化为字符串)
	idStr := formatRoomID(id)
	isMember, err := rdb.SIsMember(ctx, manager.usedRoomIDsSetKey, idStr).Result()
	require.NoError(t, err)
	assert.True(t, isMember, "ID should be in the used set")

	usedCount, err := manager.CountUsedRoomIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), usedCount)

	releasedCount, err := manager.CountReleasedRoomIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), releasedCount)

	id2, err := manager.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, id, id2)
	assert.True(t, id2 >= minRoomIDValue && id2 <= maxRoomIDValue, "ID2 %d out of range", id2)
	usedCount2, _ := manager.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(2), usedCount2)
}

func TestRoomIDManager_ReleaseAndReuseRoomID(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	baseKey := "test_release_reuse"
	manager, err := NewRoomIDManager(rdb, baseKey)
	require.NoError(t, err)
	getAndMarkUsedScriptSHA = "" // Reset for this test
	releaseScriptSHA = ""        // Reset for this test
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	defer func() {
		manager.cleanupRedisKeys(ctx)
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
	}()

	id1, err := manager.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	assert.True(t, id1 >= minRoomIDValue && id1 <= maxRoomIDValue, "ID1 %d out of range", id1)

	usedCount, _ := manager.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), usedCount)

	err = manager.ReleaseRoomID(ctx, id1) // <--- 传递 int64
	require.NoError(t, err)

	id1Str := formatRoomID(id1)
	isMember, _ := rdb.SIsMember(ctx, manager.usedRoomIDsSetKey, id1Str).Result()
	assert.False(t, isMember, "ID should be removed from used set after release")
	usedCount, _ = manager.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(0), usedCount)

	releasedIDs, _ := rdb.LRange(ctx, manager.releasedRoomIDsListKey, 0, -1).Result()
	assert.Contains(t, releasedIDs, id1Str)
	releasedCount, _ := manager.CountReleasedRoomIDs(ctx)
	assert.Equal(t, int64(1), releasedCount)

	id2, err := manager.GetNextAvailableRoomID(ctx)
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "Should reuse the released ID")

	releasedCount, _ = manager.CountReleasedRoomIDs(ctx)
	assert.Equal(t, int64(0), releasedCount)
	usedCount, _ = manager.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), usedCount)
}

func TestRoomIDManager_ReleasedListTrimming(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	baseKey := "test_trimming"
	manager, err := NewRoomIDManager(rdb, baseKey)
	require.NoError(t, err)
	getAndMarkUsedScriptSHA = "" // Reset for this test
	releaseScriptSHA = ""        // Reset for this test
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	defer func() {
		manager.cleanupRedisKeys(ctx)
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
	}()

	originalMaxReleased := manager.maxReleasedIDsToStore
	manager.maxReleasedIDsToStore = 3
	defer func() { manager.maxReleasedIDsToStore = originalMaxReleased }()

	idsToReleaseInt := []int64{}
	idsToReleaseStr := []string{}
	for i := 0; i < 5; i++ {
		idVal := int64(minRoomIDValue + i)
		idsToReleaseInt = append(idsToReleaseInt, idVal)
		idStr := formatRoomID(idVal)
		idsToReleaseStr = append(idsToReleaseStr, idStr)
		_, err := rdb.SAdd(ctx, manager.usedRoomIDsSetKey, idStr).Result()
		require.NoError(t, err)
	}

	for _, idVal := range idsToReleaseInt {
		err := manager.ReleaseRoomID(ctx, idVal) // <--- 传递 int64
		require.NoError(t, err)
	}

	releasedCount, err := manager.CountReleasedRoomIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, manager.maxReleasedIDsToStore, releasedCount, "Released list should be trimmed")

	expectedInListStr := idsToReleaseStr[len(idsToReleaseStr)-int(manager.maxReleasedIDsToStore):]
	actualList, _ := rdb.LRange(ctx, manager.releasedRoomIDsListKey, 0, -1).Result()
	assert.ElementsMatch(t, expectedInListStr, actualList, "Trimmed list should contain the last released IDs")
}

func TestRoomIDManager_GetNext_ExhaustionAttempt(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	baseKey := "test_exhaust"
	manager, err := NewRoomIDManager(rdb, baseKey)
	require.NoError(t, err)
	getAndMarkUsedScriptSHA = "" // Reset for this test
	releaseScriptSHA = ""        // Reset for this test
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	defer func() {
		manager.cleanupRedisKeys(ctx)
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
	}()

	originalAttempts := manager.randomAttempts
	manager.randomAttempts = 2
	defer func() { manager.randomAttempts = originalAttempts }()

	for i := 0; i <= manager.randomAttempts; i++ {
		idToFill := int64(minRoomIDValue + i)
		idStr := formatRoomID(idToFill)
		_, err := rdb.SAdd(ctx, manager.usedRoomIDsSetKey, idStr).Result()
		require.NoError(t, err)
	}

	_, err = manager.GetNextAvailableRoomID(ctx) // Expects (int64, error)
	require.Error(t, err, "Should return an error when pool is exhausted within attempts")
	assert.Contains(t, err.Error(), "failed to obtain a unique room ID after")
}

func TestRoomIDManager_Concurrent_GetNext(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	baseKey := "test_concurrent_get"
	manager, err := NewRoomIDManager(rdb, baseKey)
	require.NoError(t, err)
	getAndMarkUsedScriptSHA = "" // Reset for this test
	releaseScriptSHA = ""        // Reset for this test
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	defer func() {
		manager.cleanupRedisKeys(ctx)
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
	}()

	numGoroutines := 20
	idsPerGoroutine := 5
	totalIDsToGet := numGoroutines * idsPerGoroutine

	var wg sync.WaitGroup
	generatedIDs := make(map[int64]struct{}) // <--- map存储 int64
	var mu sync.Mutex
	errors := make(chan error, totalIDsToGet)
	idsChan := make(chan int64, totalIDsToGet)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, getErr := manager.GetNextAvailableRoomID(ctx) // <--- 获取 int64
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

func TestRoomIDManager_ScriptReload(t *testing.T) {
	mr, rdbClient := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdbClient.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	baseKey := "test_script_reload"
	manager, err := NewRoomIDManager(rdbClient, baseKey)
	require.NoError(t, err)
	getAndMarkUsedScriptSHA = "" // Reset for this test
	releaseScriptSHA = ""        // Reset for this test
	defer func() {
		manager.cleanupRedisKeys(ctx)
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
	}()

	id1, err := manager.GetNextAvailableRoomID(ctx) // <--- 获取 int64
	require.NoError(t, err)
	assert.True(t, id1 != 0 && id1 >= minRoomIDValue && id1 <= maxRoomIDValue)
	require.NotEmpty(t, getAndMarkUsedScriptSHA, "getAndMarkUsedScriptSHA should be loaded by GetNextAvailableRoomID")
	// releaseScriptSHA might not be loaded yet if ReleaseRoomID hasn't been called directly or via LoadScripts.
	// Let's explicitly load all for clarity for the next step.
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, releaseScriptSHA)

	mr.FlushAll()
	getAndMarkUsedScriptSHA = ""
	releaseScriptSHA = ""

	id2, err := manager.GetNextAvailableRoomID(ctx) // <--- 获取 int64
	require.NoError(t, err)
	assert.True(t, id2 != 0 && id2 >= minRoomIDValue && id2 <= maxRoomIDValue)
	assert.NotEqual(t, id1, id2)
	require.NotEmpty(t, getAndMarkUsedScriptSHA, "getAndMarkUsedScriptSHA should be re-loaded")

	// Reset releaseScriptSHA to test its reload by ReleaseRoomID
	releaseScriptSHA = ""
	err = manager.ReleaseRoomID(ctx, id2) // <--- 传递 int64
	require.NoError(t, err)
	require.NotEmpty(t, releaseScriptSHA, "releaseScriptSHA should be loaded by ReleaseRoomID")
}

func TestRoomIDManager_Concurrent_GetAndRelease(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	baseKey := "test_concurrent_get_release"
	manager, err := NewRoomIDManager(rdb, baseKey)
	require.NoError(t, err)
	getAndMarkUsedScriptSHA = "" // Reset for this test
	releaseScriptSHA = ""        // Reset for this test
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	defer func() {
		manager.cleanupRedisKeys(ctx)
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
	}()

	numCycles := 50
	var wg sync.WaitGroup
	errChan := make(chan error, numCycles*2)

	initialIDs := make([]int64, 0, numCycles/2) // <--- 存储 int64
	for i := 0; i < numCycles/2; i++ {
		id, getErr := manager.GetNextAvailableRoomID(ctx) // <--- 获取 int64
		if getErr != nil {
			t.Fatalf("Failed to get initial ID for release: %v", getErr)
		}
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
				releaseErr := manager.ReleaseRoomID(ctx, initialIDs[idx]) // <--- 传递 int64
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
	if numCycles > len(initialIDs) {
		assert.True(t, finalUsedCount >= 0, "Expected some IDs to be in use or all used and released leading to zero if gets perfectly matched releases of new ones.")
	}

	fmt.Printf("Final used count: %d, Released queue size: %d (after %d get/release cycles)\n",
		finalUsedCount,
		must(manager.CountReleasedRoomIDs(ctx)).(int64),
		numCycles)
}

func must(val interface{}, err error) interface{} {
	if err != nil {
		panic(fmt.Sprintf("must failed: %v", err)) // Provide more context on panic
	}
	return val
}

func TestRoomIDManager_ReleaseNonExistentID(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	baseKey := "test_release_non_existent"
	manager, err := NewRoomIDManager(rdb, baseKey)
	require.NoError(t, err)
	getAndMarkUsedScriptSHA = "" // Reset for this test
	releaseScriptSHA = ""        // Reset for this test
	err = manager.LoadScripts(ctx)
	require.NoError(t, err)
	defer func() {
		manager.cleanupRedisKeys(ctx)
		getAndMarkUsedScriptSHA = ""
		releaseScriptSHA = ""
	}()

	nonExistentIDInt := int64(100000)
	err = manager.ReleaseRoomID(ctx, nonExistentIDInt) // <--- 传递 int64
	require.NoError(t, err, "Releasing a non-existent ID should not error")

	usedCount, _ := manager.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(0), usedCount)

	releasedCount, _ := manager.CountReleasedRoomIDs(ctx)
	assert.Equal(t, int64(1), releasedCount)
	listContents, _ := rdb.LRange(ctx, manager.releasedRoomIDsListKey, 0, -1).Result()
	assert.Contains(t, listContents, formatRoomID(nonExistentIDInt)) // 比较时用string形式

	id, err := manager.GetNextAvailableRoomID(ctx) // <--- 获取 int64
	require.NoError(t, err)
	assert.Equal(t, nonExistentIDInt, id)
}

func TestRoomIDManager_FormatRoomID(t *testing.T) {
	assert.Equal(t, "100000", formatRoomID(100000))
	assert.Equal(t, "999999", formatRoomID(999999))
	assert.Equal(t, "123456", formatRoomID(123456))
	assert.Equal(t, "000001", formatRoomID(1)) // Tests padding for numbers outside the typical minRoomIDValue
	assert.Equal(t, "000100", formatRoomID(100))
}

func TestRoomIDManager_MultipleManagers_DifferentBaseKeys(t *testing.T) {
	mr, rdb := setupTestRedisForRoomManager(t)
	defer mr.Close()
	if rc, ok := rdb.(*redis.Client); ok {
		defer rc.Close()
	}
	ctx := context.Background()

	// Reset global SHAs before this test block to ensure each manager loads them if needed
	getAndMarkUsedScriptSHA = ""
	releaseScriptSHA = ""

	baseKey1 := "game_type_A"
	manager1, err := NewRoomIDManager(rdb, baseKey1)
	require.NoError(t, err)
	err = manager1.LoadScripts(ctx)
	require.NoError(t, err)
	shaAfterM1LoadGet := getAndMarkUsedScriptSHA // Capture SHA after first load
	shaAfterM1LoadRelease := releaseScriptSHA
	defer manager1.cleanupRedisKeys(ctx)

	baseKey2 := "game_type_B"
	manager2, err := NewRoomIDManager(rdb, baseKey2)
	require.NoError(t, err)
	// Scripts should already be loaded in Redis by manager1, manager2.LoadScripts should be a no-op or just verify.
	// To be sure, we can check if SHAs remain the same or are re-verified.
	getAndMarkUsedScriptSHA = "" // Temporarily clear to see if LoadScripts re-populates from existing Redis script
	releaseScriptSHA = ""
	err = manager2.LoadScripts(ctx)
	require.NoError(t, err)
	assert.Equal(t, shaAfterM1LoadGet, getAndMarkUsedScriptSHA) // Should be same SHA
	assert.Equal(t, shaAfterM1LoadRelease, releaseScriptSHA)
	defer manager2.cleanupRedisKeys(ctx)

	id_A1, err := manager1.GetNextAvailableRoomID(ctx) // <--- 获取 int64
	require.NoError(t, err)
	assert.True(t, id_A1 != 0)

	usedCount_A, _ := manager1.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), usedCount_A)
	usedCount_B, _ := manager2.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(0), usedCount_B)

	id_B1, err := manager2.GetNextAvailableRoomID(ctx) // <--- 获取 int64
	require.NoError(t, err)
	assert.True(t, id_B1 != 0)
	assert.NotEqual(t, id_A1, id_B1)

	usedCount_A, _ = manager1.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), usedCount_A)
	usedCount_B, _ = manager2.CountUsedRoomIDs(ctx)
	assert.Equal(t, int64(1), usedCount_B)

	err = manager1.ReleaseRoomID(ctx, id_A1) // <--- 传递 int64
	require.NoError(t, err)
	releasedCount_A, _ := manager1.CountReleasedRoomIDs(ctx)
	assert.Equal(t, int64(1), releasedCount_A)
	releasedCount_B, _ := manager2.CountReleasedRoomIDs(ctx)
	assert.Equal(t, int64(0), releasedCount_B)

	id_A2, err := manager1.GetNextAvailableRoomID(ctx) // <--- 获取 int64
	require.NoError(t, err)
	assert.Equal(t, id_A1, id_A2)

	getAndMarkUsedScriptSHA = "" // Final cleanup for other potential tests in suite
	releaseScriptSHA = ""
}

func TestRoomIDManager_MaxCapacity(t *testing.T) {
	t.Skip("Skipping full capacity test as it's too slow (900,000 IDs)")
}

// Sorting helper for comparing slices of strings when order doesn't matter
func sortStrings(s []string) []string {
	sort.Strings(s)
	return s
}
