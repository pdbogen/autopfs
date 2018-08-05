package main

import (
	"encoding/csv"
	"fmt"
	log2 "github.com/pdbogen/autopfs/log"
	"github.com/pdbogen/autopfs/paizo"
	"math/rand"
	"net/http"
	"time"
	"github.com/coreos/bbolt"
	"flag"
	"os"
	"sync"
	"os/signal"
	"context"
	"github.com/op/go-logging"
)

var log = log2.Log

func Welcome(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("content-type", "text/html")
	fmt.Fprintf(rw, `
Welcome! Provide your <em>Paizo email and password</em> below to generate a CSV export of your adventures.<br/>
Your email address and password are never stored or logged by this system.<br/>
<form method=POST action=/begin>
  <input name=email placeholder="e-mail address"><br/>
  <input type=password name=password placeholder="password"><br/>
  <input type=submit><br/>
</form><br/>
This tool is open source. You're more than welcome to inspect the <a href="https://github.com/pdbogen/autopfs">Source Code</a> if that will help you trust it.<br/>
	`)
}

func Begin(db *bolt.DB, jobsWg *sync.WaitGroup) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {

		if err := req.ParseForm(); err != nil {
			http.Error(rw, "hmm, that request didn't look right. Go back and try again, perhaps?", http.StatusBadRequest)
			return
		}

		email := req.FormValue("email")
		pass := req.FormValue("password")

		if email == "" {
			http.Error(rw, "Sorry, email address is required. Go back and try again?", http.StatusBadRequest)
			return
		}

		if pass == "" {
			http.Error(rw, "Sorry, password is required. Go back and try again?", http.StatusBadRequest)
		}

		tokenBytes := make([]byte, 32)
		if n, err := rand.Read(tokenBytes); n != 32 {
			log.Errorf("could not generate token bytes: %s", err)
			http.Error(rw, msgInternalServerError, http.StatusInternalServerError)
			return
		}
		token := fmt.Sprintf("%x", tokenBytes)

		job := &Job{
			JobId:    token,
			State:    "init",
			Sessions: nil,
			Email:    email,
			Pass:     pass,
		}

		if err := job.Save(db); err != nil {
			log.Errorf("saving job %q to DB: %v", token, err)
			http.Error(rw, msgInternalServerError, http.StatusInternalServerError)
			return
		}

		jobsWg.Add(1)
		go job.Run(db, jobsWg)

		http.Redirect(rw, req, "/status?id="+token, http.StatusFound)
	}
}

func Status(db *bolt.DB) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			http.Error(rw, "hmm, that request didn't look right. Go back and try again, perhaps?", http.StatusBadRequest)
			return
		}

		id := req.FormValue("id")
		if id == "" {
			http.Error(rw, "Sorry; I can't get a request status without a request id.", http.StatusBadRequest)
			return
		}

		job, err := Load(db, id)
		if err != nil {
			log.Errorf("retrieving job %q from DB: %v", id, err)
			http.Error(rw, msgInternalServerError, http.StatusInternalServerError)
			return
		}

		if job == nil {
			http.Error(rw, "Sorry, I could not find that job.", http.StatusNotFound)
			return
		}

		if req.FormValue("view") == "" {
			if job.Done() {
				http.Redirect(rw, req, fmt.Sprintf("/html?id=%s", id), http.StatusFound)
				return
			} else {
				rw.Header().Add("Refresh", "1; url=/status?id="+id)
			}
		}

		rw.Header().Add("content-type", "text/html")
		if err := StatusTemplate.Execute(rw, map[string]interface{}{
			"Title": "Job Status",
			"Job":   job,
		}); err != nil {
			log.Errorf("rendering status template: %v", err)
		}
	}
}

func Html(db *bolt.DB) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {

		if err := req.ParseForm(); err != nil {
			http.Error(rw, "hmm, that request didn't look right. Go back and try again, perhaps?", http.StatusBadRequest)
			return
		}

		id := req.FormValue("id")
		if id == "" {
			http.Error(rw, "Sorry; I can't get a request status without a request id.", http.StatusBadRequest)
			return
		}

		job, err := Load(db, id)
		if err != nil {
			http.Error(rw, msgInternalServerError, http.StatusInternalServerError)
			return
		}
		if job == nil {
			http.NotFound(rw, req)
		}

		if !job.Done() {
			http.Redirect(rw, req, "/status?id="+id, http.StatusFound)
			return
		}

		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)

		rows := [][]string{}
		for _, s := range job.Sessions {
			rows = append(rows, s.Record())
		}
		err = HtmlTemplate.Execute(rw, map[string]interface{}{
			"Title":   "HTML View",
			"id":      job.JobId,
			"Headers": paizo.CsvHeader,
			"Rows":    rows,
		})
		if err != nil {
			log.Errorf("Executing HtmlTemplate: %v", err)
		}
	}
}

func Csv(db *bolt.DB) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {

		if err := req.ParseForm(); err != nil {
			http.Error(rw, "hmm, that request didn't look right. Go back and try again, perhaps?", http.StatusBadRequest)
			return
		}

		id := req.FormValue("id")
		if id == "" {
			http.Error(rw, "Sorry; I can't get a request status without a request id.", http.StatusBadRequest)
			return
		}

		job, err := Load(db, id)
		if err != nil {
			http.Error(rw, msgInternalServerError, http.StatusInternalServerError)
			return
		}
		if job == nil {
			http.NotFound(rw, req)
		}

		if !job.Done() {
			http.Redirect(rw, req, "/status?id="+id, http.StatusFound)
			return
		}

		rw.Header().Set("Content-Type", "text/csv")
		rw.Header().Set("Content-Disposition", "attachment;filename=sessions.csv")
		rw.WriteHeader(http.StatusOK)

		csvW := csv.NewWriter(rw)

		csvW.Write(paizo.CsvHeader)

		for _, s := range job.Sessions {
			csvW.Write(s.Record())
		}

		csvW.Flush()
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func handleSignals(stop chan<- bool, sigs <-chan os.Signal) {
	<-sigs
	close(stop)
}

func main() {
	port := flag.Int("port", 8080, "port to listen on for incoming connections")
	dbPath := flag.String("db-path", "sessions.db", "path to the bolt DB used to persist completed jobs")
	loglevel := flag.String("loglevel", "INFO", "set to DEBUG for more logging")
	flag.Parse()

	lvl, err := logging.LogLevel(*loglevel)
	if err != nil {
		log.Fatalf("could not parse log level %q: %s", *loglevel, err)
	}
	logging.SetLevel(lvl, log.Module)

	db, err := bolt.Open(*dbPath, os.FileMode(0640), bolt.DefaultOptions)

	if err != nil {
		log.Fatalf("could not open bolt DB %q: %v", *dbPath, err)
	}

	stop := make(chan bool)
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, os.Kill)

	go handleSignals(stop, signals)

	jobsWg := &sync.WaitGroup{}

	http.HandleFunc("/", Welcome)
	http.HandleFunc("/begin", Begin(db, jobsWg))
	http.HandleFunc("/status", Status(db))
	http.HandleFunc("/csv", Csv(db))
	http.HandleFunc("/html", Html(db))
	server := http.Server{Addr: fmt.Sprintf(":%d", *port)}
	go func() {
		log.Infof("Starting up on port %d", *port)
		log.Error(server.ListenAndServe())
	}()

	<-stop
	log.Infof("Shutting down...")
	server.Shutdown(context.Background())
	jobsWg.Wait()
	log.Infof("Shutdown complete. Bye!")
}
