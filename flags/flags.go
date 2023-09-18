package flags

import (
	"fmt"
	"os"

	"github.com/fruktkartan/fruktsam/types"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // for sqlx
)

type Flags struct {
	entries []Entry
}

type Entry struct {
	By     types.NullString
	At     types.NullTime
	Key    types.NullStringTrimmed
	Flag   types.NullStringTrimmed
	Reason types.NullStringTrimmed
}

func (h *Flags) Len() int {
	return len(h.entries)
}

func (h *Flags) Entries() []Entry {
	return h.entries
}

func (h *Flags) FromDB() error {
	if len(h.entries) > 0 {
		return fmt.Errorf("history not empty, refusing to fill from db")
	}

	query := `SELECT flagged_by AS by
                   , flagged_at AS at
                   , tree AS key
                   , flag
                   , reason
                FROM flags`

	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("Connect: %w", err)
	}
	if err := db.Select(&h.entries, query); err != nil {
		return fmt.Errorf("Select: %w", err)
	}

	return nil
}
