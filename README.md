# botapi-fsm

Minimal finite-state machine (FSM) layer for [`github.com/gotd/botapi`](https://github.com/gotd/botapi).

- Generic machine: `Machine[S, D]`
- Pluggable storage: memory, Redis, SQL, etc.
- Native botapi integration: `Mount`, `MountGroup`, `Active`, `InState`

## Install

```bash
go get github.com/fluffur/botapi-fsm
```

## Quick Start

```go
type state string

const (
	idle    state = ""
	askName state = "name"
)

type data struct{ Name string }

m := fsm.NewMemory[state, data](idle)

m.Register(askName, func(c *botapi.Context, sess *fsm.Session[state, data]) error {
	sess.Data.Name = c.Message().Text
	return m.Clear(c)
})

pm := bot.Group(botapi.ChatTypeIs(botapi.ChatTypePrivate))

pm.OnCommand("profile", "Start profile flow", func(c *botapi.Context) error {
	if err := m.Enter(c, askName, data{}); err != nil {
		return err
	}
	_, err := c.Reply("What is your name? /cancel to abort.")
	return err
})

m.MountGroup(pm) // message + callback + /cancel
```

## Redis Store

```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
store := fsm.NewRedisJSONStore[state, data](client, "bot:fsm:", 24*time.Hour)
m := fsm.New[state, data](store, idle)
```
