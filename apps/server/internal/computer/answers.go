package computer

import (
	"cmp"
	"slices"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// answer is an answer to a question, as the duel board sends it.
type answer struct {
	Header string   `json:"header"`
	Prompt int      `json:"prompt"`
	Cards  []string `json:"cards"`
	Count  int      `json:"count"`
	Cancel bool     `json:"cancel"`
}

// maxRefusals is how many of its answers to one question the engine may
// refuse before the computer stops trying: the game cannot go on with it, so
// it gives up the game rather than leave its opponent waiting.
const maxRefusals = 12

// effects of a card that cost its own player: a card choosing among its
// player's own cards for one of them is choosing what to give up.
var ownCosts = []string{"discard_own", "destroy_creature_own", "manaburn_own", "shields_to_grave_own", "removal_to_mana"}

// answerTo is the computer's answer to the question, and false once the
// engine has refused too many. Asked again with no refusal, as when the
// engine is slow to take an answer, it gives its best answer again.
func (pl *Player) answerTo(q *question, state *server.MatchState) (answer, bool) {
	if q.refusals > maxRefusals {
		return answer{}, false
	}
	reply := answer{Header: "action", Prompt: q.id, Cards: []string{}}

	// A refused answer is tried again differently: by cancelling, when
	// that is allowed, or else by other choices, at random.
	if q.refusals > 0 {
		if q.cancellable {
			reply.Cancel = true
		} else {
			pl.anyAnswer(q, &reply)
		}
		return reply, true
	}

	switch {
	case q.show:
		reply.Cancel = true
	case q.actionType == "question" && len(q.choices) == 0:
		// Yes: a card asks whether its player wants something it offers.
	case q.actionType == "question" || q.actionType == "searchable":
		reply.Count = pl.rand.IntN(max(len(q.choices), 1))
	case q.actionType == "count":
		reply.Count = q.max
	case q.actionType == "order":
		for _, card := range q.cards {
			reply.Cards = append(reply.Cards, card.CardID)
		}
	default:
		cards, cancel := pl.chooseCards(q, state)
		reply.Cards, reply.Cancel = cards, cancel
	}
	return reply, true
}

// anyAnswer fills reply with an answer chosen at random within what the
// question allows.
func (pl *Player) anyAnswer(q *question, reply *answer) {
	switch {
	case q.show:
		reply.Cancel = true
	case q.actionType == "question" && len(q.choices) == 0:
	case q.actionType == "question" || q.actionType == "searchable":
		reply.Count = pl.rand.IntN(max(len(q.choices), 1))
	case q.actionType == "count":
		reply.Count = q.min + pl.rand.IntN(max(q.max-q.min, 0)+1)
	case q.actionType == "order":
		reply.Cards = pl.any(q.cards, len(q.cards))
	default:
		offered := q.selectable()
		most := min(q.max, len(offered))
		least := min(max(q.min, 0), most)
		reply.Cards = pl.any(offered, least+pl.rand.IntN(most-least+1))
	}
}

// chooseCards picks the cards to answer a question offering cards with, or
// cancels it.
func (pl *Player) chooseCards(q *question, state *server.MatchState) ([]string, bool) {
	offered := q.selectable()
	if state == nil || len(offered) == 0 {
		return nil, q.cancellable
	}
	me, opponent := board{side: &state.Me}, board{side: &state.Opponent}

	switch match.PromptKind(q.about.Kind) {
	case match.PromptPayMana:
		return pl.payMana(q, offered, state), false
	case match.PromptBlock:
		return pl.block(q, offered, state, me, opponent)
	case match.PromptAttackTarget:
		return pl.attackTarget(q, offered, state, me, opponent)
	case match.PromptShieldTrigger:
		return pl.most(offered, 1, state), false
	case match.PromptEvolve:
		return pl.least(offered, 1, state), false
	case match.PromptBreakShields:
		return pl.any(offered, q.max), false
	}

	// A card's effect: harming the opponent's cards as much as it may, and
	// its own as little as it must, unless what it chooses it gains.
	count := q.max
	theirs := 0
	for _, card := range offered {
		if q.opponents(card.CardID) {
			theirs++
		}
	}
	switch {
	case theirs > 0:
		return pl.most(slices.DeleteFunc(slices.Clone(offered), func(c server.CardState) bool { return !q.opponents(c.CardID) }), count, state), false
	case pl.costsItsPlayer(q):
		return pl.least(offered, max(q.min, 0), state), false
	default:
		return pl.most(offered, count, state), false
	}
}

// costsItsPlayer reports whether the card asking makes its own player give
// something up.
func (pl *Player) costsItsPlayer(q *question) bool {
	if q.about.Source == nil {
		return false
	}
	source, ok := pl.library.Card(q.about.Source.ImageID)
	if !ok {
		return false
	}
	return slices.ContainsFunc(ownCosts, source.Does)
}

// cardWorth is what an offered card is worth: in the battle zone as it is
// now, elsewhere as printed. Face-down cards are worth nothing known.
func (pl *Player) cardWorth(card server.CardState, state *server.MatchState) float64 {
	for _, side := range []*server.PlayerState{&state.Me, &state.Opponent} {
		if _, inPlay := side.Powers[card.CardID]; inPlay {
			return board{side: side}.creatureWorth(pl.library, card)
		}
	}
	if known, ok := pl.library.Card(card.ImageID); ok {
		return worth(known)
	}
	return 0
}

// most picks up to count of the cards, those worth most first.
func (pl *Player) most(cards []server.CardState, count int, state *server.MatchState) []string {
	return pl.pick(cards, count, state, -1)
}

// least picks count of the cards, those worth least first.
func (pl *Player) least(cards []server.CardState, count int, state *server.MatchState) []string {
	return pl.pick(cards, count, state, 1)
}

func (pl *Player) pick(cards []server.CardState, count int, state *server.MatchState, order float64) []string {
	sorted := slices.Clone(cards)
	slices.SortStableFunc(sorted, func(a, b server.CardState) int {
		return cmp.Compare(order*pl.cardWorth(a, state), order*pl.cardWorth(b, state))
	})
	return ids(sorted[:min(max(count, 0), len(sorted))])
}

// any picks count of the cards at random, as for face-down shields.
func (pl *Player) any(cards []server.CardState, count int) []string {
	shuffled := slices.Clone(cards)
	pl.rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return ids(shuffled[:min(max(count, 0), len(shuffled))])
}

func ids(cards []server.CardState) []string {
	out := make([]string, 0, len(cards))
	for _, card := range cards {
		out = append(out, card.CardID)
	}
	return out
}

// payMana picks the mana to pay for the card being played: a card of each
// civilization it needs, then those whose civilizations its hand needs
// least.
func (pl *Player) payMana(q *question, offered []server.CardState, state *server.MatchState) []string {
	var needs []string
	if q.about.Source != nil {
		if card, ok := pl.library.Card(q.about.Source.ImageID); ok {
			needs = card.Civilizations
		}
	}

	civilizations := func(card server.CardState) []string {
		if known, ok := pl.library.Card(card.ImageID); ok {
			return known.Civilizations
		}
		return []string{card.Civ}
	}

	// How much the rest of the hand needs each civilization.
	wanted := map[string]int{}
	for _, card := range state.Me.Hand {
		if known, ok := pl.library.Card(card.ImageID); ok {
			for _, civilization := range known.Civilizations {
				wanted[civilization]++
			}
		}
	}
	spare := slices.Clone(offered)
	slices.SortStableFunc(spare, func(a, b server.CardState) int {
		return cmp.Compare(neededBy(civilizations(a), wanted), neededBy(civilizations(b), wanted))
	})

	chosen := assignCivilizations(needs, spare, civilizations)
	for _, card := range spare {
		if len(chosen) >= q.max {
			break
		}
		if !slices.Contains(chosen, card.CardID) {
			chosen = append(chosen, card.CardID)
		}
	}
	return chosen[:min(len(chosen), q.max)]
}

func neededBy(civilizations []string, wanted map[string]int) int {
	most := 0
	for _, civilization := range civilizations {
		most = max(most, wanted[civilization])
	}
	return most
}

// assignCivilizations gives each civilization needed its own mana card, as
// the engine requires, trying the cards in order, and returns the cards
// given. It finds an assignment whenever one exists, by moving a card
// already given to another civilization it also has.
func assignCivilizations(needs []string, mana []server.CardState, civilizations func(server.CardState) []string) []string {
	owner := make([]int, len(mana))
	for i := range owner {
		owner[i] = -1
	}

	var assign func(need int, seen []bool) bool
	assign = func(need int, seen []bool) bool {
		for i, card := range mana {
			if seen[i] || !slices.Contains(civilizations(card), needs[need]) {
				continue
			}
			seen[i] = true
			if owner[i] == -1 || assign(owner[i], seen) {
				owner[i] = need
				return true
			}
		}
		return false
	}
	for need := range needs {
		assign(need, make([]bool, len(mana)))
	}

	var chosen []string
	for i, need := range owner {
		if need != -1 {
			chosen = append(chosen, mana[i].CardID)
		}
	}
	return chosen
}

// block decides whether to block the attack, and with which creature: one
// that wins the battle, or any when the attack would cost too much.
func (pl *Player) block(q *question, offered []server.CardState, state *server.MatchState, me, opponent board) ([]string, bool) {
	attackerPower := 0
	breaksMore := false
	if q.about.Source != nil {
		attackerPower = opponent.attackingPower(q.about.Source.CardID)
		breaksMore = opponent.has(q.about.Source.CardID, cnd.DoubleBreaker) || opponent.has(q.about.Source.CardID, cnd.TripleBreaker)
	}

	var winners []server.CardState
	for _, blocker := range offered {
		if me.power(blocker.CardID) > attackerPower {
			winners = append(winners, blocker)
		}
	}
	if len(winners) > 0 {
		return pl.least(winners, 1, state), false
	}

	shields := len(state.Me.Shieldzone)
	if shields == 0 || (breaksMore && shields <= 2) || (pl.style.Aggression < 0.5 && shields <= 1) {
		return pl.least(offered, 1, state), false
	}
	return nil, true
}

// attackTarget picks the creature to attack: the most worth it beats, or,
// for a creature that has to attack, the least worth.
func (pl *Player) attackTarget(q *question, offered []server.CardState, state *server.MatchState, me, opponent board) ([]string, bool) {
	if q.about.Source == nil {
		return nil, true
	}
	attacker := server.CardState{CardID: q.about.Source.CardID, ImageID: q.about.Source.ImageID}
	target, ok := pl.targetFor(attacker, me, board{side: &server.PlayerState{
		Battlezone: offered,
		Powers:     opponent.side.Powers,
		Statuses:   opponent.side.Statuses,
	}})
	switch {
	case ok:
		return []string{target.CardID}, false
	case pl.plan.anyTarget[attacker.CardID]:
		return pl.least(offered, 1, state), false
	}
	return nil, true
}
