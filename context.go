package botapi_fsm

import "context"

type sessionKey[S comparable, D any] struct{}

// WithSession attaches a session to ctx for downstream handlers and middleware.
func WithSession[S comparable, D any](ctx context.Context, sess Session[S, D]) context.Context {
	return context.WithValue(ctx, sessionKey[S, D]{}, sess)
}

// SessionFromContext returns a session previously stored with [WithSession].
func SessionFromContext[S comparable, D any](ctx context.Context) (Session[S, D], bool) {
	sess, ok := ctx.Value(sessionKey[S, D]{}).(Session[S, D])
	return sess, ok
}
