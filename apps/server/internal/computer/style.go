package computer

import (
	"maps"
	"slices"
)

// Style is how a computer player plays: what it values and how readily it
// takes risks. Personalities each have their own.
type Style struct {
	// Label is what the admin panel calls it.
	Label string
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
	// Favours are the keywords and effect categories it plays first, as the
	// card database names them, such as "blocker" or "discard_opp".
	Favours []string
}

// favourBonus is what each thing a style favours adds to a card's place in
// the order it plays its cards.
const favourBonus = 3

// favours is how much the style favours a card.
func (s Style) favours(card Card) float64 {
	bonus := 0.0
	for _, favoured := range s.Favours {
		if card.Has(favoured) || card.Does(favoured) {
			bonus += favourBonus
		}
	}
	return bonus
}

// Balanced is a style that plays neither all out nor holding back.
var Balanced = Style{Label: "Balanced", Aggression: 0.5, KeepBlockers: true, ManaTarget: 7}

// styles are the styles a computer opponent may play with, by the name it is
// stored under.
var styles = map[string]Style{
	"balanced": Balanced,
	// discard empties the opponent's hand first.
	"discard": {Label: "Hand discard", Aggression: 0.5, KeepBlockers: true, ManaTarget: 7, Favours: []string{"discard_opp"}},
	// rush plays cheap creatures and attacks with everything, every turn.
	"rush": {Label: "Rapid summon", Aggression: 1, KeepBlockers: false, ManaTarget: 5, Favours: []string{"speed_attack", "power_attack"}},
	// wall keeps blockers back and attacks only when it is safe.
	"wall": {Label: "Defensive wall", Aggression: 0.2, KeepBlockers: true, ManaTarget: 8, Favours: []string{"blocker"}},
	// tapper taps the opponent's creatures, then attacks.
	"tapper": {Label: "Tapper", Aggression: 0.7, KeepBlockers: false, ManaTarget: 7, Favours: []string{"tap_opp"}},
	// bounce sends the opponent's creatures back to their hand.
	"bounce": {Label: "Bounce", Aggression: 0.5, KeepBlockers: true, ManaTarget: 7, Favours: []string{"removal_bounce"}},
	// giant builds mana fast for the biggest creatures.
	"giant": {Label: "Big creatures", Aggression: 0.6, KeepBlockers: false, ManaTarget: 9, Favours: []string{"ramp", "double_break", "triple_break"}},
}

// StyleNamed returns the style stored under the name.
func StyleNamed(name string) (Style, bool) {
	style, ok := styles[name]
	return style, ok
}

// StyleNames are the names of every style, in order.
func StyleNames() []string {
	return slices.Sorted(maps.Keys(styles))
}

// Strength is how well a computer player plays.
type Strength string

const (
	// Easy now and then makes a small mistake on purpose: playing a card
	// that is not its best, holding back an attack, or not blocking.
	Easy Strength = "easy"
	// Normal always makes the move it thinks best.
	Normal Strength = "normal"
)

// mistakeChance is how often an Easy player makes a mistake when it could.
const mistakeChance = 0.3

// errs reports whether the player makes a mistake this time.
func (pl *Player) errs() bool {
	return pl.strength == Easy && pl.rand.Float64() < mistakeChance
}
