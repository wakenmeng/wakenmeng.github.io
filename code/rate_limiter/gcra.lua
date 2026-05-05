-- gcra.lua
-- ARGV[1] = rate_per_sec
-- ARGV[2] = burst
-- ARGV[3] = cost
-- ARGV[4] = ttl_ms (optional, can be 0)

local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

if burst < 1 or rate < 1 or cost < 1 then
  return redis.error_reply("invalid config: burst/rate/cost must be >= 1")
end

local T_us = math.floor(1000000 / rate) -- emission interval, us per token
local tau_us = (burst-1) * T_us     -- tolerance, us
local t = redis.call("TIME") -- arr, 1: unix timestamp in second; 2: microsec eclapsed in current second
local now_us = t[1] * 1000000 + t[2] -- microsecond

local tat = redis.call("GET", key) -- theoratical arrival time

tat = tat and tonumber(tat) or now_us

local allowed = 0
local retry_after_ms = 0

if now_us >= tat - tau_us then
  allowed = 1
  tat = math.max(tat, now_us) + T_us * cost
  redis.call("SET", key, tat)
  if ttl ~= nil and ttl > 0 then redis.call("PEXPIRE", key, ttl) end -- ms
else
  allowed = 0
  retry_after_ms = math.ceil((tat - tau_us - now_us) / 1000)
end
  


return {allowed, retry_after_ms}
