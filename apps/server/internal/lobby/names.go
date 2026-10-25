package lobby

import "math/rand/v2"

// The words random game names are made of, after the game's world.
var (
	nameStarts = []string{
		"Blazing", "Tidal", "Radiant", "Shadow", "Wild", "Armored", "Crimson", "Frozen",
		"Thunder", "Ancient", "Hidden", "Burning", "Silent", "Storm", "Iron", "Crystal",
	}
	nameEnds = []string{
		"Clash", "Duel", "Showdown", "Battle", "Challenge", "Rumble", "Trial", "Skirmish",
		"Rivalry", "Gauntlet",
	}
)

// randomName is a name for a game its host did not name, such as "Tidal
// Showdown".
func randomName() string {
	return nameStarts[rand.IntN(len(nameStarts))] + " " + nameEnds[rand.IntN(len(nameEnds))]
}
