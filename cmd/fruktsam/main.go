package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/fruktkartan/fruktsam/geo"
	"github.com/fruktkartan/fruktsam/history"
	"github.com/joho/godotenv"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// TODO consider puring history entries that contains old-style ssm_keys?

const envfile = ".env"
const outfile = "dist/index.html"

// TODO disabled for now
// const historycache = "historycache"
const reversecache = "reversecache"

const defaultSinceDays = 60

func main() {
	var sinceFlag = defaultSinceDays
	var err error

	app := kingpin.New("fruktsam", "Generate html from Fruktkartan edit history")
	app.Flag("since", fmt.Sprintf("How many days back, default: %d", defaultSinceDays)).
		PlaceHolder("DAYS").IntVar(&sinceFlag)
	app.HelpFlag.Short('h')
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if err = godotenv.Load(envfile); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading %s file: %s", envfile, err)
	}

	type templateData struct {
		SinceDays int
		History   history.History
	}
	var data templateData

	if err = history.FromDB(&data.History, sinceFlag); err != nil {
		log.Fatal(err)
	}
	data.SinceDays = sinceFlag

	// if _, err = os.Stat(historycache); err != nil {
	// 	fmt.Printf("filling cache file\n")
	// 	if err = history.HistoryFromDB(&data.History); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	if err = data.History.Store(historycache); err != nil {
	// 		log.Fatal(err)
	// 	}
	// } else {
	// 	fmt.Printf("cache file found\n")
	// }
	// if data.History, err = history.LoadCache(historycache); err != nil {
	// 	log.Fatal(err)
	// }
	fmt.Printf("history entries: %d\n", len(data.History))

	sort.Slice(data.History, func(i, j int) bool {
		return data.History[i].ChangeID > data.History[j].ChangeID
	})

	revcache := geo.ReverseCache{}

	if err = revcache.Load(reversecache); err != nil {
		log.Fatal(err)
	}

	dmp := diffmatchpatch.New()
	for idx := range data.History {
		he := &data.History[idx]

		if he.Lat.Valid {
			p := geo.Pos{Lat: he.Lat.Float64, Lon: he.Lon.Float64}
			if !revcache.Has(p) {
				fmt.Printf("get reverse address for entry %d\n", he.ChangeID)
				revcache.Add(p)
				time.Sleep(1 * time.Second)
			}
			he.Address = revcache.FormatAddress(p)
			he.GeoURL = p.GeohackURL()
		}
		if he.NewLat.Valid {
			p := geo.Pos{Lat: he.NewLat.Float64, Lon: he.NewLon.Float64}
			if !revcache.Has(p) {
				fmt.Printf("get reverse address for entry %d\n", he.ChangeID)
				revcache.Add(p)
				time.Sleep(1 * time.Second)
			}
			he.NewAddress = revcache.FormatAddress(p)
			he.NewGeoURL = p.GeohackURL()
		}

		if he.ChangeOp == "UPDATE" {
			diffs := dmp.DiffMain(he.Desc.String(), he.NewDesc.String(), false)
			he.DescDiff = dmp.DiffPrettyHtml(diffs)
		}
	}

	if err = revcache.Store(reversecache); err != nil {
		fmt.Println(err)
	}

	tmpl, err := template.ParseFiles("tmpl_index.html")
	if err != nil {
		log.Fatal(err)
	}

	var f *os.File
	if err = os.MkdirAll(filepath.Dir(outfile), 0770); err != nil {
		log.Fatal(err)
	}
	if f, err = os.Create(outfile); err != nil {
		log.Fatal(err)
	}
	if err = tmpl.Execute(f, &data); err != nil {
		log.Fatal(err)
	}
}
