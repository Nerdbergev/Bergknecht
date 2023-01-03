package bestellungHandler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Nerdbergev/Bergknecht/pkg/berghandler"
	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type BestellungHandler struct {
	Lieferdienste []LieferDienst
	subHandlers   berghandler.SubHandlers
}

func (h *BestellungHandler) Prime(he berghandler.HandlerEssentials) error {
	h.subHandlers = make(map[string]berghandler.SubHandlerSet)
	h.subHandlers["new"] = berghandler.SubHandlerSet{F: h.newOrder, H: "Erstellt eine Neue Bestellung. \nUsage: new $Lieferdienst"}
	h.subHandlers["add"] = berghandler.SubHandlerSet{F: h.addtoOrder, H: ""}
	h.subHandlers["show"] = berghandler.SubHandlerSet{F: h.printOrder, H: ""}
	h.subHandlers["call-text"] = berghandler.SubHandlerSet{F: h.getCallText, H: ""}
	h.subHandlers["print-payment"] = berghandler.SubHandlerSet{F: h.printPayment, H: ""}
	h.subHandlers["get-total"] = berghandler.SubHandlerSet{F: h.getTotal, H: ""}
	h.subHandlers["remove"] = berghandler.SubHandlerSet{F: h.deletePosition, H: ""}
	h.subHandlers["close"] = berghandler.SubHandlerSet{F: h.removeOrder, H: ""}
	h.subHandlers["add-strichliste"] = berghandler.SubHandlerSet{F: h.addStrichliste, H: ""}
	h.subHandlers["remove-strichliste"] = berghandler.SubHandlerSet{F: h.removeStrichliste, H: ""}

	return he.Storage.DecodeFile(handlerName, "lieferdienste.toml", storage.TOML, true, h)
}

func (h *BestellungHandler) GetName() string {
	return handlerName
}

func (h *BestellungHandler) GetCommand() string {
	return command
}

func (h *BestellungHandler) Handle(he berghandler.HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool {
	return h.subHandlers.Handle(command, handlerName, he, evt)
}

func (h *BestellungHandler) searchLieferdienst(ld string) (bool, LieferDienst) {
	found := false
	res := LieferDienst{}
	for _, l := range h.Lieferdienste {
		if strings.Compare(ld, strings.ToLower(l.Name)) == 0 {
			found = true
			res = l
			break
		}
	}
	return found, res
}

func (h *BestellungHandler) newOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var ld string
	err := berghandler.SplitAnswer(words, 1, 0, &ld)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	found, l := h.searchLieferdienst(ld)
	if !found {
		return berghandler.SendMessage(he, evt, handlerName, "Lieferdienst nicht gefunden, benutze !bestellung dienste für eine Liste")
	}
	z := getRandomWord(zahlen)
	a := getRandomWord(adjektive)
	n := getRandomWord(nomen)
	bn := strings.ToLower(z + "-" + a + "-" + n)
	bnf := bn + ".toml"

	be := Bestellung{}
	be.Ersteller = User{evt.Sender.Localpart(), evt.Sender.String()}
	be.LieferDienst = ld
	be.Nummer = l.Telefonnummer
	err = he.Storage.EncodeFile(handlerName, bnf, storage.TOML, false, be)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler bei erstellung der Bestellung")
	}

	return berghandler.SendMessage(he, evt, handlerName, "Neue Bestellung mit dem Name: "+bn+" erstellt")
}

func (h *BestellungHandler) addtoOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var order, artikel, version, kommentar, anzahl string
	err := berghandler.SplitAnswer(words, 2, 3, &order, &artikel, &version, &kommentar, &anzahl)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	amount := 1
	if anzahl != "" {
		a, err := strconv.Atoi(anzahl)
		if err != nil {
			return berghandler.SendMessage(he, evt, handlerName, "Menge konnte nicht konvertiert werden: "+err.Error())
		}
		amount = a
	}
	be, err := h.loadOrder(he, order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}
	ex, ld := h.searchLieferdienst(be.LieferDienst)
	if !ex {
		return berghandler.SendMessage(he, evt, handlerName, "Lieferdienst nicht gefunden, benutze !bestellung dienste für eine Liste")
	}
	ex = false
	var desiredArtikel Artikel
	for _, a := range ld.Artikel {
		if (strings.Compare(artikel, strings.ToLower(a.Name)) == 0) || (strings.Compare(artikel, strings.ToLower(a.Nummer)) == 0) {
			ex = true
			desiredArtikel = a
			break
		}
	}
	if !ex {
		return berghandler.SendMessage(he, evt, handlerName, "Artikel nicht gefunden, benutze !bestellung $Lieferdienst artikel für eine Liste")
	}
	desiredVersion := desiredArtikel.Versionen[0]
	if len(desiredArtikel.Versionen) > 1 {
		ex = false
		for _, v := range desiredArtikel.Versionen {
			if strings.Compare(version, strings.ToLower(v.Name)) == 0 {
				ex = true
				desiredVersion = v
				break
			}
		}
	}
	orderedby := User{evt.Sender.Localpart(), evt.Sender.String()}
	posi := Position{}
	posi.ArtikelNummer = desiredArtikel.Nummer
	posi.ArtikelName = desiredArtikel.Name
	posi.Version = desiredVersion.Name
	posi.Einzelpreis = desiredVersion.Preis
	posi.Besteller = append(posi.Besteller, orderedby)
	posi.Anzahl = amount
	posi.Kommentar = kommentar
	be.Positionen = append(be.Positionen, posi)
	be.calcTotal()
	err = he.Storage.EncodeFile(handlerName, order+".toml", storage.TOML, false, be)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Speicerhn der bestellung: "+err.Error())
	}
	return berghandler.SendMessage(he, evt, handlerName, "Artikel hinzugefügt")
}

