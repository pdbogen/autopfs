package main

import (
	bolt "github.com/coreos/bbolt"
	"github.com/pdbogen/autopfs/paizo"
	"net/http"
)

func Html(db *bolt.DB, JsHash string, CssHash string) func(rw http.ResponseWriter, req *http.Request) {
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
			return
		}

		if !job.Done() {
			http.Redirect(rw, req, "/status?id="+id, http.StatusFound)
			return
		}

		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)

		err = TemplateRoot.ExecuteTemplate(rw, "html", map[string]interface{}{
			"Title":   "HTML View",
			"Desc":    req.FormValue("desc"),
			"id":      job.JobId,
			"Headers": paizo.CsvHeader,
			"JsHash":  JsHash,
			"CssHash": CssHash,
		})

		if err != nil {
			log.Errorf("Executing HtmlTemplate: %v", err)
		}
	}
}
