package botapi_fsm

// Option configures a [Machine].
type Option[S comparable, D any] func(*Machine[S, D])

// WithKeyFunc sets how session keys are derived from handler contexts.
func WithKeyFunc[S comparable, D any](f KeyFunc) Option[S, D] {
	return func(m *Machine[S, D]) {
		m.key = f
	}
}

// WithUpdateKeyFunc sets how session keys are derived from updates (for predicates).
// Must agree with [WithKeyFunc] on the same updates.
func WithUpdateKeyFunc[S comparable, D any](f UpdateKeyFunc) Option[S, D] {
	return func(m *Machine[S, D]) {
		m.updateKey = f
	}
}

// WithCancelCommand registers a command that clears the active session.
// Pass "" to disable. Default is "cancel".
func WithCancelCommand[S comparable, D any](cmd string) Option[S, D] {
	return func(m *Machine[S, D]) {
		m.cancelCmd = cmd
	}
}

// WithOnCancel is called after /cancel clears a non-idle session.
func WithOnCancel[S comparable, D any](h CancelHandler[S, D]) Option[S, D] {
	return func(m *Machine[S, D]) {
		m.onCancel = h
	}
}

// WithOnUnknown is called when a session references an unregistered state.
func WithOnUnknown[S comparable, D any](h UnknownHandler[S, D]) Option[S, D] {
	return func(m *Machine[S, D]) {
		m.onUnknown = h
	}
}
