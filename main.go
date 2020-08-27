package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

type Foo struct {
	Lat  float64
	Lon  float64
	Type string
}

// http://janmatuschek.de/LatitudeLongitudeBoundingCoordinates

// https://stackoverflow.com/a/238558
func deg2rad(deg float64) float64 {
	return math.Pi * deg / 180
}
func rad2deg(rad float64) float64 {
	return 180 * rad / math.Pi
}

// Semi-axes of WGS-84 geoidal reference.
const wgs84A = 6378137.0 // Major semiaxis [m]
const wgs84B = 6356752.3 // Minor semiaxis [m]

// Earth radius at a given latitude, according to the WGS-84 ellipsoid [m].
func wgs84EarthRadius(lat float64) float64 {
	// http://en.wikipedia.org/wiki/Earth_radius
	var An = wgs84A * wgs84A * math.Cos(lat)
	var Bn = wgs84B * wgs84B * math.Sin(lat)
	var Ad = wgs84A * math.Cos(lat)
	var Bd = wgs84B * math.Sin(lat)

	return math.Sqrt((An*An + Bn*Bn) / (Ad*Ad + Bd*Bd))
}

// Bounding box surrounding the point at given coordinates, assuming local
// approximation of Earth surface as a sphere of radius given by WGS84.
func boundingBox(latDeg, lonDeg, halfsideKM float64) (latMin, lonMin, latMax, lonMax float64) {
	var lat = deg2rad(latDeg)
	var lon = deg2rad(lonDeg)
	var halfSide = 1000 * halfsideKM

	// Radius of Earth at given latitude
	var radius = wgs84EarthRadius(lat)
	// Radius of the parallel at given latitude
	var pradius = radius * math.Cos(lat)

	latMin = rad2deg(lat - halfSide/radius)
	lonMin = rad2deg(lon - halfSide/pradius)
	latMax = rad2deg(lat + halfSide/radius)
	lonMax = rad2deg(lon + halfSide/pradius)

	return
}

func getJSON(url string, trees *[]Foo) error {
	var c = &http.Client{Timeout: 10 * time.Second}
	r, err := c.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	// return json.NewDecoder(r.Body).Decode(target)
	for {
		if err := dec.Decode(&trees); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}

type tree struct {
	Key, Type, Desc, By, At, Lat, Lon string
}

type nullString struct {
	sql.NullString
}

func (ns nullString) String() string {
	if !ns.NullString.Valid {
		return ""
	}
	return ns.NullString.String
}

type historyEntry struct {
	ID                                                     int
	At                                                     string
	Op                                                     string
	OldKey, OldType, OldDesc, OldBy, OldAt, OldLat, OldLon nullString
	NewKey, NewType, NewDesc, NewBy, NewAt, NewLat, NewLon nullString
}

type history []historyEntry

func (c history) store(cachefile string) error {
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(c)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(cachefile, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = f.Write(b.Bytes()); err != nil {
		return err
	}

	return nil
}

func loadCache(cachefile string) (history, error) {
	cache := history{}

	f, err := os.Open(cachefile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		return cache, nil
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	err = dec.Decode(&cache)
	if err != nil {
		panic(err)
	}

	return cache, nil
}

func dataFromDB(data *history) error {
	query := `SELECT id, at, op
                     , old_json->>'ssm_key' AS oldkey
                     , old_json->>'type' AS oldtype
                     , old_json->>'description' AS olddesc
                     , old_json->>'added_by' AS oldby
                     , old_json->>'added_at' AS oldat
                     , ST_Y(old_point) AS oldlat
                     , ST_X(old_point) AS oldlon
                     , new_json->>'ssm_key' AS newkey
                     , new_json->>'type' AS newtype
                     , new_json->>'description' AS newdesc
                     , new_json->>'added_by' AS newby
                     , new_json->>'added_at' AS newat
                     , ST_Y(new_point) AS newlat
                     , ST_X(new_point) AS newlon
                FROM history
                     , ST_GeomFromWKB(DECODE(old_json->>'point', 'hex')) AS old_point
                     , ST_GeomFromWKB(DECODE(new_json->>'point', 'hex')) AS new_point
               ORDER BY id`

	db, err := sqlx.Connect("postgres", os.Getenv("FRUKTKARTAN_DATABASE_URI"))
	if err != nil {
		return err
	}

	err = db.Select(data, query)
	if err != nil {
		return err
	}

	return nil
}

const envFile = ".env"

func main() {
	var err error

	if err = godotenv.Load(envFile); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading %s file: %s", envFile, err)
	}

	var h history

	if _, err = os.Stat("./cache"); err != nil {
		fmt.Printf("filling cache file\n")
		if err = dataFromDB(&h); err != nil {
			log.Fatal(err)
		}
		if err = h.store("./cache"); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("cache file found\n")
	}
	if h, err = loadCache("./cache"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("history has %d\n", len(h))

	for idx := range h {
		e := h[idx]
		if e.Op == "DELETE" {
			fmt.Printf("%s at:%s:", e.Op, e.At)
			fmt.Printf(" OLD: key:%s: type:%s: desc:%s: by:%s: at:%s: geo:%s,%s",
				strings.TrimSpace(e.OldKey.String()), strings.TrimSpace(e.OldType.String()),
				e.OldDesc, e.OldBy, e.OldAt, e.OldLat, e.OldLon)
			fmt.Printf(" NEW: key:%s: type:%s: desc:%s: by:%s: at:%s: geo:%s,%s",
				strings.TrimSpace(e.NewKey.String()), strings.TrimSpace(e.NewType.String()),
				e.NewDesc, e.NewBy, e.NewAt, e.NewLat, e.NewLon)
			fmt.Printf("\n")
		}
	}

	// //
	// var trees []Foo
	// err = getJSON("https://fruktkartan-api.herokuapp.com/edits", &trees)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(trees[0])

	// boundingBox(trees[0].Lat, trees[0].Lon, 100)

	// //
	// t := trees[0]

	// ctx := sm.NewContext()
	// // TODO not gettin carto tiles, just attribution
	// // tile := sm.NewTileProviderCartoLight()
	// // ctx.SetTileProvider(tile)
	// ctx.SetSize(200, 200)
	// marker := sm.NewMarker(s2.LatLngFromDegrees(t.Lat, t.Lon), color.RGBA{0xff, 0, 0, 0xff}, 20.0)
	// marker.Label = strings.TrimSpace(t.Type)
	// marker.SetLabelColor(color.Black)
	// ctx.AddMarker(marker)

	// var overview, zoomin image.Image

	// ctx.SetZoom(10)
	// if overview, err = ctx.Render(); err != nil {
	// 	panic(err)
	// }

	// ctx.SetZoom(17)
	// if zoomin, err = ctx.Render(); err != nil {
	// 	panic(err)
	// }

	// // starting position of the second image (bottom left)
	// sp2 := image.Point{overview.Bounds().Dx(), 0}
	// // new rectangle for the second image
	// r2 := image.Rectangle{sp2, sp2.Add(zoomin.Bounds().Size())}
	// // rectangle for the big image
	// r := image.Rectangle{image.Point{0, 0}, r2.Max}
	// rgba := image.NewRGBA(r)
	// draw.Draw(rgba, overview.Bounds(), overview, image.Point{0, 0}, draw.Src)
	// draw.Draw(rgba, r2, zoomin, image.Point{0, 0}, draw.Src)

	// if err := gg.SavePNG("a.png", rgba); err != nil {
	// 	panic(err)
	// }
}

//https://fruktkartan-api.herokuapp.com/edits
