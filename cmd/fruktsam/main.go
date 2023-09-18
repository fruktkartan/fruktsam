package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/fruktkartan/fruktsam/history"
	"github.com/fruktkartan/fruktsam/trees"
	"github.com/fruktkartan/fruktsam/util"
	"github.com/joho/godotenv"
)

const (
	envfile = ".env"
	outfile = "dist/index.html"
)

const defaultSinceDays = 90

type templateData struct {
	History history.History
	Now     string
	Trees   trees.Trees
}

func main() {
	sinceFlag := defaultSinceDays
	var err error

	app := kingpin.New("fruktsam", "Generate html from Fruktkartan edit history")
	app.Flag("since", fmt.Sprintf("How many days back, default: %d", defaultSinceDays)).
		PlaceHolder("DAYS").IntVar(&sinceFlag)
	app.HelpFlag.Short('h')
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if err = godotenv.Load(envfile); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading %s file: %s", envfile, err)
	}

	var data templateData

	data.Now = util.FormatDateTime(time.Now())
	if err = data.History.FromDB(sinceFlag); err != nil {
		log.Fatalf("FromDB: %s", err)
	}

	fmt.Printf("History entries during past %d days: %d\n", sinceFlag, data.History.Len())

	tmpl, err := template.ParseFiles("tmpl_index.html")
	if err != nil {
		log.Fatal(err)
	}

	var f *os.File
	if err = os.MkdirAll(filepath.Dir(outfile), 0o770); err != nil {
		log.Fatal(err)
	}
	if f, err = os.Create(outfile); err != nil {
		log.Fatal(err)
	}
	if err = tmpl.Execute(f, &data); err != nil {
		log.Fatal(err)
	}
}
