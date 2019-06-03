package types

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Session struct {
	Date         time.Time
	EventNumber  []int64
	Game         string
	Season       int
	Number       int
	Variant      string
	ScenarioName string
	Character    []int
	Player       bool
	GM           bool
}

func (s Session) String() (ret string) {
	ret = fmt.Sprintf("%s\t%d\t", s.Date.Format("2006-01-02"), s.EventNumber)
	if s.Season >= 0 {
		ret += fmt.Sprintf("%d-%02d%s\t", s.Season, s.Number, s.Variant)
	} else {
		ret += "N/A \t"
	}
	characters := []string{}
	for _, char := range s.Character {
		if char == -2 {
			characters = append(characters, "GM")
		} else {
			characters = append(characters, strconv.Itoa(char))
		}
	}
	ret += strings.Join(characters, ",")
	ret += s.ScenarioName
	return
}

func (s Session) Record() (ret []string) {
	events := []string{}
	for _, e := range s.EventNumber {
		events = append(events, strconv.FormatInt(e, 10))
	}
	characters := []string{}
	for _, char := range s.Character {
		if char == -2 {
			characters = append(characters, "GM")
		} else {
			characters = append(characters, strconv.Itoa(char))
		}
	}
	ret = []string{
		s.Date.Format("2006-01-02"),
		strings.Join(events, " "),
		strings.Join(characters, " "),
		strconv.Itoa(s.Season),
		strconv.Itoa(s.Number),
		s.Variant,
		s.ScenarioName,
	}
	if s.Date.IsZero() {
		ret[0] = "MISSING"
	}
	if s.Player && s.GM {
		ret = append(ret, "P/GM")
	} else if s.Player {
		ret = append(ret, "P")
	} else {
		ret = append(ret, "GM")
	}
	return ret
}

func DeDupe(in []*Session) (out []*Session) {
	sessionsByName := map[string]*Session{}
	for _, session := range in {
		previous, ok := sessionsByName[session.ScenarioName]
		if ok {
			previous.Character = append(previous.Character, session.Character...)
			previous.EventNumber = append(previous.EventNumber, session.EventNumber...)
			previous.Player = previous.Player || session.Player
			previous.GM = previous.GM || session.GM
			if previous.Game == "" {
				previous.Game = session.Game
			}
		} else {
			sessionsByName[session.ScenarioName] = session
		}
	}

	out = []*Session{}
	for _, session := range sessionsByName {
		out = append(out, session)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Date.Before(out[j].Date)
	})
	return out
}
