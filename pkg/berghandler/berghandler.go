package berghandler

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strings"

	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"go.uber.org/zap"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var CommandPrefix = "!"

const WrongArguments = "Falsche Anzahl an Argumenten, benutze %v help für Hilfe."
const unkownCommand = "Unbekanntes Kommando, benutze %v help für Hilfe."

type HandlerEssentials struct {
	Client  *mautrix.Client
	Logger  *zap.SugaredLogger
	Storage *storage.Manager
}

type BergEventHandler interface {
	Handle(he HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool
	GetName() string
	GetCommand() string
	Prime(he HandlerEssentials) error
}

type BergEventHandleFunction func(he HandlerEssentials, evt *event.Event, words []string) bool

type SubHandlerSet struct {
	F BergEventHandleFunction
	H string
}

type SubHandlers map[string]SubHandlerSet

func (s *SubHandlers) getAvailableCommands() string {
	ss := *s
	result := ""
	for k := range ss {
		result += (k + " ")
	}
	return result
}

func (s *SubHandlers) Handle(command string, handlerName string, he HandlerEssentials, evt *event.Event) bool {
	if IsMessagewithPrefix(evt, command) {
		m := evt.Content.AsMessage()
		words, err := StripPrefixandGetContent(m.Body, command)
		if err != nil {
			return SendMessage(he, evt, handlerName, "Fehler bei decodieren der Nachricht: "+err.Error())
		}
		cmd := strings.ToLower(words[0])
		newwords := RemoveWord(words, 0)

		ss := *s

		if strings.Compare(cmd, "help") == 0 {
			if len(newwords) == 0 {
				msg := "Verfügbare Kommandos sind: \n"
				msg += s.getAvailableCommands()
				return SendMessage(he, evt, handlerName, msg)
			}
			set := ss[newwords[0]]
			f := set.F
			if f != nil {
				return SendMessage(he, evt, handlerName, set.H)
			}
		}
		set := ss[cmd]
		f := set.F
		if f == nil {
			return SendMessage(he, evt, handlerName, fmt.Sprintf(unkownCommand, CommandPrefix+command))
		}
		return f(he, evt, newwords)
	}
	return false
}

func SetCommandPrefix(prefix string) {
	CommandPrefix = prefix
}

func IsMessagewithPrefix(evt *event.Event, prefix string) bool {
	result := false
	if evt.Type == event.EventMessage {
		m := evt.Content.AsMessage()
		result = strings.HasPrefix(strings.ToLower(m.Body), CommandPrefix+prefix)
	}
	return result
}

func StripPrefix(message, prefix string) string {
	return strings.TrimPrefix(message, CommandPrefix+prefix+" ")
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
