package eventhandler

import (
	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type HandlerEssentials struct {
	Client  *mautrix.Client
	Logger  *zap.SugaredLogger
	Storage *storage.Manager
}

type BergEventHandleFunction func(he HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool
