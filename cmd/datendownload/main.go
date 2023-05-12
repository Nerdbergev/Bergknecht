package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Nerdbergev/Bergknecht/pkg/handlers/bestellungHandler"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pelletier/go-toml"
)

type restaurants struct {
	Rs struct {
		Rt []restaurant `json:"rt"`
	} `json:"rs"`
}

type restaurant struct {
	ID      string `json:"id"`
	Name    string `json:"nm"`
	Bn      string `json:"bn"`
	Adresse struct {
		Strasse   string `json:"st"`
		Stadt     string `json:"ci"`
		Latitude  string `json:"lt"`
		Longitude string `json:"ln"`
	} `json:"ad"`
	Bd string `json:"bd"`
}

type speisekarte struct {
	Rd Rd `json:"rd"`
}

type Rd struct {
	Plz  string `json:"bn"`
	Rci  string
	Name string `json:"nm"`
	Ddf  string
	Op   string
	// Smid <nil>
	Ds                     float64
	HeaderImgURL           string `json:"mh"`
	ChangeTime             string `json:"ct"`
	Wd                     string
	Tr                     string
	ProfileType            string `json:"pty"`
	Pro                    float64
	UrlSlug                string `json:"murl"`
	Rte                    float64
	Ck                     string
	Ac                     string
	Ri                     string
	Ply                    float64
	CloudinaryHeaderImgUrl string `json:"cloudinaryHeader"`
	Pne                    string
	Sco                    string
	Ce                     float64
	Oo                     Oo `json:"oo"`
	Dd                     Dd `json:"dd"`
	Dm                     map[string]interface{}
	Pd                     map[string]interface{} // don't know...
	Pt                     map[string]interface{} // don't know...
	Menu                   Menu                   `json:"mc"` //
	Rt                     map[string]interface{}
	Dt                     map[string]interface{}
	Dc                     map[string]interface{}
	Ad                     map[string]interface{}
	PhoneNumbers           map[string]interface{} `json:"Tel"`
	Pm                     map[string]interface{}
	Lgl                    map[string]interface{}
}

type Oo struct {
	Rv             string
	Bd             string
	Cim            float64
	Eba            bool
	LogoUrl        string `json:"lu"`
	CloudinaryLogo string
	Nt             string
	Slogan         string `json:"sl"`
	Rvd            string // revision? version? something like that
	Ft             string
}

type Dd struct {
	Da []DeliveryAria `json:"da"`
}

type DeliveryAria struct {
	PostCodes PostCodes `json:"pc"`
	Ma        string
	Costs     []Cost `json:"co"`
}

type PostCodes struct {
	Codes []string `json:"pp"`
}

type Cost struct {
	Fr string `json:"fr"` // from?
	To string `json:"to"` // to?
	Ct string `json:"ct"` // cost?
}

// Delivery info?
type Dm struct {
	Ah string
	Dl map[string]interface{} // contains min/max eta and more
	Pu map[string]interface{}
}

type Menu struct {
	Categories Categories `json:"cs"` // Categories?
}

type Categories struct {
	Items []Category `json:"ct"`
}

type Category struct {
	Id                      string `json:"id"`
	Name                    string `json:"nm"`
	Ds                      string
	CategoryImageUrl        string `json:"cti"`
	CloudinaryChainImageUrl string `json:"cloudinaryChain"`
	Ot                      []interface{}
	Products                Products `json:"ps"`
}

type Products struct {
	Items PrC `json:"pr"`
}

type PrC []Product

func (p *PrC) UnmarshalJSON(data []byte) error {
	if data[0] == 91 {
		var prs []Product
		_ = json.Unmarshal(data, &prs)
		*p = prs
	} else if data[0] == 123 {
		var m map[string]Product
		_ = json.Unmarshal(data, &m)

		prs := make([]Product, 0, len(m))
		for _, v := range m {
			prs = append(prs, v)
		}
		*p = prs
	}
	return nil
}

