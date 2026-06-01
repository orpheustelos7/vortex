package ratelimiter

import (
"context"
"fmt"
"time"

"github.com/redis/go-redis/v9"
)

type Result struct {
Allowed   bool
Remaining int64
RetryAfter time.Duration
}

type Limiter interface {
Allow(ctx context.Context, tenant string, rate float64, burst int64) (Result, error)
}

type RedisTokenBucket struct {
client *redis.Client
script *redis.Script
}

func NewRedisTokenBucket(client *redis.Client) *RedisTokenBucket {
return &RedisTokenBucket{
client: client,
script: redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local values = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(values[1])
local ts = tonumber(values[2])

if tokens == nil then
  tokens = capacity
  ts = now
end

local elapsed = math.max(0, now - ts) / 1000.0
local replenished = elapsed * rate
tokens = math.min(capacity, tokens + replenished)

local allowed = 0
local retry_ms = 0
if tokens >= requested then
  allowed = 1
  tokens = tokens - requested
else
  retry_ms = math.ceil(((requested - tokens) / rate) * 1000)
end

redis.call("HMSET", key, "tokens", tokens, "ts", now)
redis.call("PEXPIRE", key, math.ceil((capacity / rate) * 1000 * 2))

return {allowed, math.floor(tokens), retry_ms}
`),
}
}

func (r *RedisTokenBucket) Allow(ctx context.Context, tenant string, rate float64, burst int64) (Result, error) {
if tenant == "" {
return Result{}, fmt.Errorf("tenant is required")
}
if rate <= 0 || burst <= 0 {
return Result{}, fmt.Errorf("invalid rate limit parameters")
}

key := fmt.Sprintf("vortex:bucket:%s", tenant)
now := time.Now().UnixMilli()
res, err := r.script.Run(ctx, r.client, []string{key}, now, rate, burst, 1).Result()
if err != nil {
return Result{}, err
}

vals, ok := res.([]interface{})
if !ok || len(vals) != 3 {
return Result{}, fmt.Errorf("unexpected limiter result")
}

allowed := vals[0].(int64) == 1
remaining := vals[1].(int64)
retryMs := vals[2].(int64)

return Result{Allowed: allowed, Remaining: remaining, RetryAfter: time.Duration(retryMs) * time.Millisecond}, nil
}
