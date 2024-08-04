package history

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/fruktkartan/fruktsam/geo"
	"github.com/fruktkartan/fruktsam/types"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // for sqlx
	"github.com/sergi/go-diff/diffmatchpatch"
	"golang.org/x/image/draw"
)

const (
	ImageURLBase    = "https://fruktkartan-thumbs.s3.eu-north-1.amazonaws.com"
	ImageURLPathFmt = "/%s_1200.jpg"
)

type History struct {
	SinceDays                 int
	entries                   []Entry
	Deletes, Inserts, Updates int
}

type Entry struct {
	ChangeID int
	ChangeAt types.NullTime
	ChangeOp string

	Key      types.NullStringTrimmed
	Type     types.NullStringTrimmed
	Desc     types.NullStringTrimmed
	Img      types.NullString
	By       types.NullString
	At       types.NullTime
	Lat, Lon sql.NullFloat64

	KeyNew         types.NullStringTrimmed
	TypeNew        types.NullStringTrimmed
	DescNew        types.NullStringTrimmed
	ImgNew         types.NullString
	ByNew          types.NullString
	AtNew          types.NullTime
	LatNew, LonNew sql.NullFloat64

	Address, AddressNew string
	Pos, PosNew         geo.Pos
	DescDiff            string
	UpdateIsEmpty       bool
}

func (e Entry) ImgURL() string {
	img := e.Img.String()
	if img == "" {
		return ""
	}
	return ImageURLBase + fmt.Sprintf(ImageURLPathFmt, img)
}

func (e Entry) ImgURLNew() string {
	img := e.ImgNew.String()
	if img == "" {
		return ""
	}
	return ImageURLBase + fmt.Sprintf(ImageURLPathFmt, img)
}

func (e Entry) ImgFile() string {
	img := e.Img.String()
	if img == "" {
		return ""
	}
	return ImageFilePath(img)
}

func (e Entry) ImgFileNew() string {
	img := e.ImgNew.String()
	if img == "" {
		return ""
	}
	return ImageFilePath(img)
}

func (h *History) Count() int {
	return len(h.entries)
}

func (h *History) Entries() []Entry {
	return h.entries
}

func (h *History) Net() string {
	net := h.Inserts - h.Deletes
	plus := ""
	if net > 0 {
		plus = "+"
	}
	return fmt.Sprintf("%s%d", plus, net)
}

func (h *History) FromDB(sinceDays int) error {
	if len(h.entries) > 0 {
		return fmt.Errorf("history not empty, refusing to fill from db")
	}

	query := `SELECT id AS changeid, at AS changeat, op AS changeop
                     , old_json->>'ssm_key' AS key
                     , old_json->>'type' AS type
                     , old_json->>'description' AS desc
                     , old_json->>'img' AS img
                     , old_json->>'added_by' AS by
                     , (old_json->>'added_at')::timestamp AS at
                     , old_json#>>'{point,coordinates,1}' AS lat
                     , old_json#>>'{point,coordinates,0}' AS lon
                     , new_json->>'ssm_key' AS keynew
                     , new_json->>'type' AS typenew
                     , new_json->>'description' AS descnew
                     , new_json->>'img' AS imgnew
                     , new_json->>'added_by' AS bynew
                     , (new_json->>'added_at')::timestamp AS atnew
                     , new_json#>>'{point,coordinates,1}' AS latnew
                     , new_json#>>'{point,coordinates,0}' AS lonnew
                FROM history`
	if sinceDays > 0 {
		query += fmt.Sprintf(" WHERE at > (CURRENT_DATE - INTERVAL '%d days')", sinceDays)
	}

	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("Connect: %w", err)
	}
	if err := db.Select(&h.entries, query); err != nil {
		return fmt.Errorf("Select: %w", err)
	}

	h.SinceDays = sinceDays
	h.prepare()

	return nil
}

