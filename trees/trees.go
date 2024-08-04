package trees

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fruktkartan/fruktsam/geo"
	"github.com/fruktkartan/fruktsam/types"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // for sqlx
)

type Trees struct {
	entries map[string]*Entry
}

type Entry struct {
	Key      types.NullStringTrimmed
	Type     types.NullStringTrimmed
	Desc     types.NullStringTrimmed
	Img      types.NullString
	By       types.NullString
	At       types.NullTime
	Lat, Lon sql.NullFloat64

	Address string
	Pos     geo.Pos
}

func (t *Trees) FromDB() error {
	if t.entries == nil {
		t.entries = make(map[string]*Entry)
	}

	if len(t.entries) > 0 {
		return fmt.Errorf("not empty, refusing to fill from db")
	}

	query := `SELECT ssm_key AS key
                   , type
                   , description AS desc
                   , img
                   , added_by AS by
                   , added_at AS at
                   , ST_Y(point) AS lat
                   , ST_X(point) AS lon
                FROM trees`

	var rows []Entry

	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("Connect: %w", err)
	}
	if err := db.Select(&rows, query); err != nil {
		return fmt.Errorf("Select: %w", err)
	}

	for i := range rows {
		t.entries[rows[i].Key.String()] = &rows[i]
	}

	revcache := geo.GetInstance()

	for i := range t.entries {
		if t.entries[i].Lat.Valid {
			// history.prepare uses this code too
			p := geo.Pos{Lat: t.entries[i].Lat.Float64, Lon: t.entries[i].Lon.Float64}
			if !revcache.Has(p) {
				log.Printf("get reverse address for trees entry %s\n", t.entries[i].Key.String())
				revcache.Add(p)
				time.Sleep(1 * time.Second)
			}
			t.entries[i].Address = revcache.FormatAddress(p)
			t.entries[i].Pos = p
		}
	}

	return nil
}

func (t Trees) Get(key string) (Entry, bool) {
	if tree, ok := t.entries[key]; ok {
		return *tree, true
	}
	return Entry{}, false
}

func (t Trees) Count() int {
	return len(t.entries)
}
