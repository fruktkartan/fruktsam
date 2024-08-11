package types

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/fruktkartan/fruktsam/internal/util"
)

type NullString struct {
	sql.NullString
}

func (ns *NullString) String() string {
	if !ns.NullString.Valid {
		return ""
	}
	return ns.NullString.String
}

type NullStringTrimmed struct {
	sql.NullString
}

func (ns *NullStringTrimmed) String() string {
	if !ns.NullString.Valid {
		return ""
	}
	return strings.TrimSpace(ns.NullString.String)
}

type NullTime struct {
	sql.NullTime
}

func (nt *NullTime) String() string {
	if !nt.NullTime.Valid {
		return ""
	}
	return util.FormatDateTime(nt.Time)
}

func (nt *NullTime) Date() string {
	if !nt.NullTime.Valid {
		return ""
	}
	return util.FormatDate(nt.Time)
}

func (nt *NullTime) WeekNumber() string {
	if !nt.NullTime.Valid {
		return ""
	}
	_, w := nt.Time.ISOWeek()
	return strconv.Itoa(w)
}

func (nt *NullTime) TimeStr() string {
	if !nt.NullTime.Valid {
		return ""
	}
	return util.FormatTime(nt.Time)
}

type Pos struct {
	Lat, Lon float64
}

func (p *Pos) GeohackURL() string {
	return fmt.Sprintf("https://geohack.toolforge.org/geohack.php?params=%g_N_%g_E",
		p.Lat, p.Lon)
}

func (p *Pos) OSMURL() string {
	return fmt.Sprintf("https://www.openstreetmap.org/?mlat=%g&mlon=%g&zoom=15&layers=M",
		p.Lat, p.Lon)
}

func (p *Pos) GoogmapsURL() string {
	return fmt.Sprintf("https://www.google.com/maps?ll=%g,%g&q=%g,%g&hl=en&t=k&z=15",
		p.Lat, p.Lon, p.Lat, p.Lon)
}
