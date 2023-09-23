package flags

import (
	"fmt"
	"os"
	"sort"

	"github.com/fruktkartan/fruktsam/history"
	"github.com/fruktkartan/fruktsam/trees"
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
	Key    types.NullStringTrimmed // Actually NOT NULL in db table
	Flag   types.NullStringTrimmed // Actually NOT NULL in db table
	Reason types.NullStringTrimmed

	Tree trees.Entry
}

func (e Entry) TreeImgURL() string {
	img := e.Tree.Img.String()
	if img == "" {
		return ""
	}
	return history.ImageURLBase + fmt.Sprintf(history.ImageURLPathFmt, img)
}

func (f *Flags) Count() int {
	return len(f.entries)
}

func (f *Flags) Entries() []Entry {
	return f.entries
}

func (f *Flags) FromDB(trees trees.Trees) error {
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
	if err := db.Select(&f.entries, query); err != nil {
		return fmt.Errorf("Select: %w", err)
	}

	for idx := range f.entries {
		flagged := &f.entries[idx]

		tree, ok := trees.Get(flagged.Key.String())
		if !ok {
			fmt.Printf("Flagged tree %s not found in tree table\n", tree.Key.String())
		}

		flagged.Tree = tree
	}

	sort.Slice(f.entries, func(i, j int) bool {
		return f.entries[i].At.Time.After(f.entries[j].At.Time)
	})

	return nil
}
