package bestellungHandler

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Nerdbergev/Bergknecht/pkg/berghandler"
	"github.com/Nerdbergev/Bergknecht/pkg/storage"
	"github.com/jedib0t/go-pretty/v6/table"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type BestellungHandler struct {
	Lieferdienste []LieferDienst
	subHandlers   berghandler.SubHandlers
}

func (h *BestellungHandler) Prime(he berghandler.HandlerEssentials) error {
	h.subHandlers = make(map[string]berghandler.SubHandlerSet)
	h.subHandlers["new"] = berghandler.SubHandlerSet{F: h.newOrder, H: "Erstellt eine Neue Bestellung.", U: "new $Lieferdienst", NV: 1, OV: 0}
	h.subHandlers["add"] = berghandler.SubHandlerSet{F: h.addtoOrder, H: "Hinzufügen eines Items zur Bestellung", U: "add $Bestellung $Artikel [$Version $Extras $Kommentar $Anzahl]", NV: 2, OV: 4}
	h.subHandlers["show"] = berghandler.SubHandlerSet{F: h.printOrder, H: "Anzeigen einer Bestellung", U: "show $Bestellung", NV: 1, OV: 0}
	h.subHandlers["call-text"] = berghandler.SubHandlerSet{F: h.getCallText, H: "Ausgabe einen Textes zum Anrufen", U: "call-text $Bestellung", NV: 1, OV: 0}
	h.subHandlers["print-payment"] = berghandler.SubHandlerSet{F: h.printPayment, H: "Ausgabe der Informationen wer was bezahlen muss", U: "print-payment $Bestellung [$Gezahlt]", NV: 1, OV: 1}
	h.subHandlers["get-total"] = berghandler.SubHandlerSet{F: h.getTotal, H: "Ausgabe wie viel die Bestellung kostet plus Trinkgeld Vorschläge", U: "get-total $Bestellung", NV: 1, OV: 0}
	h.subHandlers["remove"] = berghandler.SubHandlerSet{F: h.deletePosition, H: "Löscht Position aus der Bestellung", U: "remove $Bestellung $Position", NV: 2, OV: 0}
	h.subHandlers["close"] = berghandler.SubHandlerSet{F: h.removeOrder, H: "Schließt Bestellung und Löscht diese", U: "close $Bestellung", NV: 1, OV: 0}
	h.subHandlers["add-strichliste"] = berghandler.SubHandlerSet{F: h.addStrichliste, H: "Verknüpft den schreibenden Matrix account mit einem Strichlisten Benutzer", U: "add-strichliste $Benutzername", NV: 1, OV: 0}
	h.subHandlers["remove-strichliste"] = berghandler.SubHandlerSet{F: h.removeStrichliste, H: "Löscht Matrix account zu Strichlisten account verknüpfung", U: "remove-strichliste", NV: 0, OV: 0}
	h.subHandlers["process-strichliste"] = berghandler.SubHandlerSet{F: h.processStrichliste, H: "Versucht Bestellung via Strichliste abzurechenen", U: "process-strichliste $Bestellung [$Bezahlendendes-Wesen]", NV: 1, OV: 1}
	h.subHandlers["menu"] = berghandler.SubHandlerSet{F: h.showMenu, H: "Zeigt Menü eines Lieferdienstes", U: "menu $Lieferdienst", NV: 1, OV: 0}
	h.subHandlers["article"] = berghandler.SubHandlerSet{F: h.showArticle, H: "Zeigt Artikelinformationen", U: "article $Lieferdienst $Article", NV: 2, OV: 0}
	h.subHandlers["restaurants"] = berghandler.SubHandlerSet{F: h.showRestaurants, H: "Zeigt alle Lieferdienste", U: "restaurants", NV: 0, OV: 0}

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

func (h *BestellungHandler) newOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var ld string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &ld)
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
	be.Datum = time.Now()
	be.Ersteller = User{evt.Sender.Localpart(), evt.Sender.String()}
	be.LieferDienst = ld
	be.Nummer = l.Telefonnummer
	err = he.Storage.EncodeFile(handlerName, bnf, storage.TOML, false, be)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler bei erstellung der Bestellung")
	}

	return berghandler.SendMessage(he, evt, handlerName, "Neue Bestellung mit dem Name: "+bn+" erstellt")
}

