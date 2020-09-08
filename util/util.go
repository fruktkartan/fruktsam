package util

import (
	"log"
	"time"

	"github.com/goodsign/monday"
)

var location *time.Location

const (
	mondayLocale = monday.LocaleSvSE
	dateTimeFmt  = "2006-01-02 15.04"
)

func FormatDate(t time.Time) string {
	if location == nil {
		initLocation()
	}
	return monday.Format(t.In(location), dateTimeFmt, mondayLocale)
}

func initLocation() {
	var err error
	if location, err = time.LoadLocation("Europe/Stockholm"); err != nil {
		log.Fatal(err)
	}
}
