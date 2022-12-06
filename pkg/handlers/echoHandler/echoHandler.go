package echoHandler

import (
	"github.com/Nerdbergev/Bergknecht/pkg/eventhandler"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var handlerName = "EchoHandler"

func Handle(he eventhandler.HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool {
	if evt.Type == event.EventMessage {
		m := evt.Content.AsMessage()
		he.Logger.Infow("Message recieved", "Handler", handlerName, "message", m.Body)
		_, err := he.Client.SendText(evt.RoomID, m.Body)
		if err != nil {
			he.Logger.Errorw("Error sending Message", "Handler", handlerName, "Error", err)
			return false
		}
		f, err := he.Storage.GetFile(handlerName, "messages.txt", true)
		if err != nil {
			he.Logger.Errorw("Error storing Message", "Handler", handlerName, "Error", err)
			return false
		}
		defer f.Close()
		f.WriteString(m.Body)
		f.Sync()
	}
	return true
}
