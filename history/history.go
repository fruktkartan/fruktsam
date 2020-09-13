package history

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fruktkartan/fruktsam/geo"
	"github.com/fruktkartan/fruktsam/util"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // for sqlx
	"github.com/sergi/go-diff/diffmatchpatch"
)

type History struct {
	SinceDays                 int
	entries                   []Entry
	prepared                  bool
	deletes, inserts, updates int
}

type Entry struct {
	ChangeID int
	ChangeAt nullTime
	ChangeOp string

	Key        nullStringTrimmed
	Type, Desc nullStringTrimmed
	By         nullString
	At         nullTime
	Lat, Lon   sql.NullFloat64

	KeyNew           nullStringTrimmed
	TypeNew, DescNew nullStringTrimmed
	ByNew            nullString
	AtNew            nullTime
	LatNew, LonNew   sql.NullFloat64

	Address, AddressNew string
	GeoURL, GeoURLNew   string
	DescDiff            string
}

func (h *History) Len() int {
	return len(h.entries)
}

func (h *History) Entries() []Entry {
	h.prepare()
	h.sort()
	return h.entries
}

func (h *History) Deletes() int {
	h.prepare()
	return h.deletes
}

func (h *History) Inserts() int {
	h.prepare()
	return h.inserts
}
func (h *History) Updates() int {
	h.prepare()
	return h.updates
}

func (h *History) Net() string {
	h.prepare()
	net := h.inserts - h.deletes
	plus := ""
	if net > 0 {
		plus = "+"
	}
	return fmt.Sprintf("%s%d", plus, net)
}

func (h *History) FromDB(sinceDays int) error {
	if len(h.entries) > 0 {
		return fmt.Errorf("history not empty, refusing to fill from db")
	}
	h.prepared = false
	h.SinceDays = sinceDays

	query := `SELECT id AS changeid, at AS changeat, op AS changeop
                     , old_json->>'ssm_key' AS key
                     , old_json->>'type' AS type
                     , old_json->>'description' AS desc
                     , old_json->>'added_by' AS by
                     , (old_json->>'added_at')::timestamp AS at
                     , ST_Y(old_point) AS lat
                     , ST_X(old_point) AS lon
                     , new_json->>'ssm_key' AS keynew
                     , new_json->>'type' AS typenew
                     , new_json->>'description' AS descnew
                     , new_json->>'added_by' AS bynew
                     , (new_json->>'added_at')::timestamp AS atnew
                     , ST_Y(new_point) AS latnew
                     , ST_X(new_point) AS lonnew
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
	if err := db.Select(&h.entries, query); err != nil {
		return err
	}

	return nil
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

func (h *History) Save(cachefile string) error {
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

func (h *History) Load(cachefile string) error {
	if len(h.entries) > 0 {
		return fmt.Errorf("history not empty, refusing to load from file")
	}
	h.prepared = false

	f, err := os.Open(cachefile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	if err := dec.Decode(h); err != nil {
		return err
	}

	return nil
}

const reversefile = "reversecache"

func (h *History) prepare() {
	if h.prepared {
		return
	}
	var err error

	revcache := geo.NewReverseCache()

	if err = revcache.Load(reversefile); err != nil {
		log.Fatal(err)
	}

	dmp := diffmatchpatch.New()
	for idx := range h.entries {
		he := &h.entries[idx]

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
		if he.LatNew.Valid {
			p := geo.Pos{Lat: he.LatNew.Float64, Lon: he.LonNew.Float64}
			if !revcache.Has(p) {
				fmt.Printf("get reverse address for entry %d\n", he.ChangeID)
				revcache.Add(p)
				time.Sleep(1 * time.Second)
			}
			he.AddressNew = revcache.FormatAddress(p)
			he.GeoURLNew = p.GeohackURL()
		}

		if he.ChangeOp == "UPDATE" {
			he.DescDiff = dmp.DiffPrettyHtml(
				dmp.DiffMain(he.Desc.String(), he.DescNew.String(), false))
		}

		switch he.ChangeOp {
		case "DELETE":
			h.deletes++
		case "INSERT":
			h.inserts++
		case "UPDATE":
			h.updates++
		}
	}

	if err = revcache.Save(reversefile); err != nil {
		fmt.Println(err)
	}

	h.prepared = true
}

func (h *History) sort() {
	sort.Slice(h.entries, func(i, j int) bool {
		return h.entries[i].ChangeID > h.entries[j].ChangeID
	})
}
