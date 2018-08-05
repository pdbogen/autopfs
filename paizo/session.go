package paizo

import (
	"fmt"
	"regexp"
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

var CsvHeader = []string{"Date", "Event Number", "Character Number", "Season", "Scenario Number", "Variant", "Scenario Name", "Player/GM"}

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
	if s.Player && s.GM {
		ret = append(ret, "P/GM")
	} else if s.Player {
		ret = append(ret, "P")
	} else {
		ret = append(ret, "GM")
	}
	return ret
}

var modules []*regexp.Regexp

func init() {
	for _, s := range moduleStrings {
		modules = append(modules, regexp.MustCompile(s))
	}
}

var s0s1Regex = regexp.MustCompile("^#([0-9]+):")

var variantRegex = regexp.MustCompile(`^([0-9]+)([^0-9]+)`)

// ParseName teases apart and stores interesting information from the scenario name- season number and scenarion number
// and stores it in the Session object. If an error occurs, error will be non-nil; some fields may be correctly
// populated; and the full raw scenario name will be saved in the ScenarioName field.
func (s *Session) ParseName(raw string) error {
	raw = strings.TrimSpace(raw)
	sn := raw

	if scenNum, ok := staticScenarioNumbers[sn]; ok {
		s.Season = scenNum[0]
		s.Number = scenNum[1]
		s.ScenarioName = sn
		return nil
	}

	if s0s1 := s0s1Regex.FindStringSubmatch(sn); s0s1 != nil {
		num := s0s1[1]
		season, err := strconv.Atoi(num)
		if err == nil {
			s.Season = season / 29
			s.Number = season
			s.ScenarioName = sn[strings.Index(sn, " ")+1:]
			return nil
		}
	}

	s.Season = -1
	s.Number = -1

	if sn[0] != '#' {
		s.ScenarioName = raw
		for _, re := range modules {
			if re.MatchString(raw) {
				return nil
			}
		}
		return fmt.Errorf("no parser or static record for %q", raw)
	}

	sn = strings.TrimLeft(sn, "#")
	term := strings.IndexAny(sn, "-–")
	if term == -1 {
		s.ScenarioName = raw
		return fmt.Errorf("parsing %q: no season terminator", raw)
	}

	season, err := strconv.Atoi(sn[0:term])
	if err != nil {
		s.ScenarioName = raw
		return fmt.Errorf("parsing %q: could not parse %q as number: %s", raw, sn[0:term], err)
	}
	s.Season = season

	sn = strings.TrimLeft(sn[term:], "-–")
	term = strings.IndexAny(sn, ":— ")
	if term == -1 {
		s.ScenarioName = raw
		return fmt.Errorf("parsing %q: could not find scenario terminator", raw)
	}

	if variant := variantRegex.FindStringSubmatch(sn[0:term]); variant != nil {
		s.Variant = variant[2]
		term -= len(variant[2])
	}

	number, err := strconv.Atoi(sn[0:term])
	if err != nil {
		s.ScenarioName = raw
		return fmt.Errorf("parsing %q: could not parse %q as scenario number: %s", raw, sn[0:term], err)
	}

	s.Number = number
	s.ScenarioName = strings.TrimLeft(sn[term+len(s.Variant):], ":— ")
	return nil
}

// sessionFromCells converts a list of cells (a 12-long string slice corresponding to the columnar format on the paizo
// sessions page) to a hydrated Session object. The first cell is handled specially, and is intended to be an RFC3339
// time string, which is retrieved from the `datetime` attribute of the `time` object that occupies the first cell.
// most parse errors will return a non-nil error and a partially hydrated session object, but some parse errors will
// return a `nil` session object.
func sessionFromCells(cells []string, player bool) (*Session, error) {
	if len(cells) < 7 {
		return nil, fmt.Errorf("expected >=7 elements in cells, received %d", len(cells))
	}
	ret := &Session{}

	t, err := time.Parse(time.RFC3339, cells[0])
	if err != nil {
		return ret, fmt.Errorf("expected first cell to be RFC3339 date, but could not parse %q: %s", cells[0], err)
	}
	ret.Date = t

	err = ret.ParseName(cells[5])
	if err != nil {
		return ret, fmt.Errorf("expected sixth cell to be scenario name, but could not parse %q: %s", cells[5], err)
	}

	evNumStr := cells[1]
	evNum, err := strconv.ParseInt(evNumStr, 10, 64)
	if err != nil {
		return ret, fmt.Errorf("expected second cell to be event number, but could not parse %q: %s", evNumStr, err)
	}
	ret.EventNumber = append(ret.EventNumber, evNum)

	if player {
		charNumStr := cells[6]
		charNumDash := strings.Index(charNumStr, "-")
		if charNumDash == -1 {
			return ret, fmt.Errorf("expected seventh cell to contain character number, but %q did not contain dash", charNumStr)
		}
		charNumPart := strings.TrimLeft(charNumStr[charNumDash:], "-")
		if charNumPart == "" {
			ret.Character = append(ret.Character, -1)
		} else {
			charNum, err := strconv.Atoi(charNumPart)
			if err != nil {
				return ret, fmt.Errorf("in seventh cell %q, could not parse character number part %q: %s", charNumStr, charNumPart, err)
			}
			ret.Character = append(ret.Character, charNum)
			if charNum >= 700 {
				ret.Game = "Starfinder"
			} else {
				ret.Game = "Pathfinder"
			}
		}
		ret.Player = true
	} else {
		ret.Character = append(ret.Character, -2)
		ret.GM = true
	}

	return ret, nil
}

type SessionsByDate []*Session

func (s SessionsByDate) Len() int {
	return len(s)
}

func (s SessionsByDate) Less(a, b int) bool {
	return s[a].Date.Before(s[b].Date)
}

func (s SessionsByDate) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

var _ sort.Interface = (SessionsByDate)(nil)

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
	sort.Sort(SessionsByDate(out))
	return out
}
