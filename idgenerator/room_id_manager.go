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

// GenerationStrategy 定义了ID的生成策略
type GenerationStrategy int

const (
	// PreferReleased (默认策略): 优先从释放列表中获取ID，如果列表为空，再尝试生成新的ID。
	PreferReleased GenerationStrategy = iota
	// PreferNew: 优先尝试生成新的ID，只有在生成失败（例如，多次尝试后都冲突）时，才尝试从释放列表中获取ID作为备用方案。
	PreferNew
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
	strategy               GenerationStrategy // ID生成策略
}

// Option 是用于配置 RoomIDManager 的函数类型
type Option func(*RoomIDManager)

// WithGenerationStrategy 是一个选项，用于设置ID生成策略。
func WithGenerationStrategy(strategy GenerationStrategy) Option {
	return func(m *RoomIDManager) {
		m.strategy = strategy
	}
}

// WithMaxReleasedIDsToStore 是一个选项，用于设置释放队列的最大长度。
func WithMaxReleasedIDsToStore(max int64) Option {
	return func(m *RoomIDManager) {
		if max > 0 {
			m.maxReleasedIDsToStore = max
		}
	}
}

// WithRandomGenerationAttempts 是一个选项，用于设置随机生成ID时的尝试次数。
func WithRandomGenerationAttempts(attempts int) Option {
	return func(m *RoomIDManager) {
		if attempts > 0 {
			m.randomAttempts = attempts
		}
	}
}

