package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"database/sql"

	"github.com/goodsign/monday"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

const envFile = ".env"

var loc *time.Location

func main() {
	var err error

	if err = godotenv.Load(envFile); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading %s file: %s", envFile, err)
	}

	if loc, err = time.LoadLocation("Europe/Stockholm"); err != nil {
		log.Fatal(err)
	}

	type templateData struct {
		History history
	}
	var data templateData

	if _, err = os.Stat("./cache"); err != nil {
		fmt.Printf("filling cache file\n")
		if err = historyFromDB(&data.History); err != nil {
			log.Fatal(err)
		}
		if err = data.History.store("./cache"); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("cache file found\n")
	}
	if data.History, err = loadCache("./cache"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("history entries: %d\n", len(data.History))

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

	// for idx := range h {
	// 	e := h[idx]
	// 	if e.Op == "DELETE" {
	// 		fmt.Printf("%s at:%s:", e.Op, e.At)
	// 		fmt.Printf(" OLD: key:%s: type:%s: desc:%s: by:%s: at:%s: geo:%s,%s",
	// 			strings.TrimSpace(e.OldKey.String()), strings.TrimSpace(e.OldType.String()),
	// 			e.OldDesc, e.OldBy, e.OldAt, e.OldLat, e.OldLon)
	// 		fmt.Printf(" NEW: key:%s: type:%s: desc:%s: by:%s: at:%s: geo:%s,%s",
	// 			strings.TrimSpace(e.NewKey.String()), strings.TrimSpace(e.NewType.String()),
	// 			e.NewDesc, e.NewBy, e.NewAt, e.NewLat, e.NewLon)
	// 		fmt.Printf("\n")
	// 	}
	// }
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

	return prettyTime(nt.NullTime.Time)
}

func prettyTime(t time.Time) string {
	return monday.Format(t.In(loc), "2 January 2006 kl. 15.04", monday.LocaleSvSE)
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

const outfile = "dist/index.html"

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
                     , ST_GeomFromWKB(DECODE(new_json->>'point', 'hex')) AS new_point
               ORDER BY id DESC`

	db, err := sqlx.Connect("postgres", os.Getenv("FRUKTKARTAN_DBURI"))
	if err != nil {
		return err
	}

	return db.Select(h, query)
}
