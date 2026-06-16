package botapi_fsm

// Session is a user's place in the state machine.
type Session[S comparable, D any] struct {
	State S
	Data  D
}
