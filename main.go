package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strings"

	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

type nullString struct {
	sql.NullString
}

func (ns nullString) String() string {
	if !ns.NullString.Valid {
		return ""
	}

	return ns.NullString.String
}

type historyEntry struct {
	ID                                                     int
	At                                                     string
	Op                                                     string
	OldKey, OldType, OldDesc, OldBy, OldAt, OldLat, OldLon nullString
	NewKey, NewType, NewDesc, NewBy, NewAt, NewLat, NewLon nullString
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
	err = dec.Decode(&cache)
	if err != nil {
		panic(err)
	}

	return cache, nil
}

func dataFromDB(data *history) error {
	query := `SELECT id, at, op
                     , old_json->>'ssm_key' AS oldkey
                     , old_json->>'type' AS oldtype
                     , old_json->>'description' AS olddesc
                     , old_json->>'added_by' AS oldby
                     , old_json->>'added_at' AS oldat
                     , ST_Y(old_point) AS oldlat
                     , ST_X(old_point) AS oldlon
                     , new_json->>'ssm_key' AS newkey
                     , new_json->>'type' AS newtype
                     , new_json->>'description' AS newdesc
                     , new_json->>'added_by' AS newby
                     , new_json->>'added_at' AS newat
                     , ST_Y(new_point) AS newlat
                     , ST_X(new_point) AS newlon
                FROM history
                     , ST_GeomFromWKB(DECODE(old_json->>'point', 'hex')) AS old_point
                     , ST_GeomFromWKB(DECODE(new_json->>'point', 'hex')) AS new_point
               ORDER BY id`

	db, err := sqlx.Connect("postgres", os.Getenv("FRUKTKARTAN_DATABASE_URI"))
	if err != nil {
		return err
	}

	err = db.Select(data, query)
	if err != nil {
		return err
	}

	return nil
}

const envFile = ".env"

func main() {
	var err error

	if err = godotenv.Load(envFile); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading %s file: %s", envFile, err)
	}

	var h history

	if _, err = os.Stat("./cache"); err != nil {
		fmt.Printf("filling cache file\n")
		if err = dataFromDB(&h); err != nil {
			log.Fatal(err)
		}
		if err = h.store("./cache"); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("cache file found\n")
	}
	if h, err = loadCache("./cache"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("history has %d\n", len(h))

	for idx := range h {
		e := h[idx]
		if e.Op == "DELETE" {
			fmt.Printf("%s at:%s:", e.Op, e.At)
			fmt.Printf(" OLD: key:%s: type:%s: desc:%s: by:%s: at:%s: geo:%s,%s",
				strings.TrimSpace(e.OldKey.String()), strings.TrimSpace(e.OldType.String()),
				e.OldDesc, e.OldBy, e.OldAt, e.OldLat, e.OldLon)
			fmt.Printf(" NEW: key:%s: type:%s: desc:%s: by:%s: at:%s: geo:%s,%s",
				strings.TrimSpace(e.NewKey.String()), strings.TrimSpace(e.NewType.String()),
				e.NewDesc, e.NewBy, e.NewAt, e.NewLat, e.NewLon)
			fmt.Printf("\n")
		}
	}
}
