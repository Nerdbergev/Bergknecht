package echoHandler

import (
	"log"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

func Handle(client *mautrix.Client, source mautrix.EventSource, evt *event.Event) bool {
	if evt.Type == event.EventMessage {
		m := evt.Content.AsMessage()
		_, err := client.SendText(evt.RoomID, m.Body)
		if err != nil {
			log.Println("EchoHandler: Error sending MEssage: " + err.Error())
			return false
		}
	}
	return true
}
