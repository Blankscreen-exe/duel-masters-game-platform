package computer

// Style is how a computer player plays: what it values and how readily it
// takes risks. Personalities each have their own.
type Style struct {
	// Aggression, from 0 to 1, is how readily it attacks the opponent's
	// shields rather than keeping its creatures back: at 0 only when it is
	// safe, at 1 whenever it can.
	Aggression float64
	// KeepBlockers keeps its blockers back to defend, unless an attack
	// would win the game.
	KeepBlockers bool
	// ManaTarget is how much mana it builds up before it only charges mana
	// with cards it cannot use.
	ManaTarget int
}

// Balanced is a style that plays neither all out nor holding back.
var Balanced = Style{Aggression: 0.5, KeepBlockers: true, ManaTarget: 7}
