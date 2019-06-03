package types

import (
	"time"
)

type Job struct {
	JobId      string
	State      string
	Sessions   []*Session
	Email      string `json:"-"`
	Pass       string `json:"-"`
	Messages   []*JobMessage
	JobDate    time.Time
	Characters []Character
}

const (
	JobStateDone = "done"
)

func (j Job) Done() bool {
	return j.State == JobStateDone
}
