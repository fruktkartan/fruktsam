package util

import (
	"log"
	"time"

	"github.com/goodsign/monday"
)

var location *time.Location

const (
	mondayLocale = monday.LocaleSvSE
	dateFmt      = "2006-01-02"
	timeFmt      = "15:04"
	dateTimeFmt  = dateFmt + " " + timeFmt
)

func FormatDate(t time.Time) string {
	if location == nil {
		initLocation()
	}
	return monday.Format(t.In(location), dateFmt, mondayLocale)
}

func FormatTime(t time.Time) string {
	if location == nil {
		initLocation()
	}
	return monday.Format(t.In(location), timeFmt, mondayLocale)
}

func FormatDateTime(t time.Time) string {
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
