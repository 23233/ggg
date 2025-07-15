package idgenerator

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	minRoomIDValue = 100000 // 最小房间号 (100000)
	maxRoomIDValue = 999999 // 最大房间号 (999999)
	roomIDLength   = 6      // 房间号长度

	usedRoomIDsSetKeyPrefix      = "room_ids:used:"     // Redis set 存储已使用的房间号 (后接特定业务key, 如游戏类型)
	releasedRoomIDsListKeyPrefix = "room_ids:released:" // Redis list 存储最近释放的房间号 (后接特定业务key)

	defaultMaxReleasedIDsToStore    = 1000 // 释放队列中保留的最大ID数量
	defaultRandomGenerationAttempts = 20   // 随机生成ID时的尝试次数
)

// RoomIDManager 负责管理房间ID的分配和释放
type RoomIDManager struct {
	client                 redis.Cmdable
	baseKey                string // 用于构造 Redis key 的基础部分, 例如 "game_type_xyz"
	usedRoomIDsSetKey      string
	releasedRoomIDsListKey string
	maxReleasedIDsToStore  int64
	randomAttempts         int
}

// NewRoomIDManager 创建一个新的 RoomIDManager 实例
// client: Redis 客户端 (可以是 *redis.Client, *redis.ClusterClient, etc.)
// baseKey: 用于区分不同类型房间号池的键，例如 specific_game_type
func NewRoomIDManager(client redis.Cmdable, baseKey string) (*RoomIDManager, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	if baseKey == "" {
		return nil, fmt.Errorf("baseKey cannot be empty")
	}

	return &RoomIDManager{
		client:                 client,
		baseKey:                baseKey,
		usedRoomIDsSetKey:      usedRoomIDsSetKeyPrefix + baseKey,
		releasedRoomIDsListKey: releasedRoomIDsListKeyPrefix + baseKey,
		maxReleasedIDsToStore:  defaultMaxReleasedIDsToStore,
		randomAttempts:         defaultRandomGenerationAttempts,
	}, nil
}

// luaGetAndMarkUsedScript:
// KEYS[1] - releasedRoomIDsListKey
// KEYS[2] - usedRoomIDsSetKey
// ARGV[1] - roomID (to mark used if randomly generated)
// ARGV[2] - mode ("try_released" or "mark_random")
//
// "try_released" mode:
//
//	Tries to LPOP from KEYS[1].
//	If an ID is popped, it SADDs it to KEYS[2]. If SADD fails (already there, unexpected), it returns an error.
//	Returns the popped ID or nil if the list was empty.
//
// "mark_random" mode:
//
//	SADD ARGV[1] to KEYS[2].
//	Returns 1 if ARGV[1] was added (was not in set), 0 if ARGV[1] was already in set.
const luaGetAndMarkUsedScript = `
local released_list_key = KEYS[1]
local used_set_key = KEYS[2]
local room_id_arg = ARGV[1]
local mode = ARGV[2]

if mode == "try_released" then
    local room_id = redis.call('LPOP', released_list_key)
    if room_id then
        -- Try to add to used set. If it's already there, it's a bit strange but we proceed.
        -- If SADD returns 0, it means it was already in the set. This is unexpected if it came from released list.
        -- However, for robustness, we accept it. The main goal is it's now "officially" used.
        redis.call('SADD', used_set_key, room_id)
        return room_id
    else
        return nil -- Or an empty string, depending on how client handles nil
    end
elseif mode == "mark_random" then
    return redis.call('SADD', used_set_key, room_id_arg)
else
    return redis.error_reply("Unknown mode: " .. mode)
end
`

var getAndMarkUsedScriptSHA string
var releaseScriptSHA string

// LoadScripts loads Lua scripts into Redis.
func (m *RoomIDManager) LoadScripts(ctx context.Context) error {
	shaGet, err := m.client.ScriptLoad(ctx, luaGetAndMarkUsedScript).Result()
	if err != nil {
		return fmt.Errorf("failed to load get_and_mark_used script: %w", err)
	}
	getAndMarkUsedScriptSHA = shaGet

	shaRelease, err := m.client.ScriptLoad(ctx, luaReleaseScript).Result()
	if err != nil {
		return fmt.Errorf("failed to load release script: %w", err)
	}
	releaseScriptSHA = shaRelease
	return nil
}

func (m *RoomIDManager) ensureScriptsLoaded(ctx context.Context) error {
	// Check both SHAs. If either is missing, attempt to load (or reload) both.
	if getAndMarkUsedScriptSHA == "" || releaseScriptSHA == "" {
		// Attempt to load if not already loaded (e.g. first call or Redis flush)
		// In a multi-instance setup, each instance would try to load; this is fine as ScriptLoad is idempotent.
		return m.LoadScripts(ctx)
	}
	return nil
}

