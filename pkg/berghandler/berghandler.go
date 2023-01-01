package berghandler

import (
	"encoding/csv"
	"errors"
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
		result = strings.HasPrefix(strings.ToLower(m.Body), commandPrefix+prefix)
	}
	return result
}

func StripPrefix(message, prefix string) string {
	return strings.TrimPrefix(message, commandPrefix+prefix+" ")
}

func StripPrefixandGetContent(message, prefix string) ([]string, error) {
	message = StripPrefix(message, prefix)
	r := csv.NewReader(strings.NewReader(message))
	r.Comma = ' '
	return r.Read()
}

func SendMessage(he HandlerEssentials, evt *event.Event, handlerName, msg string) bool {
	_, err := he.Client.SendText(evt.RoomID, msg)
	if err != nil {
		he.Logger.Errorw("Error sending Message", "Handler", handlerName, "Error", err)
		return false
	}
	return true
}

func SendFormattedMessage(he HandlerEssentials, evt *event.Event, handlerName, msg string) bool {
	_, err := he.Client.SendMessageEvent(evt.RoomID, event.EventMessage, &event.MessageEventContent{
		MsgType:       event.MsgText,
		Format:        event.FormatHTML,
		FormattedBody: msg,
	})
	if err != nil {
		he.Logger.Errorw("Error sending Message", "Handler", handlerName, "Error", err)
		return false
	}
	return true
}

func SplitAnswer(words []string, RequiredCount, OptionalCount int, vars ...*string) error {
	TotalCount := RequiredCount + OptionalCount
	if len(vars) < TotalCount {
		return errors.New("variable Count is smaller than total word count")
	}
	if len(words) < RequiredCount {
		return errors.New("too less required Variables")
	}
	end := len(words)
	if len(vars) < len(words) {
		end = len(words)
	}
	for i := 0; i < end; i++ {
		*vars[i] = strings.ToLower(words[i])
	}
	return nil
}

func RemoveWord(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

//type BergEventHandleFunction func(he HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool
