package history

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"os"
	"strings"

	"github.com/fruktkartan/fruktsam/util"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // for sqlx
)

type History []Entry

type Entry struct {
	ChangeID int
	ChangeAt nullTime
	ChangeOp string

	Key        nullStringTrimmed
	Type, Desc nullStringTrimmed
	By         nullString
	At         nullTime
	Lat, Lon   sql.NullFloat64

	NewKey           nullStringTrimmed
	NewType, NewDesc nullStringTrimmed
	NewBy            nullString
	NewAt            nullTime
	NewLat, NewLon   sql.NullFloat64

	// TODO should perhaps not serialize these, but they do need to be exported
	// (capitalized) for exposing to template
	// Maybe they can be setter/getter functions?
	Address, NewAddress string
	GeoURL, NewGeoURL   string
	DescDiff            string
}

func (h *History) Store(cachefile string) error {
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(h)
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

func LoadCache(cachefile string) ([]Entry, error) {
	cache := History{}

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

func FromDB(h *History, sinceDays int) error {
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
	if sinceDays > 0 {
		query += fmt.Sprintf(" WHERE at > (CURRENT_DATE - INTERVAL '%d days')", sinceDays)
	}

	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	return db.Select(h, query)
}

type nullString struct {
	sql.NullString
}

func (ns *nullString) String() string {
	if !ns.NullString.Valid {
		return ""
	}
	return ns.NullString.String
}

type nullStringTrimmed struct {
	sql.NullString
}

func (ns *nullStringTrimmed) String() string {
	if !ns.NullString.Valid {
		return ""
	}
	return strings.TrimSpace(ns.NullString.String)
}

type nullTime struct {
	sql.NullTime
}

func (nt *nullTime) String() string {
	if !nt.NullTime.Valid {
		return ""
	}
	return util.FormatDateTime(nt.Time)
}

func (nt *nullTime) Date() string {
	if !nt.NullTime.Valid {
		return ""
	}
	return util.FormatDate(nt.Time)
}

func (nt *nullTime) TimeStr() string {
	if !nt.NullTime.Valid {
		return ""
	}
	return util.FormatTime(nt.Time)
}
