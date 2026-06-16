package botapi_fsm

import (
	"context"
	"encoding/json"
)

// BytesStore is a minimal key-value backend for encoded sessions.
type BytesStore interface {
	Get(ctx context.Context, key int64) ([]byte, bool, error)
	Set(ctx context.Context, key int64, value []byte) error
	Clear(ctx context.Context, key int64) error
}

// JSONStore adapts [BytesStore] to [Store] using JSON encoding.
func JSONStore[S comparable, D any](inner BytesStore) Store[S, D] {
	return &jsonStore[S, D]{inner: inner}
}

type jsonStore[S comparable, D any] struct {
	inner BytesStore
}

func (s *jsonStore[S, D]) Get(ctx context.Context, key int64) (Session[S, D], bool, error) {
	raw, ok, err := s.inner.Get(ctx, key)
	if err != nil || !ok {
		return Session[S, D]{}, ok, err
	}

	var sess Session[S, D]
	if err := json.Unmarshal(raw, &sess); err != nil {
		return Session[S, D]{}, false, err
	}
	return sess, true, nil
}

func (s *jsonStore[S, D]) Set(ctx context.Context, key int64, sess Session[S, D]) error {
	raw, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return s.inner.Set(ctx, key, raw)
}

func (s *jsonStore[S, D]) Clear(ctx context.Context, key int64) error {
	return s.inner.Clear(ctx, key)
}
