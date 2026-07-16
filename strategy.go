package botapi_fsm

type KeyStrategy struct {
	Key       KeyFunc
	UpdateKey UpdateKeyFunc
}

var (
	StrategyChatSender = KeyStrategy{
		Key:       ChatSenderKey,
		UpdateKey: ChatSenderUpdateKey,
	}

	StrategySender = KeyStrategy{
		Key:       SenderKey,
		UpdateKey: SenderUpdateKey,
	}
)
