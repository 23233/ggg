package idgenerator

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

const (
	// luaGetNextID defines a Lua script for Redis.
	// It atomically retrieves the next ID.
	// If the counter key does not exist, it initializes it to ARGV[1] (initialValue) and returns initialValue.
	// If the key exists, it increments it and returns the new value.
	// KEYS[1]: The counter key.
	// ARGV[1]: The initial value as a string.
	luaGetNextID = `
local key = KEYS[1]
local initial_val_str = ARGV[1]
local initial_val = tonumber(initial_val_str)

if initial_val == nil then
  return redis.error_reply("ERR initial_val ARGV[1] ('" .. initial_val_str .. "') is not a number")
end

local exists = redis.call('EXISTS', key)

if exists == 0 then
  redis.call('SET', key, initial_val_str) -- Store initial_val as string
  return initial_val -- Return initial_val as integer
else
  return redis.call('INCR', key)
end
`
)

// RedisIDGenerator implements the IDGenerator interface using Redis as a backend.
type RedisIDGenerator struct {
	client       redis.Cmdable // Allows use of redis.Client, redis.ClusterClient, etc.
	counterKey   string
	initialValue int64
	luaScriptSHA string // SHA of the loaded Lua script for GetNextID
}

// NewRedisIDGenerator creates and returns a new RedisIDGenerator instance.
// It preloads the Lua script required for GetNextID operation.
// client: An initialized Redis commandable client (e.g., *redis.Client).
// key: The Redis key to be used for this specific counter.
// initialValue: The value from which the ID should start if the key is new.
func NewRedisIDGenerator(client redis.Cmdable, key string, initialValue int64) (IDGenerator, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	if key == "" {
		return nil, fmt.Errorf("counter key cannot be empty")
	}
	// Consistent with MySQLIDGenerator's behavior
	if initialValue < 0 {
		return nil, fmt.Errorf("initial value cannot be negative for consistency")
	}

	ctx := context.Background() // Use appropriate context in a real application

	// Load the Lua script into Redis and get its SHA sum.
	sha, err := client.ScriptLoad(ctx, luaGetNextID).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua script into Redis for key '%s': %w", key, err)
	}

	return &RedisIDGenerator{
		client:       client,
		counterKey:   key,
		initialValue: initialValue,
		luaScriptSHA: sha,
	}, nil
}

// GetNextID retrieves the next unique ID from Redis.
// It uses a Lua script to ensure atomicity of initializing the counter (if new)
// and incrementing it.
func (g *RedisIDGenerator) GetNextID() (int64, error) {
	ctx := context.Background() // Use appropriate context
	initialValueStr := strconv.FormatInt(g.initialValue, 10)

	// Attempt to execute the script using its SHA.
	val, err := g.client.EvalSha(ctx, g.luaScriptSHA, []string{g.counterKey}, initialValueStr).Result()
	if err != nil {
		// If the script is not found by SHA (NOSCRIPT error), load it again and then execute.
		// This can happen if Redis was flushed or the script cache was cleared.
		if redisErr, ok := err.(redis.Error); ok && len(redisErr.Error()) >= 8 && redisErr.Error()[:8] == "NOSCRIPT" {
			var loadErr error
			g.luaScriptSHA, loadErr = g.client.ScriptLoad(ctx, luaGetNextID).Result() // Reload and update SHA
			if loadErr != nil {
				return 0, fmt.Errorf("failed to reload Lua script for key '%s' after NOSCRIPT error: %w", g.counterKey, loadErr)
			}
			// Retry with EvalSha or just Eval, Eval is safer now as script is definitely loaded in this session.
			val, err = g.client.Eval(ctx, luaGetNextID, []string{g.counterKey}, initialValueStr).Result()
			if err != nil {
				return 0, fmt.Errorf("failed to EVAL Lua script for key '%s' after reload: %w", g.counterKey, err)
			}
		} else {
			// Other types of errors from EvalSha
			return 0, fmt.Errorf("failed to EVALSHA Lua script for key '%s': %w", g.counterKey, err)
		}
	}

	// The Lua script returns an integer (or a Redis error, handled above).
	id, ok := val.(int64)
	if !ok {
		// This should ideally not happen if the Lua script is correct and error handling is robust.
		// It might be a Redis error reply that was not converted to a Go error by the client.
		if redisErr, isRedisErr := val.(redis.Error); isRedisErr {
			return 0, fmt.Errorf("lua script for key '%s' returned a Redis error: %s", g.counterKey, redisErr.Error())
		}
		return 0, fmt.Errorf("lua script for key '%s' returned an unexpected type %T (value: %v)", g.counterKey, val, val)
	}
	return id, nil
}

// GetCurrentNextID queries and returns the next ID that would be assigned by GetNextID.
// If the counter key doesn't exist in Redis, it returns the configured initialValue.
// Otherwise, it returns the current counter value incremented by one.
func (g *RedisIDGenerator) GetCurrentNextID() (int64, error) {
	ctx := context.Background() // Use appropriate context

	valStr, err := g.client.Get(ctx, g.counterKey).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist, so the first ID to be generated will be initialValue.
			return g.initialValue, nil
		}
		return 0, fmt.Errorf("failed to GET redis counter for key '%s': %w", g.counterKey, err)
	}

	lastAssignedID, parseErr := strconv.ParseInt(valStr, 10, 64)
	if parseErr != nil {
		// This implies the key holds a non-integer value, which shouldn't happen
		// if only this ID generator manipulates it.
		return 0, fmt.Errorf("failed to parse stored redis value '%s' for key '%s' as int64: %w", valStr, g.counterKey, parseErr)
	}

	// The value stored (lastAssignedID) is the last ID that was generated.
	// The next ID to be generated will be lastAssignedID + 1.
	return lastAssignedID + 1, nil
}
