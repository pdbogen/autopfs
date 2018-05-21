package paizo

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/browser"
	"regexp"
	"strings"
	"time"
)

type Paizo struct {
	bow *browser.Browser
}

// AddModule registers at runtime a new module, which is to say a "scenario name" that does not have an associated
// season and scenario number.
func AddModule(mod *regexp.Regexp) {
	modules = append(modules, mod)
}

// Login creates and returns a new Paizo object with an active session. Logging in can take several seconds; so you
// should be parsimonious about calling this; but on the flip side, sessions may terminate unexpectedly, and the code
// currently makes no attempt to detect this.
//
// If Login is not able to login, a non-nil error is returned indicating why. This attempts to include any error message
// reported by Paizo, as well.
func Login(email, pass string) (*Paizo, error) {
	bow := surf.NewBrowser()
	ret := &Paizo{
		bow: bow,
	}

	err := bow.Open("https://paizo.com/organizedPlay/myAccount")
	if err != nil {
		return nil, fmt.Errorf("opening login page: %s", err)
	}

	var form browser.Submittable
	for _, f := range bow.Forms() {
		if f == nil {
			continue
		}
		dom := f.Dom()
		if f == nil {
			continue
		}
		if dom.Find("input[name=e]").Size() == 1 {
			form = f
			break
		}
	}

	if form == nil {
		return nil, errors.New("could not find a form having an input named `e`")
	}

	err = form.Set("e", email)
	if err != nil {
		return nil, fmt.Errorf("setting email input `e`: %s", err)
	}
	err = form.Set("z", pass)
	if err != nil {
		return nil, fmt.Errorf("setting password input `z`: %s", err)
	}

	err = form.Submit()
	if err != nil {
		return nil, fmt.Errorf("submitting login form: %s", err)
	}

	if !strings.Contains(bow.Title(), "My Organized Play") {
		err := fmt.Errorf("login failed! title was %q", bow.Title())
		am := bow.Find("div.alert-message")
		if am.Size() > 0 {
			err = fmt.Errorf("%s; alert message was %q", err, am.Text())
		}
		return nil, err
	}

	return ret, nil
}

// GetSessions returns a de-duplicated (see DeDupe) list of sessions for the user that the Paizo object is logged into.
// If sessions cannot be retrieved or parse, err is non-nil. In such a case, sessions may by non-nil and still contain
// useful data, especially if the error related to the parsing of a specific session.
func (p *Paizo) GetSessions(player bool) (sessions []*Session, err error) {
	bow := p.bow
	parseErrors := []string{}

	var pageUrl string
	if player {
		pageUrl = "https://secure.paizo.com/cgi-bin/WebObjects/Store.woa/wa/browse?path=organizedPlay/myAccount/playersessions#tabs"
	} else {
		pageUrl = "https://secure.paizo.com/cgi-bin/WebObjects/Store.woa/wa/browse?path=organizedPlay/myAccount/gmsessions#tabs"
	}

	if err := bow.Open(pageUrl); err != nil {
		return nil, fmt.Errorf("opening page %q: %s", pageUrl, err)
	}
	sessions = []*Session{}

	for {
		rows := bow.Find("div#results table tr")
		for i := 0; i < rows.Size(); i++ {
			row := rows.Slice(i, i+1)
			cells := row.Find("td").Map(func(i int, cell *goquery.Selection) string {
				if i == 0 {
					return strings.TrimSpace(cell.Find("time").AttrOr(
						"datetime",
						time.Unix(0, 0).Format(time.RFC3339)),
					)
				}
				return strings.TrimSpace(cell.Text())
			})
			if len(cells) != 12 {
				continue
			}

			sess, err := sessionFromCells(cells, player)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("trouble parsing row %q: %s",
					regexp.MustCompile(" +").ReplaceAllString(
						strings.Replace(row.Text(), "\n", " / ", -1),
						" ",
					),
					err,
				))
				if sess == nil {
					return nil, fmt.Errorf("fatal error parsing scenario row: %s", err)
				}
			}
			sessions = append(sessions, sess)
		}

		next := bow.Find("a").FilterFunction(func(_ int, a *goquery.Selection) bool {
			return a.Text() == "next >"
		})

		if next.Size() != 1 {
			break
		}

		nextUrl := fmt.Sprintf("https://secure.paizo.com/%s", next.AttrOr("href", ""))
		if err := bow.Open(nextUrl); err != nil {
			return nil, fmt.Errorf("unexpected error clicking `next`: %s", err)
		}
	}
	err = nil
	if len(parseErrors) > 0 {
		err = fmt.Errorf("Parse errors occurred: %s", strings.Join(parseErrors, ", "))
	}
	return DeDupe(sessions), err
}
