package main

import (
	"crypto/rand"
	"fmt"
	"github.com/coreos/bbolt"
	"github.com/pdbogen/autopfs/types"
	"net/http"
	"sync"
)

func Begin(db *bbolt.DB, jobsWg *sync.WaitGroup) func(rw http.ResponseWriter, req *http.Request) {
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
			Job: types.Job{
				JobId:    token,
				State:    "init",
				Sessions: nil,
				Email:    email,
				Pass:     pass,
			},
			SubscriptionsMu: &sync.Mutex{},
		}

		if err := job.Save(db); err != nil {
			log.Errorf("saving job %q to DB: %v", token, err)
			http.Error(rw, msgInternalServerError, http.StatusInternalServerError)
			return
		}

		jobsWg.Add(1)
		go job.Run(db, jobsWg)

		histories := ""
		history, _ := req.Cookie("history")
		if history != nil {
			histories = history.Value + ","
		}

		http.SetCookie(rw, &http.Cookie{
			Value: histories,
			Name:  "history",
		})
		http.Redirect(rw, req, "/status?id="+token, http.StatusFound)
	}
}
