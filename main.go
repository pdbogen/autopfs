package main

import (
	"encoding/csv"
	"flag"
	"github.com/op/go-logging"
	"github.com/pdbogen/autopfs/paizo"
	"os"
)

func main() {
	email := flag.String("email", "", "address to use for paizo sign in")
	pass := flag.String("password", "", "password to use for paizo sign in")
	loglevel := flag.String("loglevel", "info", "set to DEBUG for more logging, or INFO or ERROR for less")
	out := flag.String("out", "sessions.csv", "file to which CSV-formatted results should be saved")
	flag.Parse()

	lvl, err := logging.LogLevel(*loglevel)
	if err != nil {
		log.Fatalf("could not parse log level %q: %s", *loglevel, err)
	}
	logging.SetLevel(lvl, log.Module)

	pzo, err := paizo.Login(*email, *pass)
	if err != nil {
		log.Fatalf("during login: %s", err)
	}

	psessions, err := pzo.GetSessions(true)
	if err != nil {
		if psessions == nil {
			log.Fatalf("retrieving player sessions: %s", err)
		} else {
			log.Errorf("retrieving player sessions: %s", err)
		}
	}
	log.Infof("got %d player sessions", len(psessions))

	gsessions, err := pzo.GetSessions(false)
	if err != nil {
		if psessions == nil {
			log.Fatalf("retrieving GM sessions: %s", err)
		} else {
			log.Errorf("retrieving GM sessions: %s", err)
		}
	}
	log.Infof("got %d GM sessions", len(gsessions))

	sessions := paizo.DeDupe(append(psessions, gsessions...))

	log.Infof("Writing %d sessions out to %q...", len(sessions), *out)
	outFile, err := os.OpenFile(*out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		log.Fatalf("opening %q for writing: %s", *out, err)
	}
	outW := csv.NewWriter(outFile)
	outW.Write(paizo.CsvHeader)
	for _, session := range sessions {
		outW.Write(session.Record())
	}
	outW.Flush()
	outFile.Close()
}
