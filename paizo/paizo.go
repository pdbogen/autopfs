package paizo

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/browser"
	"github.com/op/go-logging"
	"github.com/pdbogen/autopfs/types"
	"regexp"
	"strconv"
	"strings"
)

var log = logging.MustGetLogger("paizo")

type Paizo struct {
	bow *browser.Browser
}

// Login creates and returns a new Paizo object with an active session. Logging in can take several seconds; so you
// should be parsimonious about calling this; but on the flip side, sessions may terminate unexpectedly, and the code
// currently makes no attempt to detect this.
//
// If Login is not able to login, a non-nil error is returned indicating why. This attempts to include any error message
// reported by Paizo, as well.
func Login(email, pass string) (*Paizo, error) {
	browserObject := surf.NewBrowser()
	ret := &Paizo{
		bow: browserObject,
	}

	err := browserObject.Open("https://paizo.com/organizedPlay/myAccount")
	if err != nil {
		return nil, fmt.Errorf("opening login page: %s", err)
	}

	log.Debugf("Got login page %q, %q", browserObject.Title(), browserObject.Url().String())
	var form browser.Submittable
	for _, f := range browserObject.Forms() {
		if f == nil {
			continue
		}
		dom := f.Dom()
		if f == nil {
			continue
		}
		if dom.Find("input[name=e]").Size() == 1 {
			log.Debug("Got input with name=e")
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
	err = form.Set("zzz", pass)
	if err != nil {
		return nil, fmt.Errorf("setting password input `z`: %s", err)
	}

	log.Debug("email and password fields set")
	err = form.Submit()
	if err != nil {
		return nil, fmt.Errorf("submitting login form: %s", err)
	}

	log.Debugf("Submitted login; now at %q", browserObject.Title())
	if !strings.Contains(browserObject.Title(), "My Organized Play") {
		err := fmt.Errorf("login failed! title was %q", browserObject.Title())
		am := browserObject.Find("div.alert-message")
		if am.Size() > 0 {
			err = fmt.Errorf("%s; alert message was %q", err, am.Text())
		}
		return nil, err
	}

	log.Debugf("Login appears successful!")
	return ret, nil
}

var countRegexp = regexp.MustCompile(`\d+\s+to\s+\d+\s+of\s+(\d+)`)

func GetSessionCount(bow *browser.Browser) (int, error) {
	totalElem := bow.Find("div#results table tbody tr td").FilterFunction(func(_ int, selection *goquery.Selection) bool {
		return countRegexp.MatchString(selection.Text())
	})

	if totalElem.Size() != 1 {
		return -1, fmt.Errorf("%d TDs found matching session count regex, expected one", totalElem.Size())
	}

	totalSubmatches := countRegexp.FindStringSubmatch(totalElem.Text())
	if len(totalSubmatches) != 2 {
		return -1, errors.New("surprisingly, found matching td but did not contain exactly two submatches")
	}

	total, err := strconv.Atoi(totalSubmatches[1])
	if err != nil {
		return -1, fmt.Errorf("total submatch %q could not be parsed as integer: %v", totalSubmatches[1], err)
	}

	return total, nil
}

// GetSessions returns a de-duplicated (see DeDupe) list of sessions for the user that the Paizo object is logged into.
// If sessions cannot be retrieved or parse, err is non-nil. In such a case, sessions may by non-nil and still contain
// useful data, especially if the error related to the parsing of a specific session.
func (p *Paizo) GetSessions(characters []types.Character, progress func(cur, total int)) (playerSessions []*types.Session, gmSessions []*types.Session, err error) {
	bow := p.bow
	parseErrors := []string{}

	pageUrl := "https://paizo.com/cgi-bin/WebObjects/Store.woa/wa/browse?path=organizedPlay/myAccount/allsessions#tabs"

	if progress != nil {
		progress(0, 0)
	}

	if err := bow.Open(pageUrl); err != nil {
		return nil, nil, fmt.Errorf("opening page %q: %s", pageUrl, err)
	}
	log.Debugf("Loaded sessions page %q", bow.Title())
	playerSessions = []*types.Session{}
	gmSessions = []*types.Session{}

	for {
		rows := bow.Find("div#results table tr")
		log.Debugf("found %d TRs in table in div with id=results", rows.Size())
		for i := 0; i < rows.Size(); i++ {
			row := rows.Slice(i, i+1)

			cells := row.Find("td").Map(func(i int, cell *goquery.Selection) string {
				return strings.TrimSpace(cell.Text())
			})

			if len(cells) < maxCell {
				log.Debugf("Skipping row %q with %d cells", strings.Join(cells, ","), len(cells))
				continue
			}

			datetime := row.Find("td").First().Find("time").AttrOr("datetime", "")
			if datetime != "" {
				cells[dateCell] = datetime
			}

			sess, err := sessionFromCells(characters, cells)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("trouble parsing row %q: %s",
					regexp.MustCompile(" +").ReplaceAllString(
						strings.Replace(row.Text(), "\n", " / ", -1),
						" ",
					),
					err,
				))
				if sess == nil {
					return nil, nil, fmt.Errorf("fatal error parsing scenario row: %s", err)
				}
			}

			if sess.GM {
				gmSessions = append(gmSessions, sess)
			} else {
				playerSessions = append(playerSessions, sess)
			}
		}

		if progress != nil {
			total, err := GetSessionCount(bow)
			if err != nil {
				log.Warningf("getting total session count: %v", err)
			} else {
				progress(len(gmSessions)+len(playerSessions), total)
			}
		}

		next := bow.Find("a").FilterFunction(func(_ int, a *goquery.Selection) bool {
			return a.Text() == "next >"
		})

		if next.Size() != 1 {
			break
		}

		nextUrl := fmt.Sprintf("https://secure.paizo.com/%s", next.AttrOr("href", ""))
		if err := bow.Open(nextUrl); err != nil {
			return nil, nil, fmt.Errorf("unexpected error clicking `next`: %s", err)
		}
	}
	err = nil
	if len(parseErrors) > 0 {
		err = fmt.Errorf("Parse errors occurred: %s", strings.Join(parseErrors, ", "))
	}
	return types.DeDupe(playerSessions), types.DeDupe(gmSessions), err
}
