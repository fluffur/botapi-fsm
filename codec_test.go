package botapi_fsm_test

import (
	"context"
	"sync"
	"testing"

	"activity-bot/fsm"
)

type mapBytesStore struct {
	mu   sync.Mutex
	data map[int64][]byte
}

func (s *mapBytesStore) Get(_ context.Context, key int64) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	return v, ok, nil
}

func (s *mapBytesStore) Set(_ context.Context, key int64, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *mapBytesStore) Clear(_ context.Context, key int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func TestJSONStore(t *testing.T) {
	t.Parallel()

	backend := &mapBytesStore{data: make(map[int64][]byte)}
	store := fsm.JSONStore[step, int](backend)
	ctx := context.Background()

	if err := store.Set(ctx, 1, fsm.Session[step, int]{State: ask, Data: 99}); err != nil {
		t.Fatal(err)
	}

	sess, ok, err := store.Get(ctx, 1)
	if err != nil || !ok || sess.State != ask || sess.Data != 99 {
		t.Fatalf("sess=%+v ok=%v err=%v", sess, ok, err)
	}
}