func parseExtras(extras string, artikel Artikel) ([]Zusatz, error) {
	var result []Zusatz
	if extras == "" {
		return result, nil
	}
	r := csv.NewReader(strings.NewReader(extras))
	r.Comma = ','
	read, err := r.Read()
	if err != nil {
		return result, errors.New("Fehler beim lesen der Extras: " + err.Error())
	}
	for _, r := range read {
		cont := false
		for _, e := range artikel.Extras {
			if strings.Compare(strings.ToLower(r), e.Name) == 0 {
				result = append(result, e)
				cont = true
				break
			}
		}
		if !cont {
			return result, errors.New("Konnte Zusatz " + r + " nicht zuordnen")
		}
	}
	return result, nil
}

func getExtrasTotal(zusätze []Zusatz) float64 {
	result := 0.0
	for _, z := range zusätze {
		result += z.Preis
	}
	return result
}

func (h *BestellungHandler) addtoOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var order, artikel, version, extras, kommentar, anzahl string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &order, &artikel, &version, &extras, &kommentar, &anzahl)
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
		return berghandler.SendMessage(he, evt, handlerName, "Artikel nicht gefunden, benutze !bestellung article $Lieferdienst für eine Liste")
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
	desiredExtras, err := parseExtras(extras, desiredArtikel)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim parsen der extras:"+err.Error()+" Benutze !bestellung article $Lieferdienst für eine Liste")
	}

	orderedby := User{evt.Sender.Localpart(), evt.Sender.String()}
	posi := Position{}
	posi.ArtikelNummer = desiredArtikel.Nummer
	posi.ArtikelName = desiredArtikel.Name
	posi.Version = desiredVersion.Name
	posi.Extras = extras
	posi.Einzelpreis = desiredVersion.Preis + getExtrasTotal(desiredExtras)
	posi.Besteller = append(posi.Besteller, orderedby)
	posi.Anzahl = amount
	posi.Kommentar = kommentar
	be.Positionen = append(be.Positionen, posi)
	be.calcTotal()
	err = he.Storage.EncodeFile(handlerName, order+".toml", storage.TOML, false, be)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Speichern der bestellung: "+err.Error())
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

func (h *BestellungHandler) printOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var order string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &order)
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

func (h *BestellungHandler) getCallText(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var order string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &order)
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

func (h *BestellungHandler) getTotal(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var order string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &order)
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

func (h *BestellungHandler) printPayment(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var order, payeds string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &order, &payeds)
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
	err = he.Storage.EncodeFile(handlerName, order+".toml", storage.TOML, false, be)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Speichern der bestellung: "+err.Error())
	}
	msg := be.getPayment()
	return berghandler.SendFormattedMessage(he, evt, handlerName, msg)
}

func (h *BestellungHandler) deletePosition(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var order, posis string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &order, &posis)
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
	if (!be.isCreator(evt.Sender.String())) && (!be.Positionen[posi].isBesteller(evt.Sender.String())) {
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

func (h *BestellungHandler) removeOrder(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var order string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &order)
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

func execHTTPRequest(URL string, method string, in io.Reader, v interface{}) error {
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*5)
	defer cncl()

	req, err := http.NewRequestWithContext(ctx, method, URL, in)
	if err != nil {
		return errors.New("Fehler beim Request erstellen: " + err.Error())
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("Fehler beim Request ausführen: " + err.Error())
	}

	if v != nil {
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(v)
		if err != nil {
			data, _ := io.ReadAll(resp.Body)
			fmt.Println(string(data))
			return errors.New("keine decodierung möglich: " + err.Error())
		}
	}
	return nil
}

