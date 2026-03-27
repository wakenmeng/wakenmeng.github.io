-- KEYS[1] = key
-- ARGV[1] = now_ms
-- ARGV[2] = rate_per_sec
-- ARGV[3] = burst
-- ARGV[4] = cost
-- ARGV[5] = ttl_ms (optional, can be 0)

local key = KEYS[1]
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local burst = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local tokens = redis.call("HGET", key, "tokens")
local ts = redis.call("HGET", key, "ts")

if tokens == false or ts == false then
  tokens = burst
  ts = now
else
  tokens = tonumber(tokens)
  ts = tonumber(ts)
end

-- refill
local delta = math.max(0, now - ts)
local refill = (delta / 1000.0) * rate
tokens = math.min(burst, tokens + refill)

-- 
ts = math.max(now, ts)

local allowed = 0
local retry_after_ms = 0

if tokens >= cost then
  tokens = tokens - cost
  allowed = 1
else
  allowed = 0
  local need = cost - tokens
  retry_after_ms = math.ceil((need / rate) * 1000.0)
end

redis.call("HSET", key, "tokens", tokens)
redis.call("HSET", key, "ts", ts)

if ttl ~= nil and ttl > 0 then
  redis.call("PEXPIRE", key, ttl)
end

return {allowed, retry_after_ms}
