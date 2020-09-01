package util

import (
	"log"
	"time"

	"github.com/goodsign/monday"
)

var loc *time.Location

func FormatDate(t time.Time) string {
	if loc == nil {
		var err error
		if loc, err = time.LoadLocation("Europe/Stockholm"); err != nil {
			log.Fatal(err)
		}
	}
	return monday.Format(t.In(loc), "2006-01-02 15.04", monday.LocaleSvSE)
}