// GetNextAvailableRoomID 获取下一个可用的6位数字房间号
// 它首先尝试从最近释放的列表中获取，如果列表为空，则尝试随机生成。
func (m *RoomIDManager) GetNextAvailableRoomID(ctx context.Context) (int64, error) {
	if err := m.ensureScriptsLoaded(ctx); err != nil {
		return 0, fmt.Errorf("failed to ensure Lua scripts are loaded: %w", err)
	}

	var roomIDStr string // 用于存储获取到的字符串形式的ID

	// 1. Try to get from released list and mark as used (atomically via Lua)
	var roomIDVal interface{}
	var err error

	roomIDVal, err = m.client.EvalSha(ctx, getAndMarkUsedScriptSHA,
		[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey},
		"", "try_released").Result()

	if err != nil {
		if strings.Contains(err.Error(), "NOSCRIPT") {
			if loadErr := m.LoadScripts(ctx); loadErr != nil {
				return 0, fmt.Errorf("failed to reload Lua script after NOSCRIPT: %w", loadErr)
			}
			roomIDVal, err = m.client.Eval(ctx, luaGetAndMarkUsedScript,
				[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey},
				"", "try_released").Result()
		}
	}

	if err == nil {
		if idStr, ok := roomIDVal.(string); ok && idStr != "" {
			roomIDStr = idStr // 成功从释放列表获取
		}
		// If roomIDVal is nil or empty, proceed to random generation.
	} else if err != redis.Nil {
		return 0, fmt.Errorf("error trying to get ID from released list via Lua: %w", err)
	}

	// 如果从释放列表成功获取到ID
	if roomIDStr != "" {
		parsedID, parseErr := strconv.ParseInt(roomIDStr, 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("failed to parse room ID string '%s' from released list to int64: %w", roomIDStr, parseErr)
		}
		return parsedID, nil
	}

	// 2. If released list was empty or did not yield a valid ID, try random generation
	for i := 0; i < m.randomAttempts; i++ {
		rangeSize := big.NewInt(int64(maxRoomIDValue - minRoomIDValue + 1))
		n, randErr := rand.Int(rand.Reader, rangeSize)
		if randErr != nil {
			return 0, fmt.Errorf("failed to generate random number: %w", randErr)
		}
		randomNumberInCorrectRange := n.Int64() + minRoomIDValue
		candidateIDStr := formatRoomID(randomNumberInCorrectRange) // String form for Redis

		added, saddErr := m.client.EvalSha(ctx, getAndMarkUsedScriptSHA,
			[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey},
			candidateIDStr, "mark_random").Result()

		if saddErr != nil {
			if strings.Contains(saddErr.Error(), "NOSCRIPT") {
				if loadErr := m.LoadScripts(ctx); loadErr != nil {
					return 0, fmt.Errorf("failed to reload Lua script for random mark after NOSCRIPT: %w", loadErr)
				}
				added, saddErr = m.client.Eval(ctx, luaGetAndMarkUsedScript,
					[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey},
					candidateIDStr, "mark_random").Result()
			}
		}

		if saddErr != nil {
			fmt.Printf("Error marking random ID %s as used via Lua: %v. Attempt %d/%d\n", candidateIDStr, saddErr, i+1, m.randomAttempts)
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if addedInt, ok := added.(int64); ok && addedInt == 1 {
			return randomNumberInCorrectRange, nil // Successfully reserved a random ID, return the int64 form
		}
	}

	return 0, fmt.Errorf("failed to obtain a unique room ID after %d random attempts (baseKey: %s)", m.randomAttempts, m.baseKey)
}

// luaReleaseScript:
// KEYS[1] - usedRoomIDsSetKey
// KEYS[2] - releasedRoomIDsListKey
// ARGV[1] - roomID
// ARGV[2] - maxReleasedIDs (max size for the released list)
//
// Atomically removes roomID from used set and adds it to released list (RPUSH).
// Trims the released list to maxReleasedIDs.
// Returns 1 if successfully removed from used set, 0 otherwise (e.g., was not in set).
const luaReleaseScript = `
local used_set_key = KEYS[1]
local released_list_key = KEYS[2]
local room_id = ARGV[1]
local max_released_str = ARGV[2]
local max_released = tonumber(max_released_str)

if not room_id then
    return redis.error_reply("room_id (ARGV[1]) cannot be nil or empty")
end
if max_released == nil then
    return redis.error_reply("max_released (ARGV[2]) is not a number: " .. max_released_str)
end

local removed = redis.call('SREM', used_set_key, room_id)

-- Only add to released list if it was actually in use, or always add?
-- Current logic: always try to add to released list and trim.
-- This means even if SREM returns 0 (wasn't used), it might still be added to released.
-- For room IDs, if it's being released, it should have been in use.
-- If SREM returns 0, it might indicate a logic error elsewhere or an attempt to release a non-existent/already released ID.
-- However, to be safe and ensure it's available for reuse, we RPUSH it.

redis.call('RPUSH', released_list_key, room_id)
redis.call('LTRIM', released_list_key, -max_released, -1) -- Keep only the last N elements

return removed -- Return 1 if removed from set, 0 otherwise
`

// ReleaseRoomID 将一个房间号标记为未使用，并将其添加到待重用列表
func (m *RoomIDManager) ReleaseRoomID(ctx context.Context, roomID int64) error {
	if roomID < minRoomIDValue || roomID > maxRoomIDValue { // 基本的范围检查
		return fmt.Errorf("roomID %d is out of valid range [%d, %d]", roomID, minRoomIDValue, maxRoomIDValue)
	}
	roomIDStr := formatRoomID(roomID) // 将 int64 转换为字符串形式

	if err := m.ensureScriptsLoaded(ctx); err != nil {
		return fmt.Errorf("failed to ensure Lua scripts are loaded before releasing ID: %w", err)
	}

	maxReleasedStr := strconv.FormatInt(m.maxReleasedIDsToStore, 10)

	// Lua脚本期望string类型的roomID
	_, err := m.client.EvalSha(ctx, releaseScriptSHA,
		[]string{m.usedRoomIDsSetKey, m.releasedRoomIDsListKey},
		roomIDStr, maxReleasedStr).Result()

	if err != nil {
		if strings.Contains(err.Error(), "NOSCRIPT") {
			newSHA, loadErr := m.client.ScriptLoad(ctx, luaReleaseScript).Result()
			if loadErr != nil {
				return fmt.Errorf("failed to reload release script after NOSCRIPT: %w", loadErr)
			}
			releaseScriptSHA = newSHA
			_, err = m.client.Eval(ctx, luaReleaseScript,
				[]string{m.usedRoomIDsSetKey, m.releasedRoomIDsListKey},
				roomIDStr, maxReleasedStr).Result()
		}
	}

	if err != nil {
		return fmt.Errorf("failed to release room ID %s (original int: %d) using Lua script (baseKey: %s): %w", roomIDStr, roomID, m.baseKey, err)
	}

	return nil
}

// formatRoomID 将整数ID格式化为固定长度的字符串，前面补零
func formatRoomID(id int64) string {
	return fmt.Sprintf("%0*d", roomIDLength, id)
}

// Helper (optional) to validate and format a room ID string if needed
// func formatRoomIDString(idStr string) string {
// 	num, err := strconv.ParseInt(idStr, 10, 64)
// 	if err != nil {
// 		// Handle error or return original if not parsable, depending on strictness
// 		return idStr // Or some default/error indicator
// 	}
// 	return formatRoomID(num)
// }

// Cleanup (for testing or specific scenarios, not typically part of public API)
// BE CAREFUL using this, it deletes the keys from Redis.
func (m *RoomIDManager) cleanupRedisKeys(ctx context.Context) error {
	_, err := m.client.Del(ctx, m.usedRoomIDsSetKey, m.releasedRoomIDsListKey).Result()
	if err != nil {
		return fmt.Errorf("failed to delete Redis keys for baseKey %s: %w", m.baseKey, err)
	}
	// also clear script SHAs if you want a full reset for testing LoadScripts again
	// getAndMarkUsedScriptSHA = ""
	// releaseScriptSHA = ""
	return nil
}

// CountUsedRoomIDs returns the current number of used room IDs.
// Useful for monitoring.
func (m *RoomIDManager) CountUsedRoomIDs(ctx context.Context) (int64, error) {
	count, err := m.client.SCard(ctx, m.usedRoomIDsSetKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get count of used room IDs for baseKey %s: %w", m.baseKey, err)
	}
	return count, nil
}

// CountReleasedRoomIDs returns the current number of IDs in the released queue.
// Useful for monitoring.
func (m *RoomIDManager) CountReleasedRoomIDs(ctx context.Context) (int64, error) {
	count, err := m.client.LLen(ctx, m.releasedRoomIDsListKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get count of released room IDs for baseKey %s: %w", m.baseKey, err)
	}
	return count, nil
}
