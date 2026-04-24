-- KEYS[1] = key
-- ARGV[1] = rate_per_sec
-- ARGV[2] = burst
-- ARGV[3] = cost
-- ARGV[4] = ttl_ms (optional, can be 0)

local key = KEYS[1]
local rate = tonumber(ARGV[1]) * 1000 -- 1 token = 1000 milli tokens
local burst = tonumber(ARGV[2]) * 1000
local cost = tonumber(ARGV[3]) * 1000
local ttl = tonumber(ARGV[4])

local t = redis.call("TIME") -- 1: unix timestamp 2: microsec eclapsed in current second
local now = t[1] * 1000 + math.floor(t[2]/1000)
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
local refill = (delta * rate) / 1000.0
tokens = math.min(burst, tokens + refill)

-- prevent ts from going backward if Redis clock is adjusted (NTP)
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
