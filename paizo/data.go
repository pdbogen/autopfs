package paizo

// Specials and other scenarios that have a season number and don't include it or have unique formatting.
var staticScenarioNumbers = map[string]struct {
	nums [2]int
	sys  string
}{
	"Starfinder Society Special #1-00: Claim to Salvation":                   {[2]int{1, 0}, "Starfinder"},
	"Starfinder Society Roleplaying Guild Special #1-00: Claim to Salvation": {[2]int{1, 0}, "Starfinder"},
	"Special: Year of the Shadow Lodge":                                      {[2]int{2, 0}, "Pathfinder"},
	"Special: Blood Under Absalom":                                           {[2]int{3, 0}, "Pathfinder"},
	"Special: Race for the Runecarved Key":                                   {[2]int{4, 0}, "Pathfinder"},
}

// Modules are things that are not tied to a specific season and are not numbered
var starfinderModuleStrings = []string{
	`^Starfinder Adventure Path #`,
	`^Starfinder AP:`,
	`^Starfinder Skitter Shot$`,
	`^Starfinder Society Roleplaying Guild Quest: `,
	`^Starfinder Society Quest: `,
}

// Modules are things that are not tied to a specific season and are not numbered
var pathfinderModuleStrings = []string{
	`^Intro 1:`,
	`^Feast of Ravenmoor$`,
	`^Carrion Hill$`,
	`^Crypt of the Everflame$`,
	`^The Dragon's Demand - `,
	`^The Emerald Spire Superdungeon`,
	`^We Be Goblins!$`,
	`^Masks of the Living God$`,
	`^Tears at Bitter Manor - `,
	`^Master of the Fallen Fortress$`,
	`^City of Golden Death$`,
	`^Quest: Honor's Echo$`,
	`^Fangwood Keep$`,
}

var pathfinder2ModuleStrings = []string{
	`^Playtest Scenario #`,
}
