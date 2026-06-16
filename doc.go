// Package botapi_fsm provides a finite-state machine layer for github.com/gotd/botapi.
//
// It is storage-agnostic: bring your own [Store] (memory, Redis, Postgres, …)
// or use [NewMemory] for a process-local machine. State and payload types are
// generic so each bot defines its own states and data shape.
//
// Quick start:
//
//	type state string
//
//	const (
//		idle           state = ""
//		awaitingName   state = "name"
//	)
//
//	m := fsm.NewMemory[state, struct{}](idle)
//	m.Register(awaitingName, func(c *botapi.Context, sess *fsm.Session[state, struct{}]) error {
//		// sess.Data, c.Message().Text, …
//		return m.Clear(c)
//	})
//
//	pm := bot.Group(botapi.ChatTypeIs(botapi.ChatTypePrivate))
//	pm.OnCommand("profile", "Start profile wizard", func(c *botapi.Context) error {
//		if err := m.Enter(c, awaitingName, struct{}{}); err != nil {
//			return err
//		}
//		_, err := c.Reply("What is your name? /cancel to abort.")
//		return err
//	})
//	m.MountGroup(pm)
//
// Mount registers message and callback handlers guarded by [Machine.Active]
// so only users with a non-idle session are routed into the FSM. Register FSM
// handlers before generic fallbacks — botapi dispatches the first match.
package botapi_fsm
