package idgenerator

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

// 定义常量
const (
	// Redis Key 前缀
	seqUsedRoomIDsSetKeyPrefix      = "seq_room_ids:used:"     // Redis set 存储已使用的房间号
	seqReleasedRoomIDsListKeyPrefix = "seq_room_ids:released:" // Redis list 存储最近释放的房间号
	seqCounterKeyPrefix             = "seq_room_ids:counter:"  // Redis string 存储ID计数器
)

var (
	// ErrIDPoolExhausted 表示ID池已经耗尽 (计数器超过了最大值)
	ErrIDPoolExhausted = errors.New("room ID pool is exhausted")
)

// luaGetSequentialIDScript:
// 原子地获取一个ID。优先从释放列表获取，否则通过计数器获取。
//
// KEYS[1] - releasedRoomIDsListKey
// KEYS[2] - usedRoomIDsSetKey
// KEYS[3] - counterKey
// ARGV[1] - minIDValue
// ARGV[2] - maxIDValue
//
// 返回:
// - 成功获取的ID (string)
// - 如果计数器耗尽，返回 "EXHAUSTED"
// - 如果释放列表为空且计数器未初始化，脚本会初始化并返回第一个ID
const luaGetSequentialIDScript = `
local released_list_key = KEYS[1]
local used_set_key = KEYS[2]
local counter_key = KEYS[3]
local min_id = tonumber(ARGV[1])
local max_id = tonumber(ARGV[2])

-- 1. 尝试从释放列表中获取
local room_id = redis.call('LPOP', released_list_key)
if room_id then
    redis.call('SADD', used_set_key, room_id)
    return room_id
end

-- 2. 如果释放列表为空, 使用计数器
-- 检查计数器是否存在，如果不存在，则初始化为 min_id - 1
-- 使用 SETNX 保证只在第一次调用时初始化
local is_counter_set = redis.call('SETNX', counter_key, min_id - 1)

-- 执行 INCR
local next_id = redis.call('INCR', counter_key)

-- 3. 检查ID是否超出范围
if next_id > max_id then
    -- 将计数器重置回最大值，防止无限增长
    redis.call('SET', counter_key, max_id)
    -- 检查used_set中是否有空闲ID，这是更复杂的逻辑，暂时返回耗尽错误
    if redis.call('SCARD', used_set_key) >= (max_id - min_id + 1) then
        return "EXHAUSTED"
    end
    -- 如果set未满，说明有ID被释放但又被用完，此时理论上应该从released_list拿到
    -- 但到这一步说明没拿到，返回耗尽错误比较安全
    return "EXHAUSTED"
end

-- 4. 将新生成的ID加入到used集合中
redis.call('SADD', used_set_key, next_id)

return tostring(next_id)
`

// SequentialIDManager 负责管理连续房间ID的分配和释放
type SequentialIDManager struct {
	client                 redis.Cmdable
	baseKey                string
	usedRoomIDsSetKey      string
	releasedRoomIDsListKey string
	counterKey             string
	maxReleasedIDsToStore  int64
	getScriptSHA           string
	releaseScriptSHA       string // 复用原来的 release script
}

// NewSequentialIDManager 创建一个新的 SequentialIDManager 实例
func NewSequentialIDManager(client redis.Cmdable, baseKey string) (*SequentialIDManager, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	if baseKey == "" {
		return nil, fmt.Errorf("baseKey cannot be empty")
	}

	m := &SequentialIDManager{
		client:                 client,
		baseKey:                baseKey,
		usedRoomIDsSetKey:      seqUsedRoomIDsSetKeyPrefix + baseKey,
		releasedRoomIDsListKey: seqReleasedRoomIDsListKeyPrefix + baseKey,
		counterKey:             seqCounterKeyPrefix + baseKey,
		maxReleasedIDsToStore:  defaultMaxReleasedIDsToStore, // 复用常量
	}

	// 预加载 Lua 脚本
	if err := m.loadScripts(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load initial lua scripts: %w", err)
	}

	return m, nil
}

// loadScripts 将Lua脚本加载到Redis并存储SHA摘要
func (m *SequentialIDManager) loadScripts(ctx context.Context) error {
	// 加载获取ID的脚本
	shaGet, err := m.client.ScriptLoad(ctx, luaGetSequentialIDScript).Result()
	if err != nil {
		return fmt.Errorf("failed to load get_sequential_id script: %w", err)
	}
	m.getScriptSHA = shaGet

	// 加载释放ID的脚本 (可以复用原有的luaReleaseScript)
	shaRelease, err := m.client.ScriptLoad(ctx, luaReleaseScript).Result()
	if err != nil {
		return fmt.Errorf("failed to load release script: %w", err)
	}
	m.releaseScriptSHA = shaRelease
	return nil
}

// GetNextID 获取下一个可用的房间号 (优先重用，否则连续递增)
func (m *SequentialIDManager) GetNextID(ctx context.Context) (int64, error) {
	result, err := m.client.EvalSha(ctx, m.getScriptSHA,
		[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey, m.counterKey},
		minRoomIDValue, maxRoomIDValue,
	).Result()

	// 如果脚本未加载 (例如Redis重启), 重新加载并重试
	if err != nil && strings.Contains(err.Error(), "NOSCRIPT") {
		if loadErr := m.loadScripts(ctx); loadErr != nil {
			return 0, fmt.Errorf("failed to reload scripts after NOSCRIPT error: %w", loadErr)
		}
		// 重试
		result, err = m.client.EvalSha(ctx, m.getScriptSHA,
			[]string{m.releasedRoomIDsListKey, m.usedRoomIDsSetKey, m.counterKey},
			minRoomIDValue, maxRoomIDValue,
		).Result()
	}

	if err != nil {
		return 0, fmt.Errorf("error executing get_sequential_id script: %w", err)
	}

	idStr, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected type returned from lua script: %T", result)
	}

	if idStr == "EXHAUSTED" {
		return 0, ErrIDPoolExhausted
	}

	roomID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse room ID '%s' from script result: %w", idStr, err)
	}

	return roomID, nil
}

// ReleaseID 释放一个房间号，使其可被重用
func (m *SequentialIDManager) ReleaseID(ctx context.Context, roomID int64) error {
	if roomID < minRoomIDValue || roomID > maxRoomIDValue {
		return fmt.Errorf("roomID %d is out of valid range [%d, %d]", roomID, minRoomIDValue, maxRoomIDValue)
	}

	roomIDStr := formatRoomID(roomID) // 复用原有的格式化函数
	maxReleasedStr := strconv.FormatInt(m.maxReleasedIDsToStore, 10)

	_, err := m.client.EvalSha(ctx, m.releaseScriptSHA,
		[]string{m.usedRoomIDsSetKey, m.releasedRoomIDsListKey},
		roomIDStr, maxReleasedStr,
	).Result()

	if err != nil && strings.Contains(err.Error(), "NOSCRIPT") {
		if loadErr := m.loadScripts(ctx); loadErr != nil {
			return fmt.Errorf("failed to reload scripts during release: %w", loadErr)
		}
		// 重试
		_, err = m.client.EvalSha(ctx, m.releaseScriptSHA,
			[]string{m.usedRoomIDsSetKey, m.releasedRoomIDsListKey},
			roomIDStr, maxReleasedStr,
		).Result()
	}

	if err != nil {
		return fmt.Errorf("failed to release room ID %d using script: %w", roomID, err)
	}

	return nil
}
