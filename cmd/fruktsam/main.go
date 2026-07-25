package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/fruktkartan/fruktsam/internal/history"
	"github.com/fruktkartan/fruktsam/internal/trees"
	"github.com/fruktkartan/fruktsam/internal/util"
	"github.com/joho/godotenv"
)

const (
	envfile = ".env"
	outfile = "dist/index.html"
)

const defaultSinceDays = 90

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
	var err error

	app := kingpin.New("fruktsam", "Generate html from Fruktkartan edit history")
	app.Flag("since", fmt.Sprintf("How many days back, default: %d", defaultSinceDays)).
		PlaceHolder("DAYS").IntVar(&sinceFlag)
	app.HelpFlag.Short('h')
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if err = godotenv.Load(envfile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed load file %s: %w", envfile, err)
	}

	var data templateData
	data.DatabaseName, err = getDatabaseName(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	data.Now = util.FormatDateTime(time.Now())

	if err = data.Trees.FromDB(); err != nil {
		return fmt.Errorf("failed Trees.FromDB: %w", err)
	}
	fmt.Printf("Trees: %d\n", data.Trees.Count())

	if err = data.History.FromDB(sinceFlag); err != nil {
		return fmt.Errorf("failed History.FromDB: %w", err)
	}
	fmt.Printf("History entries during past %d days: %d\n", sinceFlag, data.History.Count())

	tmpl, err := template.ParseFiles("tmpl_index.html")
	if err != nil {
		return fmt.Errorf("failed template ParseFiles: %w", err)
	}

	var f *os.File
	if err = os.MkdirAll(filepath.Dir(outfile), 0o770); err != nil {
		return fmt.Errorf("failed MkdirAll: %w", err)
	}
	if f, err = os.Create(outfile); err != nil {
		return fmt.Errorf("failed Create: %w", err)
	}
	if err = tmpl.Execute(f, &data); err != nil {
		return fmt.Errorf("failed template Execute: %w", err)
	}

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
