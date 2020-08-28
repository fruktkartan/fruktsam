package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"database/sql"

	"github.com/goodsign/monday"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

const envfile = ".env"
const outfile = "dist/index.html"
const cachefile = "historycache"

var loc *time.Location

func main() {
	var err error

	if err = godotenv.Load(envfile); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading %s file: %s", envfile, err)
	}

	if loc, err = time.LoadLocation("Europe/Stockholm"); err != nil {
		log.Fatal(err)
	}

	type templateData struct {
		History history
	}
	var data templateData

	if _, err = os.Stat(cachefile); err != nil {
		fmt.Printf("filling cache file\n")
		if err = historyFromDB(&data.History); err != nil {
			log.Fatal(err)
		}
		if err = data.History.store(cachefile); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("cache file found\n")
	}
	if data.History, err = loadCache(cachefile); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("history entries: %d\n", len(data.History))

	sort.Slice(data.History, func(i, j int) bool {
		return data.History[i].ChangeID > data.History[j].ChangeID
	})

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

type historyEntry struct {
	ChangeID int
	ChangeAt nullTime
	ChangeOp string

	Key        nullString
	Type, Desc nullStringTrimmed
	By         nullString
	At         nullTime
	Lat, Lon   nullString

	NewKey           nullString
	NewType, NewDesc nullStringTrimmed
	NewBy            nullString
	NewAt            nullTime
	NewLat, NewLon   nullString
}

type history []historyEntry

func (c history) store(cachefile string) error {
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(c)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(cachefile, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = f.Write(b.Bytes()); err != nil {
		return err
	}

	return nil
}

func loadCache(cachefile string) (history, error) {
	cache := history{}

	f, err := os.Open(cachefile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		return cache, nil
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	if err := dec.Decode(&cache); err != nil {
		return nil, err
	}

	return cache, nil
}

func historyFromDB(h *history) error {
	query := `SELECT id AS changeid, at AS changeat, op AS changeop
                     , old_json->>'ssm_key' AS key
                     , old_json->>'type' AS type
                     , old_json->>'description' AS desc
                     , old_json->>'added_by' AS by
                     , (old_json->>'added_at')::timestamp AS at
                     , ST_Y(old_point) AS lat
                     , ST_X(old_point) AS lon
                     , new_json->>'ssm_key' AS newkey
                     , new_json->>'type' AS newtype
                     , new_json->>'description' AS newdesc
                     , new_json->>'added_by' AS newby
                     , (new_json->>'added_at')::timestamp AS newat
                     , ST_Y(new_point) AS newlat
                     , ST_X(new_point) AS newlon
                FROM history
                     , ST_GeomFromWKB(DECODE(old_json->>'point', 'hex')) AS old_point
                     , ST_GeomFromWKB(DECODE(new_json->>'point', 'hex')) AS new_point`

	db, err := sqlx.Connect("postgres", os.Getenv("FRUKTKARTAN_DATABASEURI"))
	if err != nil {
		return err
	}

	return db.Select(h, query)
}

type nullString struct {
	sql.NullString
}

func (ns nullString) String() string {
	if !ns.NullString.Valid {
		return ""
	}

	return ns.NullString.String
}

type nullStringTrimmed struct {
	sql.NullString
}

func (ns nullStringTrimmed) String() string {
	if !ns.NullString.Valid {
		return ""
	}

	return strings.TrimSpace(ns.NullString.String)
}

type nullTime struct {
	sql.NullTime
}

func (nt nullTime) String() string {
	if !nt.NullTime.Valid {
		return ""
	}
	return monday.Format(nt.Time.In(loc), "2 Jan 2006 kl. 15.04", monday.LocaleSvSE)
}
