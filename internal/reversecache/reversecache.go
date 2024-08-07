package reversecache

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fruktkartan/fruktsam/internal/types"
)

// TODO locking for concurrent access?

func Add(p types.Pos) {
	getInstance().add(p)
}

func Has(p types.Pos) bool {
	return getInstance().has(p)
}

func Save() error {
	return getInstance().save()
}

func FormatAddress(p types.Pos) string {
	return getInstance().formatAddress(p)
}

const reversefile = "reversecache"

var (
	r    *ReverseCache
	once sync.Once
)

type httpError struct {
	statusCode int
}

func (e httpError) Error() string {
	return fmt.Sprintf("HTTP StatusCode: %d", e.statusCode)
}

func reverse(p types.Pos) ([]byte, error) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "https://nominatim.openstreetmap.org/reverse", nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("user-agent", "fruktsam (https://github.com/fruktkartan/fruktsam)")
	req.Header.Add("accept-language", "sv,en-US,en")
	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("lat", fmt.Sprintf("%g", p.Lat))
	q.Add("lon", fmt.Sprintf("%g", p.Lon))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpError{statusCode: resp.StatusCode}
	}
	return body, nil
}

type ReverseCache struct {
	Table apiResults // exported for gob
	dirty bool
}

type apiResults map[types.Pos][]byte

func getInstance() *ReverseCache {
	once.Do(func() {
		r = &ReverseCache{}
		r.Table = make(apiResults)
		if err := r.load(); err != nil {
			log.Fatal(err)
		}
	})

	return r
}

func (r *ReverseCache) save() error {
	if !r.dirty {
		fmt.Printf("Reversecache not modified, not saving\n")
		return nil
	}
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	if err := enc.Encode(r); err != nil {
		return err
	}

	f, err := os.OpenFile(reversefile, os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(b.Bytes()); err != nil {
		return err
	}
	return nil
}

func (r *ReverseCache) load() error {
	if len(r.Table) > 0 {
		return fmt.Errorf("reversecache not empty, refusing to load from file")
	}

	f, err := os.Open(reversefile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		fmt.Printf("Initialized empty Reversecache.\n")
		return nil
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	if err := dec.Decode(&r); err != nil {
		return err
	}
	fmt.Printf("Reversecache loaded with %d entries from file\n", len(r.Table))
	return nil
}

func (r *ReverseCache) has(p types.Pos) bool {
	_, ok := r.Table[p]
	return ok
}

func (r *ReverseCache) add(p types.Pos) {
	if !r.has(p) {
		jsonbytes, err := reverse(p)
		if err != nil {
			var httpErr httpError
			if errors.As(err, &httpErr) {
				fmt.Printf("%v: %s (nothing added)\n", p, err)
			} else {
				fmt.Printf("%v: %s (added nil)\n", p, err)
				// We store in reversecache even if we got nothing
				r.Table[p] = nil
			}
		} else {
			r.Table[p] = jsonbytes
		}
		r.dirty = true
	}
}

// func (r *ReverseCache) del(p types.Pos) {
// 	if !r.has(p) {
// 		return
// 	}
// 	delete(r.Table, p)
// 	r.dirty = true
// }

func (r *ReverseCache) formatAddress(p types.Pos) string {
	if !r.has(p) {
		return "?????"
	}
	if r.Table[p] == nil {
		log.Printf("%v: reverse in cache is nil", p)
		return "????"
	}

	root := osm{}
	err := json.Unmarshal(r.Table[p], &root)
	if err != nil {
		log.Printf("%v: %s\n", p, err)
		return "???"
	}

	a := root.Address
	if a == (address{}) {
		return "??"
	}

	items := []string{}

	muni := ""
	suburb := ""

	if a.street() != "" {
		items = append(items, a.street())
	} else {
		suburb = a.Suburb
	}

	loc := a.locality()
	switch loc {
	case "Malmö", "Tätort Göteborg", "Stockholm":
		suburb = a.Suburb
	case "":
		// details to empty locality
		suburb = a.Suburb
		muni = a.Municipality
	}

	if suburb != "" {
		items = append(items, suburb)
	}
	if loc != "" {
		if loc == "Tätort Göteborg" {
			loc = "Göteborg"
		}
		items = append(items, loc)
	}

	// detail to short address
	if len(items) < 2 {
		muni = a.Municipality
	}

	if muni != "" {
		items = append(items, muni)
	}

	if len(items) == 0 {
		return "?"
	}

	s := strings.Join(items, ", ")

	if a.CountryCode != "se" {
		cc := strings.ToUpper(a.CountryCode)
		if cc == "" {
			cc = "??"
		}
		s += " (" + cc + ")"
	}

	return s
}

type osm struct {
	DisplayName string  `json:"display_name"`
	Lat         string  `json:"lat"`
	Lon         string  `json:"lon"`
	Error       string  `json:"error"`
	Address     address `json:"address"`
}
type address struct {
	IsolatedDwelling string `json:"isolated_dwelling"`
	Neighbourhood    string `json:"neighbourhood"`
	Quarter          string `json:"quarter"`
	HouseNumber      string `json:"house_number"`
	Road             string `json:"road"`
	Pedestrian       string `json:"pedestrian"`
	Footway          string `json:"footway"`
	Cycleway         string `json:"cycleway"`
	Highway          string `json:"highway"`
	Path             string `json:"path"`
	Suburb           string `json:"suburb"`
	City             string `json:"city"`
	Town             string `json:"town"`
	Village          string `json:"village"`
	Hamlet           string `json:"hamlet"`
	Municipality     string `json:"municipality"`
	County           string `json:"county"`
	Country          string `json:"country"`
	CountryCode      string `json:"country_code"`
	State            string `json:"state"`
	StateDistrict    string `json:"state_district"`
	Postcode         string `json:"postcode"`
}

// func (a *address) dump(exclude []string) string {
// 	refval := reflect.ValueOf(a)
// 	vals := []string{}
// 	contains := func(ss []string, s string) bool {
// 		for _, v := range ss {
// 			if v == s {
// 				return true
// 			}
// 		}
// 		return false
// 	}
// 	for i := 0; i < refval.NumField(); i++ {
// 		fname := refval.Type().Field(i).Name
// 		switch fname {
// 		case "HouseNumber", "County", "Country", "CountryCode", "State", "StateDistrict", "Postcode":
// 		default:
// 			fval := refval.Field(i).String()
// 			if fval != "" && !contains(exclude, fval) {
// 				vals = append(vals, fname+":"+fval)
// 			}
// 		}
// 	}
// 	return strings.Join(vals, ",")
// }

func (a *address) locality() string {
	var locality string
	switch {
	case a.City != "":
		locality = a.City
	case a.Town != "":
		locality = a.Town
	case a.Village != "":
		locality = a.Village
	case a.Hamlet != "":
		locality = a.Hamlet
	}
	return locality
}

func (a *address) street() string {
	var street string
	switch {
	case a.Road != "":
		street = a.Road
	case a.Pedestrian != "":
		street = a.Pedestrian
	case a.Path != "":
		street = a.Path
	case a.Cycleway != "":
		street = a.Cycleway
	case a.Footway != "":
		street = a.Footway
	case a.Highway != "":
		street = a.Highway
	case a.Neighbourhood != "":
		street = a.Neighbourhood
	case a.Quarter != "":
		street = a.Quarter
	case a.IsolatedDwelling != "":
		street = a.IsolatedDwelling
	}
	return street
}
