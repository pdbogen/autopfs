package main

import (
	"github.com/pdbogen/autopfs/types"
	"math"
	"sort"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

func GetColumn(name string) *Column {
	for _, col := range Columns {
		if col.Name == name {
			return &col
		}
	}
	return nil
}

type Filter struct {
	Column string
	Values []string
	Op     string
}

func (f Filter) Apply(in []*types.Session) []*types.Session {
	//var select func()
	var selectFn func(session *types.Session, filter Filter) bool
	for _, col := range Columns {
		if col.Name == f.Column {
			selectFn = col.Select
			break
		}
	}

	if selectFn == nil {
		return in
	}

	var out []*types.Session
	for _, s := range in {
		if selectFn(s, f) {
			out = append(out, s)
		}
	}
	return out
}

type Column struct {
	Name         string
	Render       func(session *types.Session) js.Value
	Less         func(i, j *types.Session) bool
	Select       func(session *types.Session, filter Filter) bool
	FilterGadget func(sessions []*types.Session) js.Value
}

var Columns = []Column{
	{
		"Date",
		func(session *types.Session) js.Value {
			return CreateElementText("td", session.Date.Format("Mon 2006-01-02"))
		},
		func(i, j *types.Session) bool {
			return i.Date.Before(j.Date)
		},
		func(session *types.Session, filter Filter) bool {
			sessDate := time.Date(
				session.Date.Year(),
				session.Date.Month(),
				session.Date.Day(),
				0, 0, 0, 0, time.UTC,
			)
			for _, v := range filter.Values {
				valueDate, err := time.Parse(
					"2006-01-02 15:04:05 -0700",
					v+" 00:00:00 -0000")
				if err != nil {
					println("parsing filter value " + v + ": " + err.Error())
					continue
				}
				switch filter.Op {
				case "before":
					if sessDate.Before(valueDate) {
						return true
					}
					println("session date " + sessDate.String() + " was not before " + valueDate.String())
				case "after":
					if sessDate.After(valueDate) {
						return true
					}
					println("session date " + sessDate.String() + " was not after " + valueDate.String())
				}
			}
			return false
		},
		func(_ []*types.Session) js.Value {
			div := CreateElement("div")
			div.Set("style", "text-align: right;")
			div.Call("appendChild", CreateTextNode("From: "))
			inp := CreateElement("input")
			inp.Set("name", "date_after")
			inp.Set("column", "Date")
			inp.Set("op", "after")
			inp.Set("placeholder", "YYYY-MM-DD")
			inp.Set("size", "10")
			div.Call("appendChild", inp)
			div.Call("appendChild", CreateElement("br"))
			div.Call("appendChild", CreateTextNode("To: "))
			inp = CreateElement("input")
			inp.Set("name", "date_before")
			inp.Set("column", "Date")
			inp.Set("op", "before")
			inp.Set("placeholder", "YYYY-MM-DD")
			inp.Set("size", "10")
			div.Call("appendChild", inp)
			return div
		},
	},
	{
		"System",
		func(session *types.Session) js.Value {
			return CreateElementText("td", session.Game)
		},
		func(i, j *types.Session) bool {
			return i.Game < j.Game
		},
		SelectSelectFn(func(session *types.Session) interface{} {
			return session.Game
		}),
		SelectGadget("System", func(session *types.Session) interface{} {
			return session.Game
		}),
	},
	{
		"Event Number",
		func(session *types.Session) js.Value {
			var evStrs []string
			for _, ev := range session.EventNumber {
				evStrs = append(evStrs, strconv.FormatInt(ev, 10))
			}
			return CreateElementText("td", strings.Join(evStrs, ", "))
		},
		func(i, j *types.Session) bool {
			var iEv, jEv int64 = math.MaxInt64, math.MaxInt64
			if len(i.EventNumber) > 0 {
				iEv = i.EventNumber[0]
			}
			if len(j.EventNumber) > 0 {
				jEv = j.EventNumber[0]
			}
			return iEv < jEv
		},
		SelectSelectFn(func(session *types.Session) interface{} {
			return session.EventNumber
		}),
		SelectGadget("Event Number", func(session *types.Session) interface{} {
			return session.EventNumber
		}),
	},
	{
		"Character",
		func(session *types.Session) js.Value {
			td := CreateElement("td")
			for _, cn := range session.Character {
				if cn >= 0 {
					td.Call("appendChild", CreateElementText("span", strconv.Itoa(cn)+" "))
				} else {
					td.Call("appendChild", CreateElementText("span", strconv.Itoa(-1*cn)))
					td.Call("appendChild", CreateElementText("sup", "GM"))
					td.Call("appendChild", CreateTextNode(" "))
				}
			}
			return td
		},
		func(i, j *types.Session) bool {
			var iCn, jCn = math.MaxInt32, math.MaxInt32
			if len(i.Character) > 0 {
				iCn = i.Character[0]
			}
			if len(j.Character) > 0 {
				jCn = j.Character[0]
			}
			if iCn < 0 {
				iCn = iCn * -1
			}
			if jCn < 0 {
				jCn = jCn * -1
			}
			return iCn < jCn
		},
		func(session *types.Session, filter Filter) bool {
		filterCharacters:
			for _, filterCharacterStr := range filter.Values {
				for _, sessionCharacter := range session.Character {
					filterCharacter, err := strconv.Atoi(filterCharacterStr)
					if err != nil {
						continue filterCharacters
					}
					if sessionCharacter == filterCharacter {
						return true
					}
				}
			}
			return false
		},
		func(sessions []*types.Session) js.Value {
			dupes := map[int]bool{}
			var chars []int
			for _, session := range sessions {
				for _, char := range session.Character {
					if char < 0 {
						char = char * -1
					}
					if dupes[char] {
						continue
					}
					chars = append(chars, char)
					dupes[char] = true
				}
			}

			charMap := map[int]types.Character{}
			for _, char := range job.Characters {
				charMap[char.Number] = char
			}

			sort.Ints(chars)
			div := CreateElement("div")
			sel := CreateElement("select")
			sel.Set("column", "Character")
			sel.Set("multiple", true)
			sel.Set("style", "width: 100%;")
			for _, char := range chars {
				opt := CreateElementText("option", strconv.Itoa(char)+" - "+charMap[char].Name)
				opt.Set("value", char)
				sel.Call("appendChild", opt)
			}
			div.Call("appendChild", sel)
			return div
		},
	},
	{
		"Season",
		func(session *types.Session) js.Value {
			if session.Season == -1 {
				return CreateElementText("td", "N/A")
			}
			return CreateElementText("td", strconv.Itoa(session.Season))
		},
		func(i, j *types.Session) bool {
			return i.Season < j.Season
		},
		SelectSelectFn(func(session *types.Session) interface{} {
			return session.Season
		}),
		SelectGadget("Season", func(session *types.Session) interface{} {
			return session.Season
		}),
	},
	{
		"Number",
		func(session *types.Session) js.Value {
			if session.Number == -1 {
				return CreateElementText("td", "N/A")
			}
			return CreateElementText("td", strconv.Itoa(session.Number))
		},
		func(i, j *types.Session) bool {
			return i.Number < j.Number
		},
		SelectSelectFn(func(session *types.Session) interface{} {
			return session.Number
		}),
		SelectGadget("Number", func(session *types.Session) interface{} {
			return session.Number
		}),
	},
	{
		"Variant",
		func(session *types.Session) js.Value {
			return CreateElementText("td", session.Variant)
		},
		func(i, j *types.Session) bool {
			return i.Variant < j.Variant
		},
		SelectSelectFn(func(session *types.Session) interface{} {
			return session.Variant
		}),
		SelectGadget("Variant", func(session *types.Session) interface{} {
			return session.Variant
		}),
	},
	{
		"Scenario Name",
		func(session *types.Session) js.Value {
			return CreateElementText("td", session.ScenarioName)
		},
		func(i, j *types.Session) bool {
			return i.ScenarioName < j.ScenarioName
		},
		SelectSelectFn(func(session *types.Session) interface{} {
			return session.ScenarioName
		}),
		SelectGadget("Scenario Name", func(session *types.Session) interface{} {
			return session.ScenarioName
		}),
	},
	{
		"Player/GM",
		func(session *types.Session) js.Value {
			if session.Player && session.GM {
				return CreateElementText("td", "P/GM")
			}
			if session.Player {
				return CreateElementText("td", "P")
			}
			if session.GM {
				return CreateElementText("td", "GM")
			}
			return CreateElementText("td", "???")
		},
		func(i, j *types.Session) bool {
			iVal, jVal := 0, 0
			if i.GM {
				iVal += 2
			}
			if i.Player {
				iVal += 1
			}
			if j.GM {
				jVal += 2
			}
			if j.Player {
				jVal += 1
			}
			return iVal < jVal
		},
		SelectSelectFn(func(session *types.Session) interface{} {
			if session.Player && session.GM {
				return "P/GM"
			}
			if session.Player {
				return "P"
			}
			if session.GM {
				return "GM"
			}
			return "???"
		}),
		SelectGadget("Player/GM", func(session *types.Session) interface{} {
			if session.Player && session.GM {
				return "P/GM"
			}
			if session.Player {
				return "P"
			}
			if session.GM {
				return "GM"
			}
			return "???"
		}),
	},
}

func SelectSelectFn(getter func(*types.Session) interface{}) func(session *types.Session, filter Filter) bool {
	return func(session *types.Session, filter Filter) bool {
		var valueStrings []string
		var valueInts []int64

		sessionValue := getter(session)
		switch sv := sessionValue.(type) {
		case string:
			sessionValue = []string{sv}
		case int:
			sessionValue = []int64{int64(sv)}
		case int64:
			sessionValue = []int64{sv}
		}
		switch sv := sessionValue.(type) {
		case []string:
			valueStrings = sv
		case []int64:
			valueInts = sv
		}

		for _, filterOpt := range filter.Values {
			for _, valStr := range valueStrings {
				if filterOpt == valStr {
					return true
				}
			}
			for _, valInt := range valueInts {
				if filterOpt == strconv.FormatInt(valInt, 10) {
					return true
				}
			}
		}
		return false
	}
}
func SelectGadget(name string, getter func(*types.Session) interface{}) func(sessions []*types.Session) js.Value {
	return func(sessions []*types.Session) js.Value {
		var valueStrings []string
		var valueInts []int64
		dupeStrings := map[string]bool{}
		dupeInts := map[int64]bool{}

		for _, session := range sessions {
			sessionValue := getter(session)
			switch v := sessionValue.(type) {
			case string:
				sessionValue = []string{v}
			case int:
				sessionValue = []int64{int64(v)}
			case int64:
				sessionValue = []int64{v}
			}
			switch vslice := sessionValue.(type) {
			case []string:
				for _, v := range vslice {
					if dupeStrings[v] {
						continue
					}
					dupeStrings[v] = true
					valueStrings = append(valueStrings, v)
				}
			case []int64:
				for _, v := range vslice {
					if dupeInts[v] {
						continue
					}
					dupeInts[v] = true
					valueInts = append(valueInts, v)
				}
			}
		}

		sort.Strings(valueStrings)
		sort.Slice(valueInts, func(i, j int) bool {
			return valueInts[i] < valueInts[j]
		})

		div := CreateElement("div")
		sel := CreateElement("select")
		sel.Set("column", name)
		sel.Set("multiple", true)
		sel.Set("style", "width: 100%;")
		for _, intValue := range valueInts {
			opt := CreateElementText("option", strconv.FormatInt(intValue, 10))
			opt.Set("value", intValue)
			sel.Call("appendChild", opt)
		}
		for _, stringValue := range valueStrings {
			opt := CreateElementText("option", stringValue)
			opt.Set("value", stringValue)
			sel.Call("appendChild", opt)
		}
		div.Call("appendChild", sel)
		return div
	}
}
