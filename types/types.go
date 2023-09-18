package types

import (
	"database/sql"
	"strings"

	"github.com/fruktkartan/fruktsam/util"
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

func (nt *NullTime) TimeStr() string {
	if !nt.NullTime.Valid {
		return ""
	}
	return util.FormatTime(nt.Time)
}