// NewRoomIDManager 创建一个新的 RoomIDManager 实例
// client: Redis 客户端
// baseKey: 用于区分不同类型房间号池的键
// opts: 一系列可选配置，例如 WithGenerationStrategy(PreferNew)
func NewRoomIDManager(client redis.Cmdable, baseKey string, opts ...Option) (*RoomIDManager, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	if baseKey == "" {
		return nil, fmt.Errorf("baseKey cannot be empty")
	}

	m := &RoomIDManager{
		client:                 client,
		baseKey:                baseKey,
		usedRoomIDsSetKey:      usedRoomIDsSetKeyPrefix + baseKey,
		releasedRoomIDsListKey: releasedRoomIDsListKeyPrefix + baseKey,
		// 设置默认值
		maxReleasedIDsToStore: defaultMaxReleasedIDsToStore,
		randomAttempts:        defaultRandomGenerationAttempts,
		strategy:              PreferReleased, // 默认策略是释放优先
	}

	// 应用所有传入的选项
	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

// luaGetAndMarkUsedScript:
// KEYS[1] - releasedRoomIDsListKey
// KEYS[2] - usedRoomIDsSetKey
// ARGV[1] - roomID (to mark used if randomly generated)
// ARGV[2] - mode ("try_released" or "mark_random")
const luaGetAndMarkUsedScript = `
local released_list_key = KEYS[1]
local used_set_key = KEYS[2]
local room_id_arg = ARGV[1]
local mode = ARGV[2]

if mode == "try_released" then
    local room_id = redis.call('LPOP', released_list_key)
    if room_id then
        redis.call('SADD', used_set_key, room_id)
        return room_id
    else
        return nil
    end
elseif mode == "mark_random" then
    return redis.call('SADD', used_set_key, room_id_arg)
else
    return redis.error_reply("Unknown mode: " .. mode)
end
`

// luaReleaseScript:
// KEYS[1] - usedRoomIDsSetKey
// KEYS[2] - releasedRoomIDsListKey
// ARGV[1] - roomID
// ARGV[2] - maxReleasedIDs (max size for the released list)
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

redis.call('RPUSH', released_list_key, room_id)
redis.call('LTRIM', released_list_key, -max_released, -1)

return removed
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

// ensureScriptsLoaded 确保Lua脚本已经被加载到Redis
func (m *RoomIDManager) ensureScriptsLoaded(ctx context.Context) error {
	if getAndMarkUsedScriptSHA == "" || releaseScriptSHA == "" {
		return m.LoadScripts(ctx)
	}
	return nil
}

// GetNextAvailableRoomID 根据配置的策略获取下一个可用的房间号。
func (m *RoomIDManager) GetNextAvailableRoomID(ctx context.Context) (int64, error) {
	if err := m.ensureScriptsLoaded(ctx); err != nil {
		return 0, fmt.Errorf("failed to ensure Lua scripts are loaded: %w", err)
	}

	switch m.strategy {
	case PreferNew:
		// 策略: 新增优先
		id, err := m.tryGetNewID(ctx)
		if err == nil {
			return id, nil // 成功生成新ID，直接返回
		}

		// 如果生成新ID失败, 则尝试从释放列表中获取作为备用方案。
		fmt.Printf("warning: could not generate a new ID (err: %v), falling back to released list for baseKey: %s\n", err, m.baseKey)
		id, found, err := m.tryGetReleasedID(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get new ID and also failed to get released ID: %w", err)
		}
		if !found {
			return 0, fmt.Errorf("failed to get new ID and the released list is also empty for baseKey: %s", m.baseKey)
		}
		return id, nil

	case PreferReleased:
		fallthrough // fallthrough到default，因为这是默认行为
	default:
		// 策略: 释放优先 (默认)
		id, found, err := m.tryGetReleasedID(ctx)
		if err != nil {
			return 0, fmt.Errorf("error trying to get ID from released list: %w", err)
		}
		if found {
			return id, nil // 成功从释放列表获取，直接返回
		}

		// 释放列表为空，尝试生成新的ID
		return m.tryGetNewID(ctx)
	}
}

// tryGetReleasedID 尝试从释放列表中获取一个ID。
// 返回: (获取到的ID, 是否找到, 错误)
func (m *RoomIDManager) tryGetReleasedID(ctx context.Context) (int64, bool, error) {
	var roomIDStr string

	// 通过Lua脚本原子地从list中LPOP并SADD到set中
	roomIDVal, err := m.client.EvalSha(ctx, getAndMarkUsedScriptSHA,
		[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey},
		"", "try_released").Result()

	if err != nil {
		// 优雅地处理脚本未加载的情况
		if strings.Contains(err.Error(), "NOSCRIPT") {
			if loadErr := m.LoadScripts(ctx); loadErr != nil {
				return 0, false, fmt.Errorf("failed to reload Lua script after NOSCRIPT: %w", loadErr)
			}
			roomIDVal, err = m.client.Eval(ctx, luaGetAndMarkUsedScript,
				[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey},
				"", "try_released").Result()
		}
	}

	// 如果除了 redis.Nil 之外还有其他错误
	if err != nil && err != redis.Nil {
		return 0, false, fmt.Errorf("error getting ID from released list via Lua: %w", err)
	}

	// 检查返回值
	if idStr, ok := roomIDVal.(string); ok && idStr != "" {
		roomIDStr = idStr
	} else {
		return 0, false, nil // 列表为空或返回了非预期的类型，视为未找到
	}

	parsedID, parseErr := strconv.ParseInt(roomIDStr, 10, 64)
	if parseErr != nil {
		return 0, false, fmt.Errorf("failed to parse room ID '%s' from released list: %w", roomIDStr, parseErr)
	}

	return parsedID, true, nil
}

// tryGetNewID 尝试随机生成一个新的唯一ID。
func (m *RoomIDManager) tryGetNewID(ctx context.Context) (int64, error) {
	for i := 0; i < m.randomAttempts; i++ {
		rangeSize := big.NewInt(int64(maxRoomIDValue - minRoomIDValue + 1))
		n, randErr := rand.Int(rand.Reader, rangeSize)
		if randErr != nil {
			return 0, fmt.Errorf("failed to generate random number: %w", randErr)
		}
		randomNumber := n.Int64() + minRoomIDValue
		candidateIDStr := formatRoomID(randomNumber)

		// 通过Lua脚本原子地检查并添加到 used set
		added, saddErr := m.client.EvalSha(ctx, getAndMarkUsedScriptSHA,
			[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey},
			candidateIDStr, "mark_random").Result()

		if saddErr != nil {
			if strings.Contains(saddErr.Error(), "NOSCRIPT") {
				if loadErr := m.LoadScripts(ctx); loadErr != nil {
					return 0, fmt.Errorf("failed to reload Lua script for random mark: %w", loadErr)
				}
				added, saddErr = m.client.Eval(ctx, luaGetAndMarkUsedScript,
					[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey},
					candidateIDStr, "mark_random").Result()
			}
		}

		if saddErr != nil {
			fmt.Printf("Error marking random ID %s as used: %v. Attempt %d/%d\n", candidateIDStr, saddErr, i+1, m.randomAttempts)
			time.Sleep(10 * time.Millisecond) // 稍作等待再重试
			continue
		}

		if addedInt, ok := added.(int64); ok && addedInt == 1 {
			return randomNumber, nil // 成功添加，说明是唯一的，返回
		}
	}

	return 0, fmt.Errorf("failed to obtain a unique room ID after %d random attempts (baseKey: %s)", m.randomAttempts, m.baseKey)
}

// ReleaseRoomID 将一个房间号标记为未使用，并将其添加到待重用列表
func (m *RoomIDManager) ReleaseRoomID(ctx context.Context, roomID int64) error {
	if roomID < minRoomIDValue || roomID > maxRoomIDValue {
		return fmt.Errorf("roomID %d is out of valid range [%d, %d]", roomID, minRoomIDValue, maxRoomIDValue)
	}
	roomIDStr := formatRoomID(roomID)

	if err := m.ensureScriptsLoaded(ctx); err != nil {
		return fmt.Errorf("failed to ensure Lua scripts are loaded before releasing ID: %w", err)
	}

	maxReleasedStr := strconv.FormatInt(m.maxReleasedIDsToStore, 10)

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

// cleanupRedisKeys (for testing or specific scenarios)
// BE CAREFUL using this, it deletes the keys from Redis.
func (m *RoomIDManager) cleanupRedisKeys(ctx context.Context) error {
	_, err := m.client.Del(ctx, m.usedRoomIDsSetKey, m.releasedRoomIDsListKey).Result()
	if err != nil {
		return fmt.Errorf("failed to delete Redis keys for baseKey %s: %w", m.baseKey, err)
	}
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