// TODO: currently unused
func (h *History) Save(cachefile string) error {
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(h)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(cachefile, os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(b.Bytes()); err != nil {
		return err
	}
	return nil
}

// TODO: currently unused
func (h *History) Load(cachefile string) error {
	if len(h.entries) > 0 {
		return fmt.Errorf("history not empty, refusing to load from file")
	}

	f, err := os.Open(cachefile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	if err := dec.Decode(h); err != nil {
		return err
	}

	// note h.SinceDays is unknown here
	h.prepare()

	return nil
}

func (h *History) prepare() {
	var err error

	if err = os.MkdirAll(outDir, 0o770); err != nil {
		log.Fatal(err)
	}
	if err = os.MkdirAll(outDir+"/"+imageDir, 0o770); err != nil {
		log.Fatal(err)
	}

	revcache := geo.GetInstance()

	dmp := diffmatchpatch.New()
	for idx := range h.entries {
		he := &h.entries[idx]

		CreateImageThumb(he.Img.String())
		CreateImageThumb(he.ImgNew.String())

		if he.Lat.Valid {
			p := geo.Pos{Lat: he.Lat.Float64, Lon: he.Lon.Float64}
			if !revcache.Has(p) {
				log.Printf("get reverse address for history entry %d\n", he.ChangeID)
				revcache.Add(p)
				time.Sleep(1 * time.Second)
			}
			he.Address = revcache.FormatAddress(p)
			he.Pos = p
		}
		if he.LatNew.Valid {
			p := geo.Pos{Lat: he.LatNew.Float64, Lon: he.LonNew.Float64}
			if !revcache.Has(p) {
				log.Printf("get reverse address (new) for history entry %d\n", he.ChangeID)
				revcache.Add(p)
				time.Sleep(1 * time.Second)
			}
			he.AddressNew = revcache.FormatAddress(p)
			he.PosNew = p
		}

		if he.ChangeOp == "UPDATE" {
			he.DescDiff = dmp.DiffPrettyHtml(
				dmp.DiffMain(he.Desc.String(), he.DescNew.String(), false))

			// Detect strange empty update
			if he.Type == he.TypeNew &&
				he.Desc == he.DescNew &&
				he.Img == he.ImgNew &&
				he.Lat == he.LatNew && he.Lon == he.LonNew {
				he.UpdateIsEmpty = true
			}
		}

		switch he.ChangeOp {
		case "DELETE":
			h.Deletes++
		case "INSERT":
			h.Inserts++
		case "UPDATE":
			h.Updates++
		}
	}

	if err = revcache.Save(); err != nil {
		log.Printf("revcache.Save failed: %s\n", err)
	}

	sort.Slice(h.entries, func(i, j int) bool {
		return h.entries[i].ChangeID > h.entries[j].ChangeID
	})
}

// History shouldn't have to know about "dist/" huh?
const (
	outDir       = "dist"
	imageDir     = "images"
	imageFileFmt = "thumb_%s.jpg"
)

func ImageFilePath(dbImgName string) string {
	return imageDir + "/" + fmt.Sprintf(imageFileFmt, dbImgName)
}

func CreateImageThumb(dbImgName string) {
	if dbImgName == "" {
		return
	}

	imageURL := ImageURLBase + fmt.Sprintf(ImageURLPathFmt, dbImgName)
	imageFileOutPath := outDir + "/" + ImageFilePath(dbImgName)

	if _, err := os.Stat(imageFileOutPath); err == nil {
		return
	}

	data, err := fetchURL(imageURL)
	if err != nil {
		log.Printf("fetch %s failed: %s\n", imageURL, err)
		return
	}

	decoded, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("jpeg.Decode %s failed: %s\n", imageURL, err)
		return
	}

	thumb := makeThumb(decoded)

	f, err := os.Create(imageFileOutPath)
	if err != nil {
		log.Printf("os.Create %s failed: %s\n", imageFileOutPath, err)
		return
	}
	defer f.Close()

	if err = jpeg.Encode(f, thumb, &jpeg.Options{Quality: 80}); err != nil {
		log.Printf("os.Create %s failed: %s\n", imageFileOutPath, err)
		return
	}

	log.Printf("downloaded %s\n", imageFileOutPath)
}

func fetchURL(url string) ([]byte, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response code not OK: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func makeThumb(decoded image.Image) *image.RGBA {
	const sideMaxLen = 130
	width := decoded.Bounds().Dx()
	height := decoded.Bounds().Dy()

	var thumb *image.RGBA

	if width <= sideMaxLen && height <= sideMaxLen {
		thumb = image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(thumb, thumb.Bounds(), decoded, decoded.Bounds().Min, draw.Over)
		return thumb
	}

	if height > width {
		// portrait
		thumb = image.NewRGBA(image.Rect(0, 0, width/(height/sideMaxLen), sideMaxLen))
	} else {
		// landscape
		thumb = image.NewRGBA(image.Rect(0, 0, sideMaxLen, height/(width/sideMaxLen)))
	}

	draw.BiLinear.Scale(thumb, thumb.Bounds(), decoded, decoded.Bounds(), draw.Over, nil)

	return thumb
}
