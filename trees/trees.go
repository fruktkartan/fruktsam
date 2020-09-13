package trees

import (
	"log"
	"os"
	"strconv"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // for sqlx
)

type Trees struct {
	count int
}

func (t *Trees) Count() string {
	if t.count == 0 {
		var err error
		t.count, err = count()
		if err != nil {
			log.Printf("Trees.Count: " + err.Error())
			t.count = -1
		}
	}
	if t.count < 0 {
		return "?"
	}
	return strconv.Itoa(t.count)
}

func count() (int, error) {
	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return 0, err
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM trees`); err != nil {
		return 0, err
	}
	return count, nil
}
