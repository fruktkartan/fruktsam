package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"log/slog"
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
	envFile = ".env"
	outFile = "index.html"
)

// Logging setup with levels, based on slog bridge to classic log.
// Gives us simple output (not slog `time=... level=FOO msg="..."`).
// But we have to keep track of current level ourselves.
var logLevel slog.Level

func setLogLevel(level slog.Level) {
	logLevel = level
	slog.SetLogLoggerLevel(level)
}

//go:embed tmpl_index.html
var templates embed.FS

type templateData struct {
	History      history.History
	Now          string
	DatabaseName string
	Trees        trees.Trees
}

func main() {
	setLogLevel(slog.LevelInfo)
	log.SetFlags(0)
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func run() error {
	sinceFlag := 90
	destDirFlag := "dist"
	quiet := false

	app := kingpin.New("fruktsam", "Generate html from Fruktkartan edit history")
	app.Flag("since", fmt.Sprintf("How many days back, default: %d", sinceFlag)).
		PlaceHolder("DAYS").IntVar(&sinceFlag)
	app.Flag("dest", fmt.Sprintf("Destination directory, default: %s", destDirFlag)).
		PlaceHolder("DIRECTORY").Short('d').StringVar(&destDirFlag)
	app.Flag("quiet", fmt.Sprintf("Be quiet, output only warnings, errors and worse, default: %t", quiet)).
		Short('q').BoolVar(&quiet)
	app.HelpFlag.Short('h')
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if quiet {
		setLogLevel(slog.LevelWarn)
	}

	if !path.IsAbs(destDirFlag) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed Getwd: %w", err)
		}
		destDirFlag = filepath.Join(cwd, destDirFlag)
	}

	if err := godotenv.Load(envFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed load file %s: %w", envFile, err)
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
	slog.Info(fmt.Sprintf("Trees: %d", data.Trees.Count()))

	if err = data.History.FromDB(sinceFlag, destDirFlag); err != nil {
		return fmt.Errorf("failed History.FromDB: %w", err)
	}
	slog.Info(fmt.Sprintf("History entries during past %d days: %d", sinceFlag, data.History.Count()))

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
	slog.Info(fmt.Sprintf("Wrote %s", outFile))

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
