package berghandler

import (
	"log"
	"strings"

	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var commandPrefix = "!"

type HandlerEssentials struct {
	Client  *mautrix.Client
	Logger  *zap.SugaredLogger
	Storage *storage.Manager
}

type BergEventHandler interface {
	Handle(he HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool
	GetName() string
	LoadData(he HandlerEssentials) error
}

func SetCommandPrefix(prefix string) {
	commandPrefix = prefix
}

func IsMessagewithPrefix(evt *event.Event, prefix string) bool {
	result := false
	if evt.Type == event.EventMessage {
		m := evt.Content.AsMessage()
		log.Println(m.Body)
		result = strings.HasPrefix(strings.ToLower(m.Body), commandPrefix+prefix)
	}
	return result
}

func StripPrefix(message, prefix string) string {
	return strings.TrimPrefix(message, commandPrefix+prefix)
}

func SendMessage(he HandlerEssentials, evt *event.Event, handlerName, msg string) bool {
	_, err := he.Client.SendText(evt.RoomID, msg)
	if err != nil {
		he.Logger.Errorw("Error sending Message", "Handler", handlerName, "Error", err)
		return false
	}
	return true
}

//type BergEventHandleFunction func(he HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool
