package main

import (
	"encoding/csv"
	"fmt"
	log2 "github.com/pdbogen/autopfs/log"
	"github.com/pdbogen/autopfs/paizo"
	"math/rand"
	"net/http"
	"sync"
	"time"
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

func Begin(rw http.ResponseWriter, req *http.Request) {
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
		http.Error(rw, "sorry! something went wrong. please let pdbogen at cernu dot us know via email", http.StatusInternalServerError)
		return
	}
	token := fmt.Sprintf("%x", tokenBytes)

	jobsMu.Lock()
	job := &Job{
		State:    "init",
		Sessions: nil,
		Email:    email,
		Pass:     pass,
		Mu:       &sync.RWMutex{},
	}
	jobs[token] = job
	jobsMu.Unlock()

	go job.Run()

	http.Redirect(rw, req, "/status?id="+token, http.StatusFound)
}

func Status(rw http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(rw, "hmm, that request didn't look right. Go back and try again, perhaps?", http.StatusBadRequest)
		return
	}

	id := req.FormValue("id")
	if id == "" {
		http.Error(rw, "Sorry; I can't get a request status without a request id.", http.StatusBadRequest)
		return
	}

	jobsMu.RLock()
	defer jobsMu.RUnlock()
	job, ok := jobs[id]
	if !ok {
		http.NotFound(rw, req)
		return
	}
	job.Mu.RLock()
	defer job.Mu.RUnlock()

	rw.Header().Add("content-type", "text/html")

	if job.State == "Done" {
		rw.WriteHeader(http.StatusOK)
		fmt.Fprintf(rw, "Done! Click <a href='/csv?id=%s'>here</a> to download your sessions. Feel free to share this link!\n<br/>", id)
	} else {
		rw.Header().Add("Refresh", "1; url=/status?id="+id)
		rw.WriteHeader(http.StatusOK)
		fmt.Fprintf(rw, "This can potentially take a few minutes. Please hold tight.<br/>\nStatus: %s<br/>", job.State)
	}

	if len(job.Messages) > 0 {
		fmt.Fprint(rw, "Job Log:<br/><ul>\n")
		for _, msg := range job.Messages {
			fmt.Fprintf(rw, "<li><pre>%s</pre></li>\n", msg)
		}
		fmt.Fprint(rw, "</ul>\n")
	}
}

func Csv(rw http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(rw, "hmm, that request didn't look right. Go back and try again, perhaps?", http.StatusBadRequest)
		return
	}

	id := req.FormValue("id")
	if id == "" {
		http.Error(rw, "Sorry; I can't get a request status without a request id.", http.StatusBadRequest)
		return
	}
	jobsMu.RLock()
	defer jobsMu.RUnlock()
	job, ok := jobs[id]
	if !ok {
		http.NotFound(rw, req)
		return
	}
	job.Mu.RLock()
	defer job.Mu.RUnlock()

	if job.State != "Done" {
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
	return
}

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/", Welcome)
	http.HandleFunc("/begin", Begin)
	http.HandleFunc("/status", Status)
	http.HandleFunc("/csv", Csv)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