type Fai struct {
	Add []interface{}
	Xtr []interface{}
	Nut string
	All []interface{}
}
type Product struct {
	Id                string `json:"id"`
	Name              string `json:"nm"`
	Description       string `json:"ds"` // guessed
	Ah                string
	Price             string `json:"pc"` // guessed
	Tc                string
	Pu                string
	CloudinaryProduct string
	Fai               Fai `json:"fai"`
	Xfm               float64
	Extras            ExtraMenus `json:"ss,omitempty"`
	Sizes             Sizes      `json:"sz"`
}

type Sizes struct {
	Items []Product `json:"pr"`
}

type ExtraMenus struct {
	Items []ExtrasMenu `json:"sd"`
}

type ExtrasMenu struct {
	Name    string  `json:"nm"`
	Options Options `json:"cc"`
	Tp      string
}

type Options struct {
	Items []Product `json:"ch"`
}

type By func(r1, r2 *restaurant) bool

func (by By) Sort(restaurants []restaurant) {
	ps := &restaurantSorter{
		restaurants: restaurants,
		by:          by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

type restaurantSorter struct {
	restaurants []restaurant
	by          func(p1, p2 *restaurant) bool // Closure used in the Less method.
}

func (s *restaurantSorter) Len() int {
	return len(s.restaurants)
}

func (s *restaurantSorter) Swap(i, j int) {
	s.restaurants[i], s.restaurants[j] = s.restaurants[j], s.restaurants[i]
}

func (s *restaurantSorter) Less(i, j int) bool {
	return s.by(&s.restaurants[i], &s.restaurants[j])
}

type tomlFile struct {
	Lieferdienste []bestellungHandler.LieferDienst
}

func printRestaurants(restaurants []restaurant) {
	name := func(r1, r2 *restaurant) bool {
		return r1.Name < r2.Name
	}
	By(name).Sort(restaurants)
	t := table.NewWriter()
	t.SetStyle(table.StyleColoredDark)
	t.AppendHeader(table.Row{"ID", "Name", "Adresse"})
	for _, r := range restaurants {
		t.AppendRow(table.Row{r.ID, r.Name + " " + r.Bn, r.Adresse.Strasse + " " + r.Adresse.Stadt})
	}
	fmt.Println(t.Render())
}

func getNewHTTPRequest(in io.Reader) (*http.Request, error) {
	var req *http.Request
	var err error
	req, err = http.NewRequest("POST", "https://de.citymeal.com/android/android.php", in)
	if err != nil {
		return req, err
	}
	return req, nil
}

func sendauthorizedHTTPRequest(parameter []string, v interface{}) error {
	parm := url.Values{"language": {"de"}, "version": {"5.7"}, "systemVersion": {"24"}, "appVersion": {"4.15.3.2"}}
	var auth string
	for i, p := range parameter {
		vn := "var" + strconv.Itoa(i+1)
		parm.Add(vn, p)
		auth = auth + p
	}
	auth = auth + "4ndro1d"
	hash := md5.Sum([]byte(auth))
	md5 := hex.EncodeToString(hash[:])
	parm.Add("var0", md5)

	req, err := getNewHTTPRequest(strings.NewReader(parm.Encode()))
	if err != nil {
		return errors.New("Error creating request: " + err.Error())
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("Error executing request: " + err.Error())
	}

	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		return errors.New("Got bad Status Code: " + strconv.Itoa(resp.StatusCode) + " Message: " + string(b))
	}

	if v != nil {
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(v)
		if err != nil {
			return errors.New("Unable to decode on given Interface: " + err.Error())
		}
	}
	return nil
}

func queryRestaurants() (restaurants, error) {
	var res restaurants
	parm := []string{"getrestaurants", "90762", "2", "49.470532", "11.003153", "de"}
	err := sendauthorizedHTTPRequest(parm, &res)
	if err != nil {
		return res, errors.New("Error querying restaurants: " + err.Error())
	}
	return res, nil
}

var rgx = regexp.MustCompile(`\[(.*?)\]`)

func seperateProductNameAndSize(product string) (string, string) {
	rs := rgx.FindStringSubmatch(product)
	if len(rs) > 1 {
		fmt.Println(rs[1])
		rs2 := rgx.ReplaceAll([]byte(product), []byte(""))
		return string(rs2), rs[1]
	} else {
		return product, "default"
	}

}

func queryMenu(r string) (bestellungHandler.LieferDienst, error) {
	var res bestellungHandler.LieferDienst
	var m speisekarte
	parm := []string{"getrestaurantdata", r, "1", "", "", "", "android-client"}
	err := sendauthorizedHTTPRequest(parm, &m)
	if err != nil {
		return res, errors.New("Error querying speisekarte: " + err.Error())
	}
	res.Name = m.Rd.Name
	res.Telefonnummer = fmt.Sprint(m.Rd.PhoneNumbers["no1"])
	for _, cat := range m.Rd.Menu.Categories.Items {
		for _, prd := range cat.Products.Items {
			prdName, version := seperateProductNameAndSize(prd.Name)
			price, err := strconv.ParseFloat(prd.Price, 64)
			if err != nil {
				log.Fatal("Couldn't convert price on Size")
			}
			art := bestellungHandler.Artikel{Name: prdName}
			art.Versionen = append(art.Versionen, bestellungHandler.Zusatz{Name: version, Preis: price})
			for _, sz := range prd.Sizes.Items {
				_, version := seperateProductNameAndSize(sz.Name)
				price, err := strconv.ParseFloat(prd.Price, 64)
				if err != nil {
					log.Fatal("Couldn't convert price on Size")
				}
				art.Versionen = append(art.Versionen, bestellungHandler.Zusatz{Name: version, Preis: price})
			}
			for _, xtrMenu := range prd.Extras.Items {
				for _, opts := range xtrMenu.Options.Items {
					price, err := strconv.ParseFloat(prd.Price, 64)
					if err != nil {
						log.Fatal("Couldn't convert price on Extra")
					}
					option := strings.Replace(opts.Name, "mit ", "", -1)
					art.Extras = append(art.Extras, bestellungHandler.Zusatz{Name: option, Preis: price})
				}
			}
			res.Artikel = append(res.Artikel, art)
		}
	}
	return res, nil
}

var showRestaurants bool
var restaurantID string

func init() {
	flag.BoolVar(&showRestaurants, "rc", false, "Set this to show close by restaurants")
	flag.StringVar(&restaurantID, "id", "", "Set this flag to download the menu of a restaurant")
}

func main() {
	flag.Parse()
	if showRestaurants {
		log.Println("Getting all close by Restaurants")
		res, err := queryRestaurants()
		if err != nil {
			log.Fatal("Error getting restaurants: ", err)
		}

		printRestaurants(res.Rs.Rt)
	} else if restaurantID != "" {
		r := csv.NewReader(strings.NewReader(restaurantID))
		ids, err := r.Read()
		if err != nil {
			log.Fatal("Error parsing ids")
		}
		var tf tomlFile
		for _, r := range ids {
			log.Println("Getting menu for", r)
			ld, err := queryMenu(r)
			if err != nil {
				log.Fatal("Error getting menu:", err.Error())
			}
			fmt.Println(ld)
			tf.Lieferdienste = append(tf.Lieferdienste, ld)
		}
		f, err := os.Create("lieferdienste.toml")
		if err != nil {
			log.Fatalf("Error creating the file: %v \n", err)
		}
		defer f.Close()
		encoder := toml.NewEncoder(f)
		err = encoder.Encode(tf)
		if err != nil {
			fmt.Printf("Error encoding the toml: %v \n", err)
		}
		f.Close()
		log.Println("Restaurants saved as toml")
	} else {
		log.Fatal("No command given, use -h or --help to get commands")
	}
}
