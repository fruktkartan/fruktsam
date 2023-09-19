package trees

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/fruktkartan/fruktsam/types"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // for sqlx
)

type Trees struct {
	entries []Entry
}

type Entry struct {
	Key      types.NullStringTrimmed
	Type     types.NullStringTrimmed
	Desc     types.NullStringTrimmed
	Img      types.NullString
	By       types.NullString
	At       types.NullTime
	Lat, Lon sql.NullFloat64
}

func (t *Trees) FromDB() error {
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

	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("Connect: %w", err)
	}
	if err := db.Select(&t.entries, query); err != nil {
		return fmt.Errorf("Select: %w", err)
	}

	// TODO could put in a map

	// t.prepare() // TODO?

	return nil
}

func (t Trees) Get(key string) (Entry, bool) {
	for idx := range t.entries {
		tree := t.entries[idx]
		if !tree.Key.Valid {
			continue
		}
		if tree.Key.String() == key {
			return tree, true
		}
	}
	return Entry{}, false
}

func (t Trees) Count() int {
	return len(t.entries)
}
