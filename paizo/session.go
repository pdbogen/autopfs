package paizo

import (
	"fmt"
	"github.com/pdbogen/autopfs/types"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Session struct {
	types.Session
}

var CsvHeader = []string{"Date", "Event Number", "Character Number", "Season", "Scenario Number", "Variant", "Scenario Name", "Player/GM"}

var starfinderModules, pathfinderModules, pathfinder2Modules []*regexp.Regexp

func init() {
	for _, s := range starfinderModuleStrings {
		starfinderModules = append(starfinderModules, regexp.MustCompile(s))
	}
	for _, s := range pathfinderModuleStrings {
		pathfinderModules = append(pathfinderModules, regexp.MustCompile(s))
	}
	for _, s := range pathfinder2ModuleStrings {
		pathfinder2Modules = append(pathfinder2Modules, regexp.MustCompile(s))
	}
}

var s0s1Regex = regexp.MustCompile("^#([0-9]+):")
var variantRegex = regexp.MustCompile(`^([0-9]+)([^0-9]+)`)
var starfinderRegex = []*regexp.Regexp{
	regexp.MustCompile(`^Starfinder Society Scenario #([0-9]+)[-–]([0-9]+): (.*)$`),
}

// ParseName teases apart and stores interesting information from the scenario name- season number and scenarion number
// and stores it in the Session object. If an error occurs, error will be non-nil; some fields may be correctly
// populated; and the full raw scenario name will be saved in the ScenarioName field.
func (s *Session) ParseName(raw string) error {
	raw = strings.TrimSpace(raw)
	sn := raw

	if scen, ok := staticScenarioNumbers[sn]; ok {
		s.Season = scen.nums[0]
		s.Number = scen.nums[1]
		s.Game = scen.sys
		s.ScenarioName = sn
		return nil
	}

	if s0s1 := s0s1Regex.FindStringSubmatch(sn); s0s1 != nil {
		num := s0s1[1]
		season, err := strconv.Atoi(num)
		if err == nil {
			s.Game = "Pathfinder"
			s.Season = season / 29
			s.Number = season
			s.ScenarioName = strings.TrimSpace(sn[strings.Index(sn, " ")+1:])
			return nil
		}
	}

	for _, sfre := range starfinderRegex {
		if sf := sfre.FindStringSubmatch(sn); sf != nil {
			season, err := strconv.Atoi(sf[1])
			var scenario int
			if err == nil {
				scenario, err = strconv.Atoi(sf[2])
			}
			if err == nil {
				s.Game = "Starfinder"
				s.Season = season
				s.Number = scenario
				s.ScenarioName = strings.TrimSpace(sf[3])
				return nil
			}
		}
	}

	s.Season = -1
	s.Number = -1

	if sn[0] != '#' {
		s.ScenarioName = strings.TrimSpace(raw)
		for _, re := range pathfinderModules {
			if re.MatchString(raw) {
				s.Game = "Pathfinder"
				return nil
			}
		}
		for _, re := range pathfinder2Modules {
			if re.MatchString(raw) {
				s.Game = "Pathfinder2"
				return nil
			}
		}
		for _, re := range starfinderModules {
			if re.MatchString(raw) {
				s.Game = "Starfinder"
				return nil
			}
		}
		return fmt.Errorf("no parser or static record for %q", raw)
	}

	sn = strings.TrimLeft(sn, "#")
	term := strings.IndexAny(sn, "-–")
	if term == -1 {
		s.ScenarioName = strings.TrimSpace(raw)
		return fmt.Errorf("parsing %q: no season terminator", raw)
	}

	season, err := strconv.Atoi(sn[0:term])
	if err != nil {
		s.ScenarioName = strings.TrimSpace(raw)
		return fmt.Errorf("parsing %q: could not parse %q as number: %s", raw, sn[0:term], err)
	}
	s.Season = season

	sn = strings.TrimLeft(sn[term:], "-–")
	term = strings.IndexAny(sn, ":— ")
	if term == -1 {
		s.ScenarioName = strings.TrimSpace(raw)
		return fmt.Errorf("parsing %q: could not find scenario terminator", raw)
	}

	if variant := variantRegex.FindStringSubmatch(sn[0:term]); variant != nil {
		s.Variant = variant[2]
		term -= len(variant[2])
	}

	number, err := strconv.Atoi(sn[0:term])
	if err != nil {
		s.ScenarioName = strings.TrimSpace(raw)
		return fmt.Errorf("parsing %q: could not parse %q as scenario number: %s", raw, sn[0:term], err)
	}

	s.Number = number
	s.ScenarioName = strings.TrimSpace(strings.TrimLeft(sn[term+len(s.Variant):], ":— "))
	s.Game = "Pathfinder"
	return nil
}

const (
	dateCell     = 0
	gmCell       = 1
	scenarioCell = 2
	eventCell    = 4
	playerCell   = 7
	charNameCell = 8
	prestigeCell = 10
	maxCell      = 10
)

// sessionFromCells converts a list of cells (a 12-long string slice corresponding to the columnar format on the paizo
// sessions page) to a hydrated Session object. The first cell is handled specially, and is intended to be an RFC3339
// time string, which is retrieved from the `datetime` attribute of the `time` object that occupies the first cell.
// most parse errors will return a non-nil error and a partially hydrated session object, but some parse errors will
// return a `nil` session object.
func sessionFromCells(characters []types.Character, cells []string) (*types.Session, error) {
	if len(cells) < maxCell {
		return nil, fmt.Errorf("expected >=%d elements in cells, received %d", maxCell, len(cells))
	}
	ret := &Session{}

	if cells[dateCell] != "" {
		t, err := time.Parse(time.RFC3339, cells[dateCell])
		if err != nil {
			return &ret.Session, fmt.Errorf("expected first cell to be RFC3339 date, but could not parse %q: %s", cells[dateCell], err)
		}
		ret.Date = t
	} else {
		ret.Date = time.Time{}
	}

	err := ret.ParseName(cells[scenarioCell])
	if err != nil {
		return &ret.Session, fmt.Errorf("expected sixth cell to be scenario name, but could not parse %q: %s", cells[scenarioCell], err)
	}

	evNumStr := cells[eventCell]
	evNum, err := strconv.ParseInt(evNumStr, 10, 64)
	if err != nil {
		return &ret.Session, fmt.Errorf("expected second cell to be event number, but could not parse %q: %s", evNumStr, err)
	}
	ret.EventNumber = append(ret.EventNumber, evNum)

	if !strings.Contains(cells[prestigeCell], "GM") {
		charNumStr := cells[playerCell]
		charNumDash := strings.Index(charNumStr, "-")
		if charNumDash == -1 {
			return &ret.Session, fmt.Errorf("expected eighth cell to contain character number, but %q did not contain dash", charNumStr)
		}
		charNumPart := strings.TrimLeft(charNumStr[charNumDash:], "-")
		if charNumPart == "" {
			ret.Character = append(ret.Character, 0)
		} else {
			charNum, err := strconv.Atoi(charNumPart)
			if err != nil {
				return &ret.Session, fmt.Errorf("in seventh cell %q, could not parse character number part %q: %s", charNumStr, charNumPart, err)
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
		ret.GM = true
		for _, char := range characters {
			if char.Name == cells[charNameCell] {
				ret.Character = append(ret.Character, -1*char.Number)
				break
			}
		}
	}

	return &ret.Session, nil
}
