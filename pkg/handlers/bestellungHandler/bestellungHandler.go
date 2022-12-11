package bestellungHandler

import (
	"math/rand"
	"strings"
	"time"

	"github.com/Nerdbergev/Bergknecht/pkg/berghandler"
	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var handlerName = "BestellungHandler"

type BestellungHandler struct {
	Lieferdienste []LieferDienst
}

type LieferDienst struct {
	Name          string
	Telefonnummer string
	Artikel       []Artikel
}

type Artikel struct {
	Name      string
	Versionen []Version
}

type Version struct {
	Name  string
	Preis float32
}

func (h BestellungHandler) LoadData(he berghandler.HandlerEssentials) error {
	return he.Storage.DecodeFile(handlerName, "lieferdienste.toml", storage.TOML, true, &h)
}

func (h BestellungHandler) GetName() string {
	return handlerName
}

func (h BestellungHandler) Handle(he berghandler.HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool {
	result := false
	if berghandler.IsMessagewithPrefix(evt, "bestellung") {
		m := evt.Content.AsMessage()
		msg := berghandler.StripPrefix(m.Body, "bestellung")
		words := strings.Split(msg, " ")
		if len(words) < 2 {
			return berghandler.SendMessage(he, evt, handlerName, "Zu wenig Argumente, benutze !bestellung help für Hilfe")
		}
		cmd := strings.ToLower(words[0])
		switch cmd {
		case "neu":
			result = h.newOrder(he, evt, words)
		default:
			return berghandler.SendMessage(he, evt, handlerName, "Kein valides Argument, benutze !bestellung help für Hilfe")
		}
	}
	return result
}

func getRandomWord(r *rand.Rand, slice []string) string {
	return slice[rand.Intn(len(slice))]
}

func (h *BestellungHandler) newOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	if len(words) < 2 {
		return berghandler.SendMessage(he, evt, handlerName, "Zu wenig Argumente, benutze !bestellung help für Hilfe")
	}
	ld := strings.ToLower(words[1])
	found := false
	for _, l := range h.Lieferdienste {
		if strings.Compare(ld, l.Name) == 0 {
			found = true
		}
	}
	if !found {
		return berghandler.SendMessage(he, evt, handlerName, "Lieferdienst nicht gefunden, benutze !bestellung dienste für eine Liste")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	z := getRandomWord(r, zahlen)
	a := getRandomWord(r, adjektive)
	n := getRandomWord(r, nomen)
	bn := z + "-" + a + "-" + n
	_, err := he.Storage.GetFile(handlerName, bn, false)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler bei erstellung der Bestellung")
	}
	return berghandler.SendMessage(he, evt, handlerName, "Neue Bestellung mit dem Name: "+bn+" erstellt")
}
