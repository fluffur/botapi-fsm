package botapi_fsm_test

import (
	"context"
	"testing"

	"github.com/gotd/botapi"

	"activity-bot/fsm"
)

type step string

const (
	idle step = ""
	ask  step = "ask"
)

func TestMachineEnterAndDispatch(t *testing.T) {
	t.Parallel()

	m := fsm.NewMemory[step, string](idle)
	var got string

	m.Register(ask, func(c *botapi.Context, sess *fsm.Session[step, string]) error {
		got = c.Message().Text
		return m.Clear(c)
	})

	h := m.Handler()
	u := &botapi.Update{
		Message: &botapi.Message{
			Text: "Alice",
			From: &botapi.User{ID: 42},
			Chat: botapi.Chat{ID: 1, Type: botapi.ChatTypePrivate},
		},
	}
	c := &botapi.Context{Context: context.Background(), Update: u}

	if err := m.Enter(c, ask, ""); err != nil {
		t.Fatal(err)
	}
	if !m.Active()(u) {
		t.Fatal("expected active predicate")
	}
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if got != "Alice" {
		t.Fatalf("got %q, want Alice", got)
	}
	if m.Active()(u) {
		t.Fatal("expected inactive after clear")
	}
}

func TestInStatePredicate(t *testing.T) {
	t.Parallel()

	m := fsm.NewMemory[step, struct{}](idle)
	u := &botapi.Update{
		Message: &botapi.Message{
			Text: "/start",
			From: &botapi.User{ID: 7},
			Chat: botapi.Chat{ID: 1, Type: botapi.ChatTypePrivate},
		},
	}
	c := &botapi.Context{Context: context.Background(), Update: u}

	if m.InState(ask)(u) {
		t.Fatal("unexpected match before enter")
	}
	if err := m.Enter(c, ask, struct{}{}); err != nil {
		t.Fatal(err)
	}
	if !m.InState(ask)(u) {
		t.Fatal("expected in-state match")
	}
}

func TestUnknownStateClearsSession(t *testing.T) {
	t.Parallel()

	m := fsm.NewMemory[step, struct{}](idle, fsm.WithOnUnknown(func(c *botapi.Context, sess fsm.Session[step, struct{}]) error {
		if sess.State != step("missing") {
			t.Fatalf("state %q", sess.State)
		}
		return nil
	}))

	u := &botapi.Update{
		Message: &botapi.Message{
			Text: "hi",
			From: &botapi.User{ID: 1},
			Chat: botapi.Chat{ID: 1, Type: botapi.ChatTypePrivate},
		},
	}
	c := &botapi.Context{Context: context.Background(), Update: u}

	if err := m.Set(c, fsm.Session[step, struct{}]{State: step("missing")}); err != nil {
		t.Fatal(err)
	}
	if err := m.Handler()(c); err != nil {
		t.Fatal(err)
	}
	if m.Active()(u) {
		t.Fatal("session should be cleared")
	}
}

func TestMemoryStorePersistenceAcrossMachines(t *testing.T) {
	t.Parallel()

	store := fsm.NewMemoryStore[step, int]()
	m1 := fsm.New(store, idle)
	u := &botapi.Update{
		Message: &botapi.Message{
			From: &botapi.User{ID: 99},
			Chat: botapi.Chat{ID: 1, Type: botapi.ChatTypePrivate},
		},
	}
	c := &botapi.Context{Context: context.Background(), Update: u}

	if err := m1.Enter(c, ask, 42); err != nil {
		t.Fatal(err)
	}

	m2 := fsm.New(store, idle)
	sess, ok, err := m2.Get(c)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || sess.State != ask || sess.Data != 42 {
		t.Fatalf("session %+v ok=%v", sess, ok)
	}
}

func TestSessionFromContextMiddleware(t *testing.T) {
	t.Parallel()

	m := fsm.NewMemory[step, struct{}](idle)
	u := &botapi.Update{
		Message: &botapi.Message{
			From: &botapi.User{ID: 5},
			Chat: botapi.Chat{ID: 1, Type: botapi.ChatTypePrivate},
		},
	}
	c := &botapi.Context{Context: context.Background(), Update: u}

	if err := m.Enter(c, ask, struct{}{}); err != nil {
		t.Fatal(err)
	}

	var seen bool
	wrapped := m.Middleware()(func(c *botapi.Context) error {
		_, seen = fsm.SessionFromContext[step, struct{}](c.Context)
		return nil
	})
	if err := wrapped(c); err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("session not injected by middleware")
	}
}
