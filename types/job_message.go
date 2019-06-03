package types

import "time"

type JobMessage struct {
	JobId   string `json:"-"`
	Time    time.Time
	Message string
	State   string
}
