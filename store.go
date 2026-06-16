package botapi_fsm

import (
	"context"
	"sync"
)

// Store persists FSM sessions. Implement it for Redis, SQL, or any backend.
type Store[S comparable, D any] interface {
	Get(ctx context.Context, key int64) (Session[S, D], bool, error)
	Set(ctx context.Context, key int64, sess Session[S, D]) error
	Clear(ctx context.Context, key int64) error
}

// MemoryStore keeps sessions in process memory.
type MemoryStore[S comparable, D any] struct {
	mu   sync.RWMutex
	data map[int64]Session[S, D]
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore[S comparable, D any]() *MemoryStore[S, D] {
	return &MemoryStore[S, D]{data: make(map[int64]Session[S, D])}
}

func (s *MemoryStore[S, D]) Get(_ context.Context, key int64) (Session[S, D], bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.data[key]
	return sess, ok, nil
}

func (s *MemoryStore[S, D]) Set(_ context.Context, key int64, sess Session[S, D]) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = sess
	return nil
}

func (s *MemoryStore[S, D]) Clear(_ context.Context, key int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

type sessionCache[S comparable, D any] struct {
	mu   sync.RWMutex
	data map[int64]Session[S, D]
}

func newSessionCache[S comparable, D any]() *sessionCache[S, D] {
	return &sessionCache[S, D]{data: make(map[int64]Session[S, D])}
}

func (c *sessionCache[S, D]) peek(key int64) (Session[S, D], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sess, ok := c.data[key]
	return sess, ok
}

func (c *sessionCache[S, D]) set(key int64, sess Session[S, D]) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = sess
}

func (c *sessionCache[S, D]) remove(key int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
}
