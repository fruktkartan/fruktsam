package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/fruktkartan/fruktsam/geo"
	"github.com/fruktkartan/fruktsam/history"
	"github.com/joho/godotenv"
	"github.com/sergi/go-diff/diffmatchpatch"

	_ "github.com/lib/pq"
)

// TODO maybe don't do historycache for now? I removed lots of early
// import-edits from the history. I would be easier to move forward without it,
// I think. Just batch runs to generate page every night or so.

// TODO consider puring history that contains old-style ssm_keys?

const envfile = ".env"
const outfile = "dist/index.html"
const historycache = "historycache"
const reversecache = "reversecache"

func main() {
	var err error

	if err = godotenv.Load(envfile); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading %s file: %s", envfile, err)
	}

	type templateData struct {
		History history.History
	}
	var data templateData

	if _, err = os.Stat(historycache); err != nil {
		fmt.Printf("filling cache file\n")
		if err = history.HistoryFromDB(&data.History); err != nil {
			log.Fatal(err)
		}
		if err = data.History.Store(historycache); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("cache file found\n")
	}
	if data.History, err = history.LoadCache(historycache); err != nil {
		log.Fatal(err)
	}
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
				fmt.Println(len(data.History) - idx)
				revcache.Add(p)
				time.Sleep(1 * time.Second)
			}
			he.Address = revcache.FormatAddress(p)
			he.GeoURL = p.GeohackURL()
		}
		if he.NewLat.Valid {
			p := geo.Pos{Lat: he.NewLat.Float64, Lon: he.NewLon.Float64}
			if !revcache.Has(p) {
				fmt.Println(len(data.History) - idx)
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
