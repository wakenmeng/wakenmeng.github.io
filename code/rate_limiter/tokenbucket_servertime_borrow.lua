-- KEYS[1] = key
-- ARGV[1] = batchSize(need)
-- ARGV[2] = rate
-- ARGV[3] = burst
-- ARGV[4] = ttl_ms (optional, can be 0)

local key = KEYS[1]
local need = tonumber(ARGV[1]) * 1000
local rate = tonumber(ARGV[2]) * 1000
local burst = tonumber(ARGV[3]) * 1000
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

local retry_ms = 0
local borrowed = math.min(tokens, need)

if borrowed < 1000 then
  retry_ms = math.ceil((1000 - tokens) / rate * 1000) 
  if retry_ms < 1 then retry_ms = 1 end
end

tokens = tokens - borrowed

redis.call("HSET", key, "tokens", tokens)
redis.call("HSET", key, "ts", ts)

if ttl ~= nil and ttl > 0 then
  redis.call("PEXPIRE", key, ttl)
end

return {borrowed, retry_ms}
