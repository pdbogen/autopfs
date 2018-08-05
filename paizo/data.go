package paizo

// Specials and other scenarios that have a season number and don't include it or have unique formatting.
var staticScenarioNumbers = map[string][]int{
	"Starfinder Society Special #1-00: Claim to Salvation":                   {1, 0},
	"Starfinder Society Roleplaying Guild Special #1-00: Claim to Salvation": {1, 0},
	"Special: Year of the Shadow Lodge":                                      {2, 0},
	"Special: Blood Under Absalom":                                           {3, 0},
	"Special: Race for the Runecarved Key":                                   {4, 0},
}

// Modules are things that are not tied to a specific season and are not numbered
var moduleStrings = []string{
	`^Starfinder Adventure Path #`,
	`^Starfinder AP:`,
	`^Starfinder Society Roleplaying Guild Quest: `,
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