func (h *BestellungHandler) loadOrder(he berghandler.HandlerEssentials, order string) (Bestellung, error) {
	be := Bestellung{}
	ex := he.Storage.DoesFileExist(handlerName, order+".toml", false)
	if !ex {
		return be, errors.New("Bestellung nicht vorhanden")
	}
	err := he.Storage.DecodeFile(handlerName, order+".toml", storage.TOML, false, &be)
	if err != nil {
		return be, errors.New("Fehler beim Laden der bestellung: " + err.Error())
	}
	return be, nil
}

func (h *BestellungHandler) printOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var order string
	err := berghandler.SplitAnswer(words, 1, 0, &order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	be, err := h.loadOrder(he, order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}
	msg := be.prettyFormat()
	return berghandler.SendFormattedMessage(he, evt, handlerName, msg)
}

func (h *BestellungHandler) getCallText(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var order string
	err := berghandler.SplitAnswer(words, 1, 0, &order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	be, err := h.loadOrder(he, order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}
	msg := be.getCallText()
	return berghandler.SendMessage(he, evt, handlerName, msg)
}

func (h *BestellungHandler) getTotal(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var order string
	err := berghandler.SplitAnswer(words, 1, 0, &order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	be, err := h.loadOrder(he, order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}
	msg := be.getTotal()
	return berghandler.SendFormattedMessage(he, evt, handlerName, msg)
}

func (h *BestellungHandler) printPayment(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var order, payeds string
	err := berghandler.SplitAnswer(words, 1, 1, &order, &payeds)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	var payed float64
	if payeds != "" {
		p, err := strconv.ParseFloat(payeds, 64)
		if err != nil {
			return berghandler.SendMessage(he, evt, handlerName, "Zahlung konnte nicht konvertiert werden: "+err.Error())
		}
		payed = float64(p)
	}
	be, err := h.loadOrder(he, order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}
	if payed != 0 {
		be.Payed = payed
	} else {
		be.Payed = be.Total
	}
	msg := be.getPayment()
	return berghandler.SendFormattedMessage(he, evt, handlerName, msg)
}

func (h *BestellungHandler) deletePosition(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var order, posis string
	err := berghandler.SplitAnswer(words, 2, 0, &order, &posis)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	posi, err := strconv.Atoi(posis)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Position konnte nicht konvertiert werden: "+err.Error())
	}

	be, err := h.loadOrder(he, order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}

	if (posi >= len(be.Positionen)) || (posi < 0) {
		return berghandler.SendMessage(he, evt, handlerName, "Position nicht vorhanden")
	}
	if (!be.isCreator(evt.Sender.String())) || (!be.Positionen[posi].isBesteller(evt.Sender.String())) {
		return berghandler.SendMessage(he, evt, handlerName, unauthorized)
	}
	be.removePosition(posi)
	be.calcTotal()
	err = he.Storage.EncodeFile(handlerName, order+".toml", storage.TOML, false, be)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Speicerhn der bestellung: "+err.Error())
	}
	return berghandler.SendMessage(he, evt, handlerName, "Artikel entfernt")
}

func (h *BestellungHandler) removeOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var order string
	err := berghandler.SplitAnswer(words, 1, 0, &order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	be, err := h.loadOrder(he, order)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}
	if !be.isCreator(evt.Sender.String()) {
		return berghandler.SendMessage(he, evt, handlerName, unauthorized)
	}
	ex := he.Storage.DoesFileExist(handlerName, order+".toml", false)
	if !ex {
		return berghandler.SendMessage(he, evt, handlerName, "Bestellung nicht vorhanden")
	}
	err = he.Storage.DeleteFile(handlerName, order+".toml", false)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Error deleting file: "+err.Error())
	}
	return berghandler.SendMessage(he, evt, handlerName, "Bestellung geschlossen")
}

func (h *BestellungHandler) addStrichliste(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var username string
	err := berghandler.SplitAnswer(words, 1, 0, &username)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	var si strichlistenInfo
	err = he.Storage.DecodeFile(handlerName, "strichliste.toml", storage.TOML, true, &si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Strichlisten Info: "+err.Error())
	}
	if si.Link == nil {
		si.Link = make(map[string]string)
	}
	si.Link[evt.Sender.String()] = username
	err = he.Storage.EncodeFile(handlerName, "strichliste.toml", storage.TOML, true, si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim speichern der Strichlisten Info: "+err.Error())
	}
	return berghandler.SendMessage(he, evt, handlerName, "Link hinzugefügt")
}

func (h *BestellungHandler) removeStrichliste(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	var si strichlistenInfo
	err := he.Storage.DecodeFile(handlerName, "strichliste.toml", storage.TOML, true, &si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Strichlisten Info: "+err.Error())
	}
	if si.Link == nil {
		si.Link = make(map[string]string)
	}
	delete(si.Link, evt.Sender.String())
	err = he.Storage.EncodeFile(handlerName, "strichliste.toml", storage.TOML, true, si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim speichern der Strichlisten Info: "+err.Error())
	}
	return berghandler.SendMessage(he, evt, handlerName, "Link entfernt")
}
