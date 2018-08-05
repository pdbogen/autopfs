package main

import (
	"fmt"
	"github.com/pdbogen/autopfs/paizo"
	"sync"
	"time"
)

type Job struct {
	State    string
	Sessions []*paizo.Session
	Email    string
	Pass     string
	Mu       *sync.RWMutex
	Messages []string
}

func (j *Job) Message(msg string) {
	j.Messages = append(j.Messages,
		time.Now().Format(time.ANSIC)+": "+msg)
}

func (j *Job) Run() {
	j.Mu.Lock()
	j.State = "login"
	j.Message("Logging in...")
	e, p := j.Email, j.Pass
	j.Mu.Unlock()

	paizoSession, err := paizo.Login(e, p)
	if err != nil {
		j.Mu.Lock()
		j.State = fmt.Sprintf("error")
		j.Message("Error: " + err.Error())
		j.Mu.Unlock()
		return
	}

	j.Mu.Lock()
	j.State = "player"
	j.Message("Getting player sessions...")
	j.Mu.Unlock()

	ps, err := paizoSession.GetSessions(true)
	if err != nil {
		if ps == nil {
			j.Mu.Lock()
			j.State = "error"
			j.Message("fatal error: " + err.Error())
			j.Mu.Unlock()
			return
		}
		j.Mu.Lock()
		j.Message("Player Session Parse Errors (not a big deal): " + err.Error())
		j.Mu.Unlock()
	}

	j.Mu.Lock()
	j.Message(fmt.Sprintf("Got %d player sessions", len(ps)))
	j.State = "gm"
	j.Message("Getting GM sessions...")
	j.Mu.Unlock()

	gs, err := paizoSession.GetSessions(false)
	if err != nil {
		if gs == nil {
			j.Mu.Lock()
			j.State = "error"
			j.Message("fatal error: " + err.Error())
			j.Mu.Unlock()
			return
		}
		j.Mu.Lock()
		j.Message("GM Session Parse Errors (not a big deal): " + err.Error())
		j.Mu.Unlock()
	}

	j.Mu.Lock()
	j.Sessions = paizo.DeDupe(append(ps, gs...))
	j.State = "Done"
	j.Message("Done!")
	j.Mu.Unlock()
}

var jobs = map[string]*Job{}
var jobsMu = &sync.RWMutex{}