func getStrichlistenID(address, name string) (int, error) {

	url := fmt.Sprintf(address+"/api/user/search?query=%v", name)

	var userResponse siUserResponse
	err := execHTTPRequest(url, http.MethodGet, nil, &userResponse)
	if err != nil {
		return -1, err
	}

	var siUser siUser

	switch userResponse.Count {
	case 0:
		return -1, errors.New("keinen Strichlisten Benutzer gefunden")
	case 1:
		siUser = userResponse.SiUsers[0]
	default:
		found := false
		for _, u := range userResponse.SiUsers {
			if strings.Compare(strings.ToLower(name), strings.ToLower(u.Name)) == 0 {
				found = true
				siUser = u
				break
			}
		}
		if !found {
			return -1, errors.New("keine exakte übereinstimmung gefunden. Bitte Name prüfen")
		}
	}
	if siUser.IsDisabled {
		return -1, errors.New("benutzer disabled")
	}
	return siUser.ID, nil
}

func (h *BestellungHandler) addStrichliste(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var username string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &username)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	var si strichlistenInfo
	err = he.Storage.DecodeFile(handlerName, "strichliste.toml", storage.TOML, true, &si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Strichlisten Info: "+err.Error())
	}
	if si.Link == nil {
		si.Link = make(map[string]int)
	}
	id, err := getStrichlistenID(si.Address, username)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim finden des Strichlisten Users: "+err.Error())
	}
	si.Link[evt.Sender.String()] = id
	err = he.Storage.EncodeFile(handlerName, "strichliste.toml", storage.TOML, true, si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim speichern der Strichlisten Info: "+err.Error())
	}
	return berghandler.SendMessage(he, evt, handlerName, "Link hinzugefügt")
}

func (h *BestellungHandler) removeStrichliste(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var si strichlistenInfo
	err := he.Storage.DecodeFile(handlerName, "strichliste.toml", storage.TOML, true, &si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Strichlisten Info: "+err.Error())
	}
	if si.Link == nil {
		si.Link = make(map[string]int)
	}
	delete(si.Link, evt.Sender.String())
	err = he.Storage.EncodeFile(handlerName, "strichliste.toml", storage.TOML, true, si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim speichern der Strichlisten Info: "+err.Error())
	}
	return berghandler.SendMessage(he, evt, handlerName, "Link entfernt")
}

func writePaymentResult(wg *sync.WaitGroup, ses *safeExecStatus, payee User, result string) {
	ses.mu.Lock()
	ses.es[payee] = result
	ses.mu.Unlock()
	wg.Done()
}

func (h *BestellungHandler) doPayment(payer int, p paymentInfo, comment string, si strichlistenInfo, wg *sync.WaitGroup, ses *safeExecStatus) {
	siID, ex := si.Link[p.Payee.MatrixID]
	if !ex {
		writePaymentResult(wg, ses, p.Payee, "Keinen Strichlisten Benutzer gefunden")
		return
	}

	if payer == siID {
		writePaymentResult(wg, ses, p.Payee, "Benutzer hat bei Lieferdienst bezahlt")
		return
	}

	url := fmt.Sprintf(si.Address+"/api/user/%v", siID)

	var userResponse siUser
	err := execHTTPRequest(url, http.MethodGet, nil, &userResponse)
	if err != nil {
		writePaymentResult(wg, ses, p.Payee, "Fehler beim User Request:"+err.Error())
		return
	}

	if userResponse.IsDisabled {
		writePaymentResult(wg, ses, p.Payee, "Benutzer disabled")
		return
	}

	t := siTransaction{Amount: int(math.Ceil(p.Amount * 100)), Comment: comment, RecipientID: payer}
	to := siTransactionOJ{}

	b := new(bytes.Buffer)
	encoder := json.NewEncoder(b)
	encoder.Encode(t)

	url = fmt.Sprintf(si.Address+"/api/user/%v/transaction", siID)
	err = execHTTPRequest(url, http.MethodPost, b, &to)
	if err != nil {
		writePaymentResult(wg, ses, p.Payee, "Fehler beim Transaction Request:"+err.Error())
		return
	}

	writePaymentResult(wg, ses, p.Payee, fmt.Sprintf("Transaction mit der ID %v angelegt", to.ID))
}

