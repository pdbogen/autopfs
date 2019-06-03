package main

import (
	"encoding/json"
	"fmt"
	bolt "github.com/coreos/bbolt"
	"github.com/pdbogen/autopfs/paizo"
	"github.com/pdbogen/autopfs/types"
	"sort"
	"sync"
	"time"
)

type JobMessageSubscription struct {
	Id   int
	Send chan<- types.JobMessage
	Recv <-chan types.JobMessage
}

type Job struct {
	types.Job
	Subscriptions   []*JobMessageSubscription `json:"-"`
	SubscriptionsMu *sync.Mutex               `json:"-"`
}

// Load loads a job from the given DB. If the job does not exist, both job and err will be nil. If anything else goes
// wrong, job will be nil and err will be non-nil. If the job exists and is loaded properly, job will be non-nil and
// err will be nil.
func Load(db *bolt.DB, jobId string) (job *Job, err error) {
	jobs, err := LoadMany(db, []string{jobId})
	if err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, nil
	}
	return jobs[0], nil
}

// LoadMany loads and returns whichever named job IDs exist in the database.
// This means that jobs might well be empty. An error is returned only for
// cases where there are DB issues. Unparseable jobs are treated as
// nonexistent.
func LoadMany(db *bolt.DB, jobIds []string) (jobs []*Job, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		jobsBucket := tx.Bucket([]byte("jobs"))
		if jobsBucket == nil {
			return nil
		}

		for _, jobId := range jobIds {
			jobJson := jobsBucket.Get([]byte(jobId))
			if jobJson == nil {
				continue
			}

			job := &Job{
				SubscriptionsMu: &sync.Mutex{},
			}
			if err := json.Unmarshal(jobJson, job); err != nil {
				log.Warningf("DB contained job with id %q, but could not parse: %v", err)
				continue
			}

			for _, j := range job.Messages {
				j.JobId = job.JobId
			}

			for _, sess := range job.Sessions {
				sort.Slice(sess.EventNumber, func(i, j int) bool {
					return sess.EventNumber[i] < sess.EventNumber[j]
				})
				sort.Ints(sess.Character)
			}
			jobs = append(jobs, job)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return jobs, nil
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
	jobMsg := &types.JobMessage{
		JobId:   j.JobId,
		State:   status,
		Time:    time.Now(),
		Message: msg,
	}

	j.Messages = append(j.Messages, jobMsg)
	if err := j.Save(db); err != nil {
		return fmt.Errorf("updating job %q status: %v", j.JobId, err)
	}

	Publish(jobMsg)
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

	if err := j.UpdateStatus(db, "sessions", "Getting player sessions..."); err != nil {
		log.Error(err)
	}

	if err := j.UpdateStatus(db, "sessions", "Getting characters..."); err != nil {
		log.Error(err)
	}
	chars, err := paizoSession.GetCharacters()

	if err := j.UpdateStatus(db, "sessions", fmt.Sprintf("Got %d characters.", len(chars))); err != nil {
		log.Error(err)
	}

	if err != nil {
		if err := j.UpdateStatus(db, "error", "fatal error getting characters: "+err.Error()); err != nil {
			log.Errorf("updating job status: %q", err)
		}
		return
	}
	j.Characters = chars

	ps, gs, err := paizoSession.GetSessions(j.Characters, func(cur, total int) {
		if err := j.UpdateStatus(db, "sessions",
			fmt.Sprintf("Getting sessions (%d/%d)...", cur, total),
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

	if err := j.UpdateStatus(db, "gm", fmt.Sprintf("Got %d unique GM scenarios", len(gs))); err != nil {
		log.Error(err)
	}

	j.Sessions = types.DeDupe(append(ps, gs...))
	if err := j.UpdateStatus(db, types.JobStateDone, fmt.Sprintf("Done! %d total unique scenarios", len(j.Sessions))); err != nil {
		log.Warningf("saving completed job %q: %v", j.JobId, err)
	}
}
