package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/NYTimes/gziphandler"
	bolt "github.com/coreos/bbolt"
	"github.com/lpar/gzipped"
	"github.com/op/go-logging"
	log2 "github.com/pdbogen/autopfs/log"
	"github.com/pdbogen/autopfs/paizo"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

var log = log2.Log

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

var JsHash string
var CssHash string

const JsFile = "js/autopfs.js"
const CssFile = "css/autopfs.css"

func hash(file string) string {
	f, err := assets.Open(file)
	if err != nil {
		log.Fatalf("could not open %q: %s", file, err)
	}
	defer f.Close()

	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatalf("hashing %q: %s", file, err)
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func init() {
	rand.Seed(time.Now().UnixNano())
	JsHash = hash(JsFile)
	CssHash = hash(CssFile)
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

	http.HandleFunc("/", IndexController(db, JsHash, CssHash))
	http.Handle("/static/", http.StripPrefix("/static/", gzipped.FileServer(assets)))
	http.HandleFunc("/begin", Begin(db, jobsWg))
	http.HandleFunc("/status", Status(db, JsHash, CssHash, false))
	http.HandleFunc("/status/ws", Status(db, JsHash, CssHash, true))
	http.HandleFunc("/csv", Csv(db))
	http.HandleFunc("/html", Html(db, JsHash, CssHash))
	http.Handle("/json", gziphandler.GzipHandler(http.HandlerFunc(GetJob(db))))
	server := http.Server{Addr: fmt.Sprintf(":%d", *port)}
	go func() {
		log.Infof("Starting up on port %d", *port)
		log.Error(server.ListenAndServe())
	}()

	<-stop
	log.Infof("Shutting down...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Errorf("during shutdown: %s", err)
	}
	jobsWg.Wait()
	log.Infof("Shutdown complete. Bye!")
}
