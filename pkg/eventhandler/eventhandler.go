package eventhandler

import (
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type BergEventHandler func(client *mautrix.Client, source mautrix.EventSource, evt *event.Event) bool
