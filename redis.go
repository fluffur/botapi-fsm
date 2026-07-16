package botapi_fsm

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBytesStore persists FSM payloads in Redis.
// Keys are stored as "<prefix><session-key>" and values are raw bytes.
type RedisBytesStore struct {
	client redis.Cmdable
	prefix string
	ttl    time.Duration
}

// NewRedisBytesStore builds a Redis-backed BytesStore.
// If ttl is zero, keys do not expire.
func NewRedisBytesStore(client redis.Cmdable, prefix string, ttl time.Duration) *RedisBytesStore {
	if prefix == "" {
		prefix = "fsm:"
	}
	return &RedisBytesStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

func (s *RedisBytesStore) Get(ctx context.Context, key int64) ([]byte, bool, error) {
	raw, err := s.client.Get(ctx, s.key(key)).Bytes()
	if err == nil {
		return raw, true, nil
	}
	if err == redis.Nil {
		return nil, false, nil
	}
	return nil, false, err
}

func (s *RedisBytesStore) Set(ctx context.Context, key int64, value []byte) error {
	return s.client.Set(ctx, s.key(key), value, s.ttl).Err()
}

func (s *RedisBytesStore) Clear(ctx context.Context, key int64) error {
	return s.client.Del(ctx, s.key(key)).Err()
}

func (s *RedisBytesStore) key(key int64) string {
	return s.prefix + strconv.FormatInt(key, 10)
}

// NewRedisJSONStore is a convenience constructor for JSON-encoded sessions.
func NewRedisJSONStore[S comparable, D any](
	client redis.Cmdable,
	prefix string,
	ttl time.Duration,
) Store[S, D] {
	return JSONStore[S, D](NewRedisBytesStore(client, prefix, ttl))
}

// MustPingRedis verifies connectivity and returns detailed error.
func MustPingRedis(ctx context.Context, client redis.Cmdable) error {
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

func NewRedisFSM[S ~string, D any](
	client redis.UniversalClient,
	prefix string,
	expiration time.Duration,
	idleState S,
	opts ...Option[S, D],
) *Machine[S, D] {
	store := NewRedisJSONStore[S, D](client, prefix, expiration)

	defaultOpts := []Option[S, D]{
		WithKeyFunc[S, D](ChatSenderKey),
		WithUpdateKeyFunc[S, D](ChatSenderUpdateKey),
	}

	finalOpts := append(defaultOpts, opts...)

	return New[S, D](
		store,
		idleState,
		finalOpts...,
	)
}

func WithStrategy[S ~string, D any](strat KeyStrategy) Option[S, D] {
	return func(m *Machine[S, D]) {
		m.key = strat.Key
		m.updateKey = strat.UpdateKey
	}
}
