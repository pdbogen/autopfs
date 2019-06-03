package main

import (
	"encoding/json"
	"github.com/coreos/bbolt"
	"net/http"
)

func GetJob(db *bbolt.DB) func(rw http.ResponseWriter, req *http.Request) {
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

		rw.Header().Set("content-type", "application/json")
		rw.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(rw)
		if err := enc.Encode(job); err != nil {
			log.Errorf("encoding JSON for job %q: %s", job.JobId, err)
		}
	}
}
