package main

import (
	"github.com/coreos/bbolt"
	"net/http"
	"strings"
)

func IndexController(db *bbolt.DB, JsHash, CssHash string) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		context := map[string]interface{}{
			"JsHash":  JsHash,
			"CssHash": CssHash,
		}
		histories, _ := req.Cookie("history")
		var jobs []*Job
		if histories != nil {
			var err error
			historyIdList := strings.Split(histories.Value, ",")
			jobs, err = LoadMany(db, historyIdList)
			if err != nil {
				log.Errorf("loading jobs: %s", err)
			}
			context["jobs"] = jobs
		}

		var historyIdList []string
		for _, job := range jobs {
			historyIdList = append(historyIdList, job.JobId)
		}

		http.SetCookie(rw, &http.Cookie{
			Value: strings.Join(historyIdList, ","),
			Name:  "history",
		})
		rw.Header().Add("content-type", "text/html")
		rw.WriteHeader(http.StatusOK)
		if err := TemplateRoot.ExecuteTemplate(rw, "index", context); err != nil {
			log.Errorf("writing index template: %s", err)
		}
	}
}
