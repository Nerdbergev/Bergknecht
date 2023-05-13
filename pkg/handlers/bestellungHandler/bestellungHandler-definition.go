package bestellungHandler

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
)

const handlerName = "BestellungHandler"
const command = "bestellung"

const unauthorized = "Nur der Bestellungs ersteller kann dieses Kommando ausführen"

func getRandomWord(slice []string) string {
	return slice[rand.Intn(len(slice))]
}

type User struct {
	DisplayName string
	MatrixID    string
}

type LieferDienst struct {
	Name          string
	ID            string
	Telefonnummer string
	Artikel       []Artikel
}

func (ld *LieferDienst) prettyFormat() string {
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.SetTitle("Menü von " + ld.Name)
	t.AppendHeader(table.Row{"#", "Artikel Nummer", "Artikel Name"})
	for i, a := range ld.Artikel {
		t.AppendRow(table.Row{i, a.Nummer, a.Name})
	}
	return t.RenderHTML()
}

type Artikel struct {
	Nummer    string
	Name      string
	ID        string
	Versionen []Zusatz
	Extras    []Zusatz
}

func (a *Artikel) prettyFormat() string {
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.SetTitle("Versionen und Extras von " + a.Name)
	t.AppendHeader(table.Row{"#", "Version/Extra", "Preis"})
	t.AppendRow(table.Row{"", "Versionen"})
	for i, v := range a.Versionen {
		t.AppendRow(table.Row{i, v.Name, v.Preis})
	}
	t.AppendRow(table.Row{"", "Extras"})
	for i, e := range a.Extras {
		t.AppendRow(table.Row{i, e.Name, e.Preis})
	}
	return t.RenderHTML()
}

type Zusatz struct {
	Name  string
	ID    string
	Preis float64
}

type Bestellung struct {
	Ersteller    User
	Datum        time.Time
	LieferDienst string
	Nummer       string
	Positionen   []Position
	Total        float64
	Payed        float64
}

func (b *Bestellung) removePosition(i int) {
	if (i > -1) && (i < len(b.Positionen)) {
		b.Positionen = append(b.Positionen[:i], b.Positionen[i+1:]...)
	}
}

func (b *Bestellung) prettyFormat() string {
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.SetTitle("Bestellung bei " + b.LieferDienst)
	t.AppendHeader(table.Row{"#", "Nummer", "Name", "Version", "Anzahl", "Extras", "Kommentar", "Besteller"})
	for i, p := range b.Positionen {
		t.AppendRow(table.Row{i, p.ArtikelNummer, p.ArtikelName, p.Version, p.Anzahl, p.Extras, p.Kommentar, p.Besteller[0].DisplayName})
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
			result = result + fmt.Sprintf("%v mal die Nummer %v %v in %v mit %v %v\n", p.Anzahl, p.ArtikelNummer, p.ArtikelName, p.Version, p.Extras, p.Kommentar)
		} else {
			result = result + fmt.Sprintf("%v mal %v in %v mit %v %v\n", p.Anzahl, p.ArtikelName, p.Version, p.Extras, p.Kommentar)
		}
	}
	return result
}

func (b *Bestellung) calcTotal() {
	var t float64
	for _, p := range b.Positionen {
		t += p.getTotal()
	}
	b.Total = t
}

func (b *Bestellung) calcTips() (float64, float64, float64, float64) {
	var up, five, ten, twenty float64
	up = math.Ceil(b.Total)
	five = math.Floor(b.Total*1.05 + 0.5)
	ten = math.Floor(b.Total*1.10 + 0.5)
	twenty = math.Floor(b.Total*1.20 + 0.5)
	return up, five, ten, twenty
}

func (b *Bestellung) getTotal() string {
	up, five, ten, twenty := b.calcTips()
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.SetTitle("Bestellung bei " + b.LieferDienst + " zu Zahlen")
	t.AppendRow(table.Row{"Total", b.Total})
	t.AppendRow(table.Row{"Aufgerundet", up})
	t.AppendRow(table.Row{"5% Trinkgeld", five})
	t.AppendRow(table.Row{"10% Trinkgeld", ten})
	t.AppendRow(table.Row{"15% Trinkgeld", twenty})
	return t.RenderHTML()
}

type paymentInfo struct {
	Payee  User
	Amount float64
}

func (b *Bestellung) calcPayment() ([]paymentInfo, float64) {
	var result []paymentInfo
	off := (100 / b.Total * b.Payed) / 100
	for _, p := range b.Positionen {
		payment := p.getTotal() * off
		found := false
		for i, pi := range result {
			if pi.Payee == p.Besteller[0] {
				result[i].Amount += payment
				found = true
				break
			}
		}
		if !found {
			result = append(result, paymentInfo{Payee: p.Besteller[0], Amount: payment})
		}
	}
	return result, off
}

func (b *Bestellung) getPayment() string {
	pi, off := b.calcPayment()
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.SetTitle("Bestellung bei " + b.LieferDienst + " Besteller Schulden")
	t.AppendHeader(table.Row{"Rabatt", off})
	t.AppendHeader(table.Row{"Name", "Schulden"})
	for _, p := range pi {
		t.AppendRow(table.Row{p.Payee.DisplayName, p.Amount})
	}
	return t.RenderHTML()
}

func (b *Bestellung) isCreator(id string) bool {
	return strings.Compare(b.Ersteller.MatrixID, id) == 0
}

type Position struct {
	ArtikelNummer string
	ArtikelName   string
	Version       string
	Extras        string
	Einzelpreis   float64
	Anzahl        int
	Besteller     []User
	Kommentar     string
}

func (p *Position) isSameAs(p2 Position) bool {
	result := true
	result = result && (p.ArtikelName == p2.ArtikelName)
	result = result && (p.ArtikelNummer == p2.ArtikelNummer)
	result = result && (p.Version == p2.Version)
	result = result && (p.Extras == p2.Extras)
	result = result && (p.Kommentar == p2.Kommentar)
	return result
}

func (p *Position) getTotal() float64 {
	return float64(p.Anzahl) * p.Einzelpreis
}

func (p *Position) isBesteller(id string) bool {
	return strings.Compare(p.Besteller[0].MatrixID, id) == 0
}

type strichlistenInfo struct {
	Address string
	Link    map[string]int
}

type safeExecStatus struct {
	mu sync.RWMutex
	es map[User]string
}

type siUser struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	IsDisabled bool   `json:"isDisabled"`
}

type siUserResponse struct {
	Count   int      `json:"count"`
	SiUsers []siUser `json:"users"`
}

type siTransaction struct {
	Amount      int    `json:"amount"`
	RecipientID int    `json:"recipientId"`
	Comment     string `json:"comment"`
}

type siTransactionOJ struct {
	ID int `json:"id"`
}
