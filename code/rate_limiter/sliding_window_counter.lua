-- sliding_window_counter.lua
-- 
-- ARGV[1] = limit (max per window)
-- ARGV[2] = window_ms
-- ARGV[3] = cost
-- ARGV[4] = ttl_ms (optional, can be 0)

local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])
local ttl_ms = tonumber(ARGV[4])


if limit < 1 or window_ms < 1 or cost < 1 then
  return redis.error_reply("invalid config: limit/window/cost must be >= 1")
end

local t = redis.call("TIME") -- arr, 1: unix timestamp in second; 2: microsec eclapsed in current second
local now_ms = t[1] * 1000 + math.floor(t[2]/1000)

local cur_window = math.floor(now_ms/window_ms) * window_ms
local prev_window = cur_window - window_ms

local vals = redis.call("HMGET", key, "win", "cur", "prev")
local stored_win = tonumber(vals[1]) or 0
local stored_cur = tonumber(vals[2]) or 0
local stored_prev = tonumber(vals[3]) or 0

local cur_cnt = 0
local prev_cnt = 0

if stored_win == cur_window then
  cur_cnt = stored_cur
  prev_cnt = stored_prev
elseif stored_win == prev_window then
  cur_cnt = 0
  prev_cnt = stored_cur
else
  cur_cnt = 0
  prev_cnt = 0
end

local elapsed_in_cur = now_ms - cur_window
local prev_weight = (window_ms - elapsed_in_cur) / window_ms
local estimated = prev_cnt * prev_weight + cur_cnt

local allowed = 0

if estimated + cost <= limit then
  cur_cnt = cur_cnt + cost
  allowed = 1
  redis.call("HSET", key, "win", cur_window, "cur", cur_cnt, "prev", prev_cnt)
  if ttl_ms ~= nil and ttl_ms > 0 then redis.call("PEXPIRE", key, ttl_ms) end -- ms
end

return allowed
