package main

import (
	"github.com/coreos/bbolt"
	"net/http"
	"sort"
	"github.com/pdbogen/autopfs/paizo"
)

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

		sortCol := req.FormValue("sort")
		switch sortCol {
		case "": // default
		case "Date": // default
		default:
			col := 0
			for idx, column := range paizo.CsvHeader {
				if column == sortCol {
					col = idx
					break
				}
			}
			sort.Slice(rows, func(i, j int) bool {
				return rows[i][col] < rows[j][col]
			})
		}

		if req.FormValue("desc") != "" {
			for i := 0; i < len(rows)/2; i++ {
				rows[i], rows[len(rows)-1-i] = rows[len(rows)-1-i], rows[i]
			}
		}

		err = HtmlTemplate.Execute(rw, map[string]interface{}{
			"Title":   "HTML View",
			"Sort":    sortCol,
			"Desc":    req.FormValue("desc"),
			"id":      job.JobId,
			"Headers": paizo.CsvHeader,
			"Rows":    rows,
		})
		if err != nil {
			log.Errorf("Executing HtmlTemplate: %v", err)
		}
	}
}
