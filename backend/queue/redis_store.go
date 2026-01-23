package queue

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	c   *redis.Client
	ctx context.Context
}

func NewRedisStore(c *redis.Client) *RedisStore {
	if c == nil {
		return nil
	}
	return &RedisStore{c: c, ctx: context.Background()}
}

func NewRedisClientFromEnv() *redis.Client {
	addr := strings.TrimSpace(os.Getenv("KABAO_REDIS_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	password := os.Getenv("KABAO_REDIS_PASSWORD")
	db := 0
	if s := strings.TrimSpace(os.Getenv("KABAO_REDIS_DB")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			db = n
		}
	}

	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func InitDefaultStoreFromEnv() {
	backend := strings.TrimSpace(os.Getenv("KABAO_QUEUE_BACKEND"))
	if backend == "" {
		backend = "redis"
	}
	if backend != "redis" {
		Default = NewMemStore()
		return
	}

	c := NewRedisClientFromEnv()
	rs := NewRedisStore(c)
	if rs == nil {
		log.Println("[queue] init redis store failed: nil client, fallback to mem")
		Default = NewMemStore()
		return
	}
	if err := rs.Ping(); err != nil {
		log.Printf("[queue] redis ping failed (%v), addr=%s, fallback to mem\n", err, c.Options().Addr)
		Default = NewMemStore()
		return
	}
	log.Printf("[queue] redis queue enabled, addr=%s\n", c.Options().Addr)
	Default = rs
}

func (s *RedisStore) Enqueue(merchantID uint, date string, qt QueueType, id uint, startNo int, now time.Time) (Ticket, bool) {
	if s == nil || s.c == nil {
		return Ticket{}, false
	}
	if startNo <= 0 {
		startNo = 1
	}

	k := makeKey(merchantID, date, qt)
	listKey := "q:order:" + k
	startKey := "q:start:" + k
	calledKey := "q:called:" + k
	enqKey := "q:enq:" + k

	nowMs := now.UnixMilli()

	res, err := enqueueLua.Run(s.ctx, s.c, []string{listKey, startKey, calledKey, enqKey}, strconv.FormatUint(uint64(id), 10), strconv.Itoa(startNo), strconv.FormatInt(nowMs, 10)).Result()
	if err != nil {
		log.Printf("[queue] redis enqueue failed: merchant=%d date=%s type=%s id=%d err=%v\n", merchantID, date, qt, id, err)
		return Ticket{}, false
	}
	vals, ok := res.([]interface{})
	if !ok || len(vals) < 4 {
		log.Printf("[queue] redis enqueue unexpected result: merchant=%d date=%s type=%s id=%d res=%T len=%d\n", merchantID, date, qt, id, res, func() int { if ok { return len(vals) }; return -1 }())
		return Ticket{}, false
	}

	no := int(toInt64(vals[0]))
	created := toInt64(vals[1]) == 1
	calledAtMs := toInt64(vals[2])
	enqAtMs := toInt64(vals[3])

	enqAt := time.UnixMilli(enqAtMs)
	var calledAt *time.Time
	if calledAtMs > 0 {
		t := time.UnixMilli(calledAtMs)
		calledAt = &t
	}
	return Ticket{ID: id, No: no, EnqueuedAt: enqAt, CalledAt: calledAt}, created
}

func (s *RedisStore) MarkDoneAndCallNext(merchantID uint, date string, qt QueueType, doneID uint, now time.Time) (nextID uint) {
	if s == nil || s.c == nil {
		return 0
	}
	k := makeKey(merchantID, date, qt)
	listKey := "q:order:" + k
	calledKey := "q:called:" + k
	enqKey := "q:enq:" + k

	nowMs := now.UnixMilli()

	res, err := doneNextLua.Run(s.ctx, s.c, []string{listKey, calledKey, enqKey}, strconv.FormatUint(uint64(doneID), 10), strconv.FormatInt(nowMs, 10)).Result()
	if err != nil {
		log.Printf("[queue] redis done-next failed: merchant=%d date=%s type=%s done_id=%d err=%v\n", merchantID, date, qt, doneID, err)
		return 0
	}
	return uint(toInt64(res))
}

func (s *RedisStore) Snapshot(merchantID uint, date string, qt QueueType) Snapshot {
	out := Snapshot{Tickets: nil, ByID: make(map[uint]Ticket)}
	if s == nil || s.c == nil {
		return out
	}

	k := makeKey(merchantID, date, qt)
	listKey := "q:order:" + k
	startKey := "q:start:" + k
	calledKey := "q:called:" + k
	enqKey := "q:enq:" + k

	pipe := s.c.Pipeline()
	startCmd := pipe.Get(s.ctx, startKey)
	orderCmd := pipe.LRange(s.ctx, listKey, 0, -1)
	_, _ = pipe.Exec(s.ctx)

	startNo := 1
	if v, err := startCmd.Result(); err == nil {
		if n, err2 := strconv.Atoi(v); err2 == nil {
			startNo = n
		}
	}
	idsStr, err := orderCmd.Result()
	if err != nil || len(idsStr) == 0 {
		return out
	}

	ids := make([]uint, 0, len(idsStr))
	fields := make([]string, 0, len(idsStr))
	for _, sID := range idsStr {
		u64, err := strconv.ParseUint(sID, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, uint(u64))
		fields = append(fields, sID)
	}
	if len(ids) == 0 {
		return out
	}

	pipe2 := s.c.Pipeline()
	calledCmd := pipe2.HMGet(s.ctx, calledKey, fields...)
	enqCmd := pipe2.HMGet(s.ctx, enqKey, fields...)
	_, _ = pipe2.Exec(s.ctx)

	calledVals, _ := calledCmd.Result()
	enqVals, _ := enqCmd.Result()

	for idx, id := range ids {
		no := startNo + idx

		enqAt := time.Time{}
		if idx < len(enqVals) {
			ms := toInt64(enqVals[idx])
			if ms > 0 {
				enqAt = time.UnixMilli(ms)
			}
		}
		var calledAt *time.Time
		if idx < len(calledVals) {
			ms := toInt64(calledVals[idx])
			if ms > 0 {
				t := time.UnixMilli(ms)
				calledAt = &t
			}
		}

		t := Ticket{ID: id, No: no, EnqueuedAt: enqAt, CalledAt: calledAt}
		out.Tickets = append(out.Tickets, t)
		out.ByID[id] = t
	}

	out.CurrentID = ids[0]
	if len(ids) > 1 {
		out.NextID = ids[1]
	}
	if tk, ok := out.ByID[out.CurrentID]; ok {
		out.CurrentCalledAt = tk.CalledAt
	}
	return out
}

var enqueueLua = redis.NewScript(`
-- KEYS: [listKey, startKey, calledHash, enqHash]
-- ARGV: [id, startNo, nowMs]
local listKey = KEYS[1]
local startKey = KEYS[2]
local calledKey = KEYS[3]
local enqKey = KEYS[4]

local id = ARGV[1]
local startNo = tonumber(ARGV[2])
local nowMs = tonumber(ARGV[3])

if redis.call('EXISTS', startKey) == 0 then
  redis.call('SET', startKey, startNo)
end
local start = tonumber(redis.call('GET', startKey) or startNo)

local pos = redis.call('LPOS', listKey, id)
if pos then
  local called = redis.call('HGET', calledKey, id)
  local enq = redis.call('HGET', enqKey, id)
  local calledMs = tonumber(called or '0')
  local enqMs = tonumber(enq or tostring(nowMs))
  local no = start + tonumber(pos)
  return {no, 0, calledMs, enqMs}
end

redis.call('RPUSH', listKey, id)
redis.call('HSET', enqKey, id, nowMs)
local len = redis.call('LLEN', listKey)
if tonumber(len) == 1 then
  redis.call('HSET', calledKey, id, nowMs)
  return {start, 1, nowMs, nowMs}
end
local no = start + tonumber(len) - 1
return {no, 1, 0, nowMs}
`)

var doneNextLua = redis.NewScript(`
-- KEYS: [listKey, calledHash, enqHash]
-- ARGV: [doneID, nowMs]
local listKey = KEYS[1]
local calledKey = KEYS[2]
local enqKey = KEYS[3]

local doneID = ARGV[1]
local nowMs = tonumber(ARGV[2])

local head = redis.call('LINDEX', listKey, 0)
if not head then
  return 0
end

if tostring(head) == tostring(doneID) then
  redis.call('LPOP', listKey)
end
redis.call('HDEL', calledKey, doneID)
redis.call('HDEL', enqKey, doneID)

local nextID = redis.call('LINDEX', listKey, 0)
if not nextID then
  return 0
end

local called = redis.call('HGET', calledKey, nextID)
if not called then
  redis.call('HSET', calledKey, nextID, nowMs)
end
return tonumber(nextID)
`)

func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case int64:
		return t
	case int:
		return int64(t)
	case uint64:
		return int64(t)
	case string:
		i, _ := strconv.ParseInt(t, 10, 64)
		return i
	case []byte:
		i, _ := strconv.ParseInt(string(t), 10, 64)
		return i
	default:
		return 0
	}
}

func (s *RedisStore) Ping() error {
	if s == nil || s.c == nil {
		return errors.New("nil redis client")
	}
	ctx, cancel := context.WithTimeout(s.ctx, 800*time.Millisecond)
	defer cancel()
	return s.c.Ping(ctx).Err()
}
