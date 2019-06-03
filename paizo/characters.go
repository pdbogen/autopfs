package paizo

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/pdbogen/autopfs/types"
	"strconv"
	"strings"
)

func (p *Paizo) GetCharacters() ([]types.Character, error) {
	bow := p.bow
	pageUrl := "https://paizo.com/organizedPlay/myAccount"

	if err := bow.Open(pageUrl); err != nil {
		return nil, fmt.Errorf("opening %s: %s", pageUrl, err)
	}

	characters := []types.Character{}

	rows := bow.Find("div.bb-content div table tbody tr")
	log.Debugf("found %d rows", rows.Size())
	for i := 0; i < rows.Size(); i++ {
		row := rows.Slice(i, i+1)
		cells := row.Find("td").Map(func(_ int, cell *goquery.Selection) string {
			return strings.TrimSpace(cell.Text())
		})
		if cells[0][0] != '#' {
			continue
		}

		imgs := row.Find("td a img").Map(func(_ int, img *goquery.Selection) string {
			return strings.TrimSpace(img.AttrOr("alt", ""))
		})
		if len(imgs) == 0 {
			imgs = []string{"unknown"}
		}

		numStr := strings.Split(cells[0], "-")
		if len(numStr) != 2 {
			log.Errorf("unexpected character number format %q: was not two `-`-separated items", numStr)
			numStr = []string{"", "0"}
		}
		num, err := strconv.Atoi(numStr[1])
		if err != nil {
			log.Errorf("unexpected character number format %q: %s", numStr, err)
			num = 0
		}

		char := types.Character{
			Name:     cells[2],
			Faction:  imgs[0],
			Number:   num,
			Prestige: map[string]int{},
		}

		var prestige string
		for _, line := range strings.Split(cells[3], "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Total Reputation: ") {
				line = strings.SplitN(line, "Total Reputation:", 2)[1]
				line = strings.TrimSpace(line)
				prestige = "Total"
			}
			if strings.HasPrefix(line, "Fame: ") {
				line = strings.SplitN(line, "Fame:", 2)[1]
				line = strings.TrimSpace(line)
				prestige = "Fame"
			}
			if len(line) == 0 {
				continue
			}
			if line[len(line)-1] == ':' {
				prestige = line[:len(line)-1]
				continue
			}

			if amt, err := strconv.Atoi(line); err == nil {
				char.Prestige[prestige] = amt
				continue
			}
			log.Errorf("unsure how to handle prestige line %q", line)
		}

		switch cells[1] {
		case "STAR":
			char.System = types.Starfinder
		case "RPG":
			char.System = types.Pathfinder
		case "PFC":
			char.System = types.PathfinderCore
		default:
			log.Errorf("unexpected system specifier %q", cells[1])
		}
		//log.Debug(strings.Join(imgs, " / "))
		log.Debugf("%+v", char)
		characters = append(characters, char)
	}

	return characters, nil
}
