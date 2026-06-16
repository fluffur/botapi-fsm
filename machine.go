package botapi_fsm

import (
	"context"
	"errors"

	"github.com/gotd/botapi"
)

// ErrNoSessionKey is returned when a context carries no session key.
var ErrNoSessionKey = errors.New("fsm: no session key in context")

// Handler processes an update while the user is in a specific state.
type Handler[S comparable, D any] func(c *botapi.Context, sess *Session[S, D]) error

// CancelHandler runs after a session is cleared via the cancel command.
type CancelHandler[S comparable, D any] func(c *botapi.Context, sess *Session[S, D]) error

// UnknownHandler runs when stored state has no registered handler (session is cleared first).
type UnknownHandler[S comparable, D any] func(c *botapi.Context, sess Session[S, D]) error

// Machine routes botapi updates to per-state handlers.
type Machine[S comparable, D any] struct {
	store     Store[S, D]
	cache     *sessionCache[S, D]
	idle      S
	handlers  map[S]Handler[S, D]
	key       KeyFunc
	updateKey UpdateKeyFunc
	cancelCmd string
	onCancel  CancelHandler[S, D]
	onUnknown UnknownHandler[S, D]
}

// New builds a machine backed by store. idle is the zero/active-outside-FSM state.
func New[S comparable, D any](store Store[S, D], idle S, opts ...Option[S, D]) *Machine[S, D] {
	m := &Machine[S, D]{
		store:     store,
		cache:     newSessionCache[S, D](),
		idle:      idle,
		handlers:  make(map[S]Handler[S, D]),
		key:       SenderKey,
		updateKey: SenderUpdateKey,
		cancelCmd: "cancel",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// NewMemory is shorthand for an in-memory machine.
func NewMemory[S comparable, D any](idle S, opts ...Option[S, D]) *Machine[S, D] {
	return New(NewMemoryStore[S, D](), idle, opts...)
}

// Register binds a state to its handler.
func (m *Machine[S, D]) Register(state S, h Handler[S, D]) {
	m.handlers[state] = h
}

// Handler returns a botapi handler that dispatches by current session state.
// Pair it with [Machine.Active] (as [Machine.Mount] does) so idle users are not consumed.
func (m *Machine[S, D]) Handler() botapi.Handler {
	return func(c *botapi.Context) error {
		key, ok := m.sessionKey(c)
		if !ok {
			return nil
		}

		sess, active, err := m.load(c, key)
		if err != nil {
			return err
		}
		if !active || sess.State == m.idle {
			return nil
		}

		h, ok := m.handlers[sess.State]
		if !ok {
			_ = m.clear(c, key)
			if m.onUnknown != nil {
				return m.onUnknown(c, sess)
			}
			return nil
		}

		c.Context = WithSession(c.Context, sess)
		return h(c, &sess)
	}
}

// Middleware loads the current session into the context without handling the update.
func (m *Machine[S, D]) Middleware() botapi.Middleware {
	return func(next botapi.Handler) botapi.Handler {
		return func(c *botapi.Context) error {
			key, ok := m.key(c)
			if ok {
				if sess, active, err := m.load(c, key); err != nil {
					return err
				} else if active && sess.State != m.idle {
					c.Context = WithSession(c.Context, sess)
				}
			}
			return next(c)
		}
	}
}

// Active is a predicate that matches updates for users currently inside the FSM.
func (m *Machine[S, D]) Active() botapi.Predicate {
	return func(u *botapi.Update) bool {
		key, ok := m.updateKey(u)
		if !ok {
			return false
		}
		sess, ok := m.cache.peek(key)
		return ok && sess.State != m.idle
	}
}

// InState matches updates whose session is one of the given states.
func (m *Machine[S, D]) InState(states ...S) botapi.Predicate {
	set := make(map[S]struct{}, len(states))
	for _, s := range states {
		set[s] = struct{}{}
	}

	return func(u *botapi.Update) bool {
		key, ok := m.updateKey(u)
		if !ok {
			return false
		}
		sess, ok := m.cache.peek(key)
		if !ok {
			return false
		}
		_, ok = set[sess.State]
		return ok
	}
}

// Mount registers FSM handlers on bot (messages + callback queries).
// Extra predicates narrow the scope (e.g. private chats only).
func (m *Machine[S, D]) Mount(bot *botapi.Bot, predicates ...botapi.Predicate) {
	m.mount(bot.OnMessage, bot.OnCallbackQuery, bot.OnCommand, predicates...)
}

// MountGroup registers FSM handlers on a handler group.
func (m *Machine[S, D]) MountGroup(g *botapi.Group, predicates ...botapi.Predicate) {
	m.mount(g.OnMessage, g.OnCallbackQuery, g.OnCommand, predicates...)
}

func (m *Machine[S, D]) mount(
	onMessage func(botapi.Handler, ...botapi.Predicate),
	onCallback func(botapi.Handler, ...botapi.Predicate),
	onCommand func(string, string, botapi.Handler, ...botapi.Predicate),
	predicates ...botapi.Predicate,
) {
	active := append([]botapi.Predicate{m.Active()}, predicates...)

	// Cancel is registered before the catch-all message handler so /cancel is not
	// swallowed as FSM input while a session is active.
	if m.cancelCmd != "" {
		onCommand(m.cancelCmd, "Cancel the current dialog", m.cancelHandler(), predicates...)
	}

	onMessage(m.Handler(), append(active, botapi.HasText())...)
	onCallback(m.Handler(), active...)
}

func (m *Machine[S, D]) cancelHandler() botapi.Handler {
	return func(c *botapi.Context) error {
		key, ok := m.sessionKey(c)
		if !ok {
			return nil
		}

		sess, active, err := m.load(c, key)
		if err != nil {
			return err
		}
		if !active || sess.State == m.idle {
			return nil
		}

		if err := m.clear(c, key); err != nil {
			return err
		}
		if m.onCancel != nil {
			return m.onCancel(c, &sess)
		}
		return nil
	}
}

// Enter starts or replaces a session.
func (m *Machine[S, D]) Enter(c *botapi.Context, state S, data D) error {
	return m.Set(c, Session[S, D]{State: state, Data: data})
}

// Set stores a session for the context's key.
func (m *Machine[S, D]) Set(c *botapi.Context, sess Session[S, D]) error {
	key, ok := m.sessionKey(c)
	if !ok {
		return ErrNoSessionKey
	}
	return m.set(c, key, sess)
}

// Get returns the current session for the context's key.
func (m *Machine[S, D]) Get(c *botapi.Context) (Session[S, D], bool, error) {
	key, ok := m.sessionKey(c)
	if !ok {
		return Session[S, D]{}, false, ErrNoSessionKey
	}
	return m.load(c, key)
}

// Clear removes the session for the context's key.
func (m *Machine[S, D]) Clear(c *botapi.Context) error {
	key, ok := m.sessionKey(c)
	if !ok {
		return ErrNoSessionKey
	}
	return m.clear(c, key)
}

func (m *Machine[S, D]) sessionKey(c *botapi.Context) (int64, bool) {
	if key, ok := m.key(c); ok {
		return key, true
	}
	return PrivateChatKey(c)
}

// Warm loads all sessions from store into the in-process cache.
// Call after restart when using a persistent store so predicates work immediately.
func (m *Machine[S, D]) Warm(ctx context.Context, keys []int64) error {
	for _, key := range keys {
		sess, ok, err := m.store.Get(ctx, key)
		if err != nil {
			return err
		}
		if ok && sess.State != m.idle {
			m.cache.set(key, sess)
		}
	}
	return nil
}

func (m *Machine[S, D]) load(ctx context.Context, key int64) (Session[S, D], bool, error) {
	if sess, ok := m.cache.peek(key); ok {
		return sess, true, nil
	}

	sess, ok, err := m.store.Get(ctx, key)
	if err != nil {
		return Session[S, D]{}, false, err
	}
	if ok && sess.State != m.idle {
		m.cache.set(key, sess)
	}
	return sess, ok, nil
}

func (m *Machine[S, D]) set(ctx context.Context, key int64, sess Session[S, D]) error {
	if err := m.store.Set(ctx, key, sess); err != nil {
		return err
	}
	if sess.State == m.idle {
		m.cache.remove(key)
	} else {
		m.cache.set(key, sess)
	}
	return nil
}

func (m *Machine[S, D]) clear(ctx context.Context, key int64) error {
	if err := m.store.Clear(ctx, key); err != nil {
		return err
	}
	m.cache.remove(key)
	return nil
}
