package botapi_fsm

import (
	"github.com/gotd/botapi"
)

// KeyFunc extracts a session key from a handler context.
// The default is [SenderKey].
type KeyFunc func(c *botapi.Context) (int64, bool)

// UpdateKeyFunc extracts a session key from a raw update.
// Used by predicates; must stay consistent with [KeyFunc].
type UpdateKeyFunc func(u *botapi.Update) (int64, bool)

// SenderKey uses the update sender's user id (messages, callbacks, inline).
// In private chats it falls back to chat id when From is absent — that can
// happen with some MTProto updates even though chat id equals the user id.
func SenderKey(c *botapi.Context) (int64, bool) {
	if u := c.Sender(); u != nil {
		return u.ID, true
	}
	return privateChatKey(c)
}

// SenderUpdateKey is the update-level counterpart of [SenderKey].
func SenderUpdateKey(u *botapi.Update) (int64, bool) {
	switch {
	case u.CallbackQuery != nil:
		if u.CallbackQuery.From.ID != 0 {
			return u.CallbackQuery.From.ID, true
		}
	case u.InlineQuery != nil:
		return u.InlineQuery.From.ID, true
	}

	if m := u.EffectiveMessage(); m != nil {
		if m.From != nil {
			return m.From.ID, true
		}
		if m.Chat.Type == botapi.ChatTypePrivate {
			return m.Chat.ID, true
		}
	}

	if u.CallbackQuery != nil && u.CallbackQuery.Message != nil {
		msg := u.CallbackQuery.Message
		if msg.Chat.Type == botapi.ChatTypePrivate {
			return msg.Chat.ID, true
		}
	}

	return 0, false
}

// PrivateChatKey scopes sessions to the private-chat user id.
// Prefer [SenderKey]; use this explicitly when you only ever run in PM.
func PrivateChatKey(c *botapi.Context) (int64, bool) {
	if u := c.Sender(); u != nil {
		return u.ID, true
	}
	return privateChatKey(c)
}

func privateChatKey(c *botapi.Context) (int64, bool) {
	if m := c.Message(); m != nil && m.Chat.Type == botapi.ChatTypePrivate {
		return m.Chat.ID, true
	}
	if cq := c.Update.CallbackQuery; cq != nil && cq.Message != nil &&
		cq.Message.Chat.Type == botapi.ChatTypePrivate {
		return cq.Message.Chat.ID, true
	}
	return 0, false
}

// ChatSenderKey scopes sessions to a chat and user pair (for group FSMs).
func ChatSenderKey(c *botapi.Context) (int64, bool) {
	uid, ok := SenderKey(c)
	if !ok {
		return 0, false
	}

	chat, ok := c.Chat()
	if !ok {
		return 0, false
	}

	id, ok := chatIDInt(chat)
	if !ok {
		return 0, false
	}

	return (id << 32) ^ uid, true
}

// ChatSenderUpdateKey is the update-level counterpart of [ChatSenderKey].
func ChatSenderUpdateKey(u *botapi.Update) (int64, bool) {
	uid, ok := SenderUpdateKey(u)
	if !ok {
		return 0, false
	}

	var chatID int64
	switch {
	case u.CallbackQuery != nil && u.CallbackQuery.Message != nil:
		chatID = u.CallbackQuery.Message.Chat.ID
	default:
		if m := u.EffectiveMessage(); m != nil {
			chatID = m.Chat.ID
		}
	}

	if chatID == 0 {
		return 0, false
	}

	return (chatID << 32) ^ uid, true
}

func chatIDInt(chat botapi.ChatID) (int64, bool) {
	id, ok := chat.(botapi.ChatIDInt)
	return int64(id), ok
}
