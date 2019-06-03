package types

type System int

const (
	Unknown System = iota
	Pathfinder
	PathfinderCore
	Starfinder
	Pathfinder2
)

type Character struct {
	System   System
	Number   int
	Name     string
	Prestige map[string]int
	Faction  string
}
