package bestellungHandler

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/Nerdbergev/Bergknecht/pkg/berghandler"
	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"github.com/jedib0t/go-pretty/v6/table"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

const wrongArguments = "Falsche Anzahl an Argumenten, benutze !bestellung help für Hilfe."

var handlerName = "BestellungHandler"

type User struct {
	DisplayName string
	MatrixID    string
}

type BestellungHandler struct {
	Lieferdienste []LieferDienst
}

type LieferDienst struct {
	Name          string
	Telefonnummer string
	Artikel       []Artikel
}

type Artikel struct {
	Nummer    string
	Name      string
	Versionen []Version
}

type Version struct {
	Name  string
	Preis float32
}

type Bestellung struct {
	Ersteller    User
	LieferDienst string
	Nummer       string
	Positionen   []Position
}

func (b *Bestellung) prettyFormat() string {
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.SetTitle("Bestellung bei " + b.LieferDienst)
	t.AppendHeader(table.Row{"#", "Nummer", "Name", "Version", "Anzahl", "Kommentar", "Besteller"})
	for i, p := range b.Positionen {
		t.AppendRow(table.Row{i, p.ArtikelNummer, p.ArtikelName, p.Version, p.Anzahl, p.Kommentar, p.Besteller[0].DisplayName})
	}
	return t.RenderHTML()
}

func (b *Bestellung) getCallText() string {
	var newbestellung Bestellung
	for _, p := range b.Positionen {
		added := false
		for i, p2 := range newbestellung.Positionen {
			if p.isSameAs(p2) {
				newbestellung.Positionen[i].Anzahl += p.Anzahl
				newbestellung.Positionen[i].Besteller = append(newbestellung.Positionen[i].Besteller, p.Besteller...)
				added = true
				break
			}
		}
		if !added {
			newbestellung.Positionen = append(newbestellung.Positionen, p)
		}
	}
	result := "Lieferdienst: " + b.LieferDienst + "\n"
	result = result + "Telefonnummer: " + b.Nummer + "\n\n"
	result = result + "Hallo Nord mein Name ich würde gerne Bestellen und zwar: \n"
	for _, p := range newbestellung.Positionen {
		if p.ArtikelNummer != "" {
			result = result + fmt.Sprintf("%v mal die Nummer %v %v in %v %v\n", p.Anzahl, p.ArtikelNummer, p.ArtikelName, p.Version, p.Kommentar)
		} else {
			result = result + fmt.Sprintf("%v mal %v in %v %v\n", p.Anzahl, p.ArtikelName, p.Version, p.Kommentar)
		}
	}
	return result
}

type Position struct {
	ArtikelNummer string
	ArtikelName   string
	Version       string
	Einzelpreis   float32
	Anzahl        int
	Besteller     []User
	Kommentar     string
}

func (p *Position) isSameAs(p2 Position) bool {
	result := true
	result = result && (p.ArtikelName == p2.ArtikelName)
	result = result && (p.ArtikelNummer == p2.ArtikelNummer)
	result = result && (p.Version == p2.Version)
	result = result && (p.Kommentar == p2.Kommentar)
	return result
}

func (h *BestellungHandler) LoadData(he berghandler.HandlerEssentials) error {
	return he.Storage.DecodeFile(handlerName, "lieferdienste.toml", storage.TOML, true, h)
}

func (h *BestellungHandler) GetName() string {
	return handlerName
}

func (h *BestellungHandler) Handle(he berghandler.HandlerEssentials, source mautrix.EventSource, evt *event.Event) bool {
	result := false
	if berghandler.IsMessagewithPrefix(evt, "bestellung") {
		m := evt.Content.AsMessage()
		words, err := berghandler.StripPrefixandGetContent(m.Body, "bestellung")
		if err != nil {
			return berghandler.SendMessage(he, evt, handlerName, "Fehler bei decodieren der Nachricht: "+err.Error())
		}
		if len(words) < 2 {
			return berghandler.SendMessage(he, evt, handlerName, wrongArguments)
		}
		cmd := strings.ToLower(words[0])
		newwords := berghandler.RemoveWord(words, 0)
		switch cmd {
		case "new":
			result = h.newOrder(he, evt, newwords)
		case "add":
			result = h.addtoOrder(he, evt, newwords)
		case "show":
			result = h.printOrder(he, evt, newwords)
		case "call-text":
			result = h.getCallText(he, evt, newwords)
		default:
			return berghandler.SendMessage(he, evt, handlerName, "Kein valides Argument, benutze !bestellung help für Hilfe")
		}
	}
	return result
}

func getRandomWord(slice []string) string {
	return slice[rand.Intn(len(slice))]
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
		return berghandler.SendMessage(he, evt, handlerName, wrongArguments+" "+err.Error())
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
		return berghandler.SendMessage(he, evt, handlerName, wrongArguments+" "+err.Error())
	}
	ex := he.Storage.DoesFileExist(handlerName, order+".toml", false)
	if !ex {
		return berghandler.SendMessage(he, evt, handlerName, "Bestellung nicht vorhanden")
	}
	amount := 1
	if len(words) >= 6 {
		a, err := strconv.Atoi(anzahl)
		if err != nil {
			return berghandler.SendMessage(he, evt, handlerName, "Menge konnte nicht konvertiert werden: "+err.Error())
		}
		amount = a
	}
	be := Bestellung{}
	err = he.Storage.DecodeFile(handlerName, order+".toml", storage.TOML, false, &be)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der bestellung: "+err.Error())
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
	err = he.Storage.EncodeFile(handlerName, order+".toml", storage.TOML, false, be)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Speicerhn der bestellung: "+err.Error())
	}
	return berghandler.SendMessage(he, evt, handlerName, "Artikel hinzugefügt")
}

func (h *BestellungHandler) loadOrder(he berghandler.HandlerEssentials, words []string) (Bestellung, error) {
	var order string
	be := Bestellung{}
	err := berghandler.SplitAnswer(words, 1, 0, &order)
	if err != nil {
		return be, errors.New(wrongArguments + " " + err.Error())
	}
	ex := he.Storage.DoesFileExist(handlerName, order+".toml", false)
	if !ex {
		return be, errors.New("Bestellung nicht vorhanden")
	}
	err = he.Storage.DecodeFile(handlerName, order+".toml", storage.TOML, false, &be)
	if err != nil {
		return be, errors.New("Fehler beim Laden der bestellung: " + err.Error())
	}
	return be, nil
}

func (h *BestellungHandler) printOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	be, err := h.loadOrder(he, words)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}
	msg := be.prettyFormat()
	return berghandler.SendFormattedMessage(he, evt, handlerName, msg)
}

func (h *BestellungHandler) getCallText(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	be, err := h.loadOrder(he, words)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Bestellung: "+err.Error())
	}
	msg := be.getCallText()
	return berghandler.SendMessage(he, evt, handlerName, msg)
}

func (h *BestellungHandler) printPayment(he berghandler.HandlerEssentials, evt *event.Event, words []string) bool {
	return true
}
