package main

import (
	"github.com/coreos/bbolt"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

func Status(db *bbolt.DB, JsHash, CssHash string, websocket bool) func(rw http.ResponseWriter, req *http.Request) {
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

		if !websocket {
			statusPage(job, JsHash, CssHash, rw, req)
			return
		}

		statusWebsocket(job, rw, req)
	}
}

func statusPage(job *Job, JsHash, CssHash string, rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("content-type", "text/html")
	if err := TemplateRoot.ExecuteTemplate(rw, "status", map[string]interface{}{
		"Title":   "Job Status",
		"Job":     job,
		"JsHash":  JsHash,
		"CssHash": CssHash,
	}); err != nil {
		log.Errorf("rendering status template: %v", err)
	}
}

// statusWebsocket upgrades the request to a websocket, over which it sends any new messages after `since`
func statusWebsocket(job *Job, rw http.ResponseWriter, req *http.Request) {
	sinceStr := req.FormValue("since")

	// discard the error, we'll just get zero time instead.
	since, _ := time.Parse(time.RFC3339Nano, sinceStr)

	conn, err := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}).Upgrade(rw, req, nil)

	if err != nil {
		log.Errorf("upgrading HTTP request to websocket: %s", err)
		return
	}

	stream := Subscribe(job.JobId)
	defer stream.Unsubscribe()

	last := job.Messages[len(job.Messages)-1].Time
	for _, message := range job.Messages {
		if message.Time.Before(since) {
			continue
		}
		log.Debugf("sending backfill message %+v", message)
		if err := conn.WriteJSON(message); err != nil {
			log.Error("writing to websocket: %s", err)
			return
		}
	}

	for message := range stream.Chan {
		if message.Time.Before(last) {
			continue
		}
		log.Debugf("sending message %+v", message)
		if err := conn.WriteJSON(message); err != nil {
			log.Error("writing to websocket: %s", err)
			return
		}
	}
}
