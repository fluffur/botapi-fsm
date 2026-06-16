package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	fsm "github.com/fluffur/botapi-fsm"
	"github.com/gotd/botapi"
)

type profileState string

const (
	profileIdle      profileState = ""
	profileAwaitName profileState = "name"
	profileAwaitAge  profileState = "age"
)

type profileData struct {
	Name string
}

func main() {
	appID, err := strconv.Atoi(os.Getenv("APP_ID"))
	if err != nil {
		log.Fatal("Invalid app id", err)
	}
	bot, err := botapi.New(os.Getenv("BOT_TOKEN"), botapi.Options{
		AppID:     appID,
		AppHash:   os.Getenv("APP_HASH"),
		FloodWait: true,
	})
	if err != nil {
		log.Fatal("Create bot", err)
	}

	store := fsm.NewMemoryStore[profileState, profileData]()

	profileFSM(bot, store)

	if err := bot.Run(nil); err != nil {
		panic(err)
	}
}

func profileFSM(bot *botapi.Bot, store fsm.Store[profileState, profileData]) {
	m := fsm.New(store, profileIdle,
		fsm.WithKeyFunc[profileState, profileData](fsm.ChatSenderKey),
		fsm.WithUpdateKeyFunc[profileState, profileData](fsm.ChatSenderUpdateKey),
	)

	m.Register(profileAwaitName, func(c *botapi.Context, s *fsm.Session[profileState, profileData]) error {
		name := strings.TrimSpace(c.Message().Text)

		if name == "" {
			_, err := c.Reply("Name cannot be empty")
			return err
		}

		s.Data.Name = name

		if err := m.Enter(c, profileAwaitAge, s.Data); err != nil {
			return err
		}

		_, err := c.Reply("How old are you?")
		return err
	})

	m.Register(profileAwaitAge, func(c *botapi.Context, s *fsm.Session[profileState, profileData]) error {
		age, err := strconv.Atoi(strings.TrimSpace(c.Message().Text))
		if err != nil {
			_, err := c.Reply("Enter a valid age")
			return err
		}

		if err := m.Clear(c); err != nil {
			return err
		}

		_, err = c.Reply(
			fmt.Sprintf("Profile saved: %s (%d years old)", s.Data.Name, age),
		)

		return err
	})

	chat := bot.Group(
		botapi.Not(botapi.ChatTypeIs(botapi.ChatTypePrivate)),
	)

	chat.OnCommand("profile", "Create profile", func(c *botapi.Context) error {
		if err := m.Enter(c, profileAwaitName, profileData{}); err != nil {
			return err
		}

		_, err := c.Reply("What's your name?")
		return err
	})

	m.MountGroup(chat)
}
