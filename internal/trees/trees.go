package trees

import (
	"database/sql"
	"fmt"
	"os"
	"sort"

	"github.com/fruktkartan/fruktsam/internal/types"
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
		return fmt.Errorf("failed Connect: %w", err)
	}
	if err := db.Select(&rows, query); err != nil {
		return fmt.Errorf("failed Select: %w", err)
	}

	for idx := range rows {
		t.entries[rows[idx].Key.String()] = &rows[idx]
	}

	// t.prepare() // TODO?

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

type TypeCount struct {
	Type  string
	Count int
}

func (t Trees) TypeCounts() []TypeCount {
	counts := make(map[string]int)

	for _, e := range t.entries {
		counts[e.Type.String()]++
	}

	typeCounts := make([]TypeCount, 0, len(counts))
	for typ, count := range counts {
		typeCounts = append(typeCounts, TypeCount{
			Type:  typ,
			Count: count,
		})
	}

	sort.Slice(typeCounts, func(i, j int) bool {
		if typeCounts[i].Count != typeCounts[j].Count {
			return typeCounts[i].Count > typeCounts[j].Count
		}
		// secondary sort alpha for determinism
		return typeCounts[i].Type < typeCounts[j].Type
	})

	return typeCounts
}
