package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/fruktkartan/fruktsam/internal/history"
	"github.com/fruktkartan/fruktsam/internal/trees"
	"github.com/fruktkartan/fruktsam/internal/util"
	"github.com/google/renameio/v2"
	"github.com/joho/godotenv"
)

const (
	envfile          = ".env"
	outFile          = "index.html"
	defaultSinceDays = 90
	defaultDestDir   = "dist"
)

//go:embed tmpl_index.html
var templates embed.FS

type templateData struct {
	History      history.History
	Now          string
	DatabaseName string
	Trees        trees.Trees
}

func main() {
	if err := run(); err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func run() error {
	sinceFlag := defaultSinceDays
	destDirFlag := defaultDestDir

	app := kingpin.New("fruktsam", "Generate html from Fruktkartan edit history")
	app.Flag("since", fmt.Sprintf("How many days back, default: %d", defaultSinceDays)).
		PlaceHolder("DAYS").IntVar(&sinceFlag)
	app.Flag("dest", fmt.Sprintf("Destination directory, default: %s", defaultDestDir)).
		PlaceHolder("DIRECTORY").Short('d').StringVar(&destDirFlag)
	app.HelpFlag.Short('h')
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if !path.IsAbs(destDirFlag) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed Getwd: %w", err)
		}
		destDirFlag = filepath.Join(cwd, destDirFlag)
	}

	if err := godotenv.Load(envfile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed load file %s: %w", envfile, err)
	}

	var data templateData
	var err error
	data.DatabaseName, err = getDatabaseName(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	data.Now = util.FormatDateTime(time.Now())

	if err = data.Trees.FromDB(); err != nil {
		return fmt.Errorf("failed Trees.FromDB: %w", err)
	}
	log.Printf("Trees: %d", data.Trees.Count())

	if err = data.History.FromDB(sinceFlag, destDirFlag); err != nil {
		return fmt.Errorf("failed History.FromDB: %w", err)
	}
	log.Printf("History entries during past %d days: %d", sinceFlag, data.History.Count())

	tmpl, err := template.ParseFS(templates, "tmpl_index.html")
	if err != nil {
		return fmt.Errorf("failed template ParseFS: %w", err)
	}

	if err = os.MkdirAll(destDirFlag, 0o755); err != nil {
		return fmt.Errorf("failed MkdirAll: %w", err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, &data); err != nil {
		return fmt.Errorf("failed template Execute: %w", err)
	}

	outFile := filepath.Join(destDirFlag, outFile)
	if err = renameio.WriteFile(outFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed WriteFile: %w", err)
	}
	log.Printf("Wrote %s", outFile)

	return nil
}

func getDatabaseName(dbURL string) (string, error) {
	if dbURL == "" {
		return "", fmt.Errorf("env variable DATABASE_URL is empty")
	}

	// split postgres://user:pass:word@example.com:port/dbname
	parts := strings.Split(dbURL, "/")
	if len(parts) != 4 {
		return "", fmt.Errorf("DATABASE_URL: expected 4 /-separated parts")
	}

	return parts[3], nil
}
