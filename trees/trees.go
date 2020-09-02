package trees

import (
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // for sqlx
)

func Count() (int, error) {
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
