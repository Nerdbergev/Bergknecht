package echoHandler

import (
	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var handlerName = "EchoHandler"

func Handle(client *mautrix.Client, logger *zap.SugaredLogger, source mautrix.EventSource, evt *event.Event) bool {
	if evt.Type == event.EventMessage {
		m := evt.Content.AsMessage()
		logger.Infow("Message recieved", "Handler", handlerName, "message", m.Body)
		_, err := client.SendText(evt.RoomID, m.Body)
		if err != nil {
			logger.Errorw("Error sending Message", "Handler", handlerName, "Error", err)
			return false
		}
		f, err := storage.GetFile(handlerName, "messages.txt")
		if err != nil {
			logger.Errorw("Error storing Message", "Handler", handlerName, "Error", err)
			return false
		}
		defer f.Close()
		f.WriteString(m.Body)
		f.Sync()
	}
	return true
}