func (h *BestellungHandler) processStrichliste(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var order string
	var payer string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &order, &payer)
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
	if be.Payed == 0 {
		return berghandler.SendMessage(he, evt, handlerName, "Bestellung hat noch keinen gezahlten Geldwert")
	}
	var si strichlistenInfo
	err = he.Storage.DecodeFile(handlerName, "strichliste.toml", storage.TOML, true, &si)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, "Fehler beim Laden der Strichlisten Info: "+err.Error())
	}
	if payer == "" {
		payer = be.Ersteller.MatrixID
	}
	siPayer, ex := si.Link[payer]
	if !ex {
		return berghandler.SendMessage(he, evt, handlerName, "Zahlender hat keine Strichliste verlinkt")
	}
	var wg sync.WaitGroup
	ses := safeExecStatus{es: make(map[User]string)}
	pi, _ := be.calcPayment()
	ti := be.Datum.Format(time.RFC3339)
	c := fmt.Sprintf("Bestellung bei %v am %v", be.LieferDienst, ti)
	wg.Add(len(pi))
	for _, p := range pi {
		go h.doPayment(siPayer, p, c, si, &wg, &ses)
	}
	wg.Wait()
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.SetTitle("Bestellung bei " + be.LieferDienst + " Strichlisten Abrechnung")
	t.AppendHeader(table.Row{"Name", "Ergebnis"})
	for p, r := range ses.es {
		t.AppendRow(table.Row{p.MatrixID, r})
	}
	return berghandler.SendFormattedMessage(he, evt, handlerName, t.RenderHTML())
}

func (h *BestellungHandler) showMenu(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var ld string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &ld)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	found, l := h.searchLieferdienst(ld)
	if !found {
		return berghandler.SendMessage(he, evt, handlerName, "Lieferdienst nicht gefunden, benutze !bestellung dienste für eine Liste")
	}
	return berghandler.SendFormattedMessage(he, evt, handlerName, l.prettyFormat())
}

func (h *BestellungHandler) showArticle(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	var ld, artikel string
	err := berghandler.SplitAnswer(words, neededVariables, optionalVariables, &ld, &artikel)
	if err != nil {
		return berghandler.SendMessage(he, evt, handlerName, fmt.Sprintf(berghandler.WrongArguments, berghandler.CommandPrefix+command)+" "+err.Error())
	}
	found, l := h.searchLieferdienst(ld)
	if !found {
		return berghandler.SendMessage(he, evt, handlerName, "Lieferdienst nicht gefunden, benutze !bestellung dienste für eine Liste")
	}

	ex := false
	var desiredArtikel Artikel
	for _, a := range l.Artikel {
		if (strings.Compare(artikel, strings.ToLower(a.Name)) == 0) || (strings.Compare(artikel, strings.ToLower(a.Nummer)) == 0) {
			ex = true
			desiredArtikel = a
			break
		}
	}
	if !ex {
		return berghandler.SendMessage(he, evt, handlerName, "Artikel nicht gefunden, benutze !bestellung menu $Lieferdienst für eine Liste")
	}

	return berghandler.SendFormattedMessage(he, evt, handlerName, desiredArtikel.prettyFormat())
}

func (h *BestellungHandler) prettyFormatRestaurants() string {
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.SetTitle("Lieferdienste")
	t.AppendHeader(table.Row{"#", "Name", "Telefonnummer"})
	for i, l := range h.Lieferdienste {
		t.AppendRow(table.Row{i, l.Name, l.Telefonnummer})
	}
	return t.RenderHTML()
}

func (h *BestellungHandler) showRestaurants(he berghandler.HandlerEssentials, evt *event.Event, words []string, neededVariables, optionalVariables int) bool {
	return berghandler.SendFormattedMessage(he, evt, handlerName, h.prettyFormatRestaurants())
}
