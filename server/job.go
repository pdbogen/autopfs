package main

import (
	"fmt"
	"github.com/pdbogen/autopfs/paizo"
	"sync"
	"time"
	"github.com/coreos/bbolt"
	"encoding/json"
)

type Job struct {
	JobId    string
	State    string
	Sessions []*paizo.Session
	Email    string `json:"-"`
	Pass     string `json:"-"`
	Messages []string
}

const (
	JobStateDone = "done"
)

func (j Job) Done() bool {
	return j.State == JobStateDone
}

// Load loads a job from the given DB. If the job does not exist, both job and err will be nil. If anything else goes
// wrong, job will be nil and err will be non-nil. If the job exists and is loaded properly, job will be non-nil and
// err will be nil.
func Load(db *bolt.DB, jobId string) (job *Job, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		jobs := tx.Bucket([]byte("jobs"))
		if jobs == nil {
			return nil
		}

		jobJson := jobs.Get([]byte(jobId))
		if jobJson == nil {
			return nil
		}

		job = &Job{}
		if err := json.Unmarshal(jobJson, job); err != nil {
			log.Warningf("DB contained job with id %q, but could not parse: %v", err)
			return nil
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (j *Job) Save(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		jobs, err := tx.CreateBucketIfNotExists([]byte("jobs"))
		if err != nil {
			return fmt.Errorf("error opening jobs bucket: %v", err)
		}

		jsonBytes, err := json.Marshal(j)
		if err != nil {
			return fmt.Errorf("error marshaling job to JSON: %v", err)
		}

		if err := jobs.Put([]byte(j.JobId), jsonBytes); err != nil {
			return fmt.Errorf("saving job to DB: %v", err)
		}
		return nil
	})
}

func (j *Job) UpdateStatus(db *bolt.DB, status string, msg string) error {
	j.State = status
	j.Messages = append(j.Messages, time.Now().Format(time.ANSIC)+": "+msg)
	if err := j.Save(db); err != nil {
		return fmt.Errorf("updating job %q status: %v", j.JobId, err)
	}
	return nil
}

func (j *Job) Run(db *bolt.DB, wg *sync.WaitGroup) {
	defer wg.Done()
	if err := j.UpdateStatus(db, "login", "Logging in..."); err != nil {
		log.Error(err)
	}

	e, p := j.Email, j.Pass

	paizoSession, err := paizo.Login(e, p)
	if err != nil {
		if err := j.UpdateStatus(db, "error", "error logging in to Paizo: "+err.Error()); err != nil {
			log.Error(err)
		}
		return
	}

	if err := j.UpdateStatus(db, "player", "Getting player sessions..."); err != nil {
		log.Error(err)
	}

	ps, err := paizoSession.GetSessions(true, func(cur, total int) {
		if err := j.UpdateStatus(db, "player",
			fmt.Sprintf("Getting player sessions (%d/%d)...", cur, total),
		); err != nil {
			log.Error(err)
		}
	})

	if err != nil {
		log.Error("Getting sessions for job %q: %v", j.JobId, err)
		if ps == nil {
			if err := j.UpdateStatus(db, "error", "fatal error: "+err.Error()); err != nil {
				log.Error(err)
			}
			return
		}
		if err := j.UpdateStatus(db, j.State, "minor errors while parsing sessions: "+err.Error()); err != nil {
			log.Error(err)
		}
	}

	if err := j.UpdateStatus(db, "player", fmt.Sprintf("Got %d unique player scenarios", len(ps))); err != nil {
		log.Error(err)
	}
	if err := j.UpdateStatus(db, "gm", "Getting GM sessions..."); err != nil {
		log.Error(err)
	}

	gs, err := paizoSession.GetSessions(false, func(cur, total int) {
		if err := j.UpdateStatus(db, "gm", fmt.Sprintf("Getting GM sessions (%d/%d)...", cur, total)); err != nil {
			log.Error(err)
		}
	})

	if err != nil {
		log.Errorf("getting gm sessions for job %q: %v", j.JobId, err)
		if gs == nil {
			if err := j.UpdateStatus(db, "error", "fatal error: "+err.Error()); err != nil {
				log.Error(err)
			}
			return
		}
		if err := j.UpdateStatus(db, j.State, "GM Session Parse Errors (not a big deal): "+err.Error()); err != nil {
			log.Error(err)
		}
	}
	if err := j.UpdateStatus(db, "gm", fmt.Sprintf("Got %d unique GM scenarios", len(gs))); err != nil {
		log.Error(err)
	}
	j.Sessions = paizo.DeDupe(append(ps, gs...))
	if err := j.UpdateStatus(db, JobStateDone, fmt.Sprintf("Done! %d total unique scenarios", len(j.Sessions))); err != nil {
		log.Warningf("saving completed job %q: %v", j.JobId, err)
	}
}
