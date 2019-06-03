package main

import (
	"github.com/pdbogen/autopfs/types"
	"sync"
)

type Subscription struct {
	JobId string
	Id    int
	Chan  chan *types.JobMessage
}

var Subscriptions = map[string][]Subscription{}
var SubscriptionsMu = &sync.RWMutex{}

func Publish(jm *types.JobMessage) {
	SubscriptionsMu.RLock()
	defer SubscriptionsMu.RUnlock()
	for _, sub := range Subscriptions[jm.JobId] {
		go func() { sub.Chan <- jm }()
	}
}

func Subscribe(jobId string) Subscription {
	SubscriptionsMu.Lock()
	defer SubscriptionsMu.Unlock()

	ch := make(chan *types.JobMessage)
	newSub := Subscription{JobId: jobId, Chan: ch}
	for _, sub := range Subscriptions[jobId] {
		if sub.Id > newSub.Id {
			newSub.Id = sub.Id + 1
		}
	}

	Subscriptions[jobId] = append(Subscriptions[jobId], newSub)

	return newSub
}

func (s Subscription) Unsubscribe() {
	SubscriptionsMu.Lock()
	defer SubscriptionsMu.Unlock()
	for i, sub := range Subscriptions[s.JobId] {
		if s.Id == sub.Id {
			close(s.Chan)
			Subscriptions[s.JobId] = append(Subscriptions[s.JobId][:i], Subscriptions[s.JobId][i+1:]...)
			return
		}
	}
}
