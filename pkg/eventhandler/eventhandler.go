package eventhandler

import (
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type BergEventHandleFunction func(client *mautrix.Client, logger *zap.SugaredLogger, source mautrix.EventSource, evt *event.Event) bool
