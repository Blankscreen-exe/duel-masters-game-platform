package computer

import (
	"slices"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// cardMove is a move made with a card, as the duel board sends it.
type cardMove struct {
	Header string `json:"header"`
	ID     string `json:"virtualId"`
}

// headerOnly is a move that names nothing, such as ending the turn.
type headerOnly struct {
	Header string `json:"header"`
}

// turnPlan remembers the moves made in the computer's turn, so each is
// tried once: a move the engine refuses, such as an evolution with nothing
// to evolve from, is not tried again that turn.
type turnPlan struct {
	turn     int
	charged  bool
	played   map[string]bool
	attacked map[string]bool
	// anyTarget are the creatures attacking a creature because they have to,
	// which take any target rather than only one they beat.
	anyTarget map[string]bool
	// ending is set once it has tried to end the turn, when the engine had
	// sent warningsAtEnd warnings; another since is the engine refusing.
	ending        bool
	warningsAtEnd int
	// refusedEndings counts the times the engine refused to end the turn,
	// as it does while a creature must still attack, so once refused, it
	// attacks with whatever can.
	refusedEndings int
}

// maxRefusedEndings is how many times the engine may refuse to end the
// computer's turn before it gives up the game, the engine refusing however
// it plays.
const maxRefusedEndings = 4

func (p *turnPlan) startTurn(turn int) {
	if p.turn == turn && p.played != nil {
		return
	}
	*p = turnPlan{turn: turn, played: map[string]bool{}, attacked: map[string]bool{}, anyTarget: map[string]bool{}}
}

// nextMove is the computer's next move in its own turn: charging mana, then
// playing cards, then attacking, then ending the turn.
func (pl *Player) nextMove(state *server.MatchState) any {
	pl.plan.startTurn(state.Turn)
	me := board{side: &state.Me}
	opponent := board{side: &state.Opponent}

	if !pl.plan.charged && !state.TurnManaCharged && state.CanChargeMana {
		pl.plan.charged = true
		if card, ok := pl.manaToCharge(state); ok {
			return cardMove{Header: "add_to_manazone", ID: card}
		}
	}

	if card, ok := pl.cardToPlay(state); ok {
		pl.plan.played[card] = true
		return cardMove{Header: "add_to_playzone", ID: card}
	}

	if pl.plan.ending && pl.view.warnings > pl.plan.warningsAtEnd {
		pl.plan.refusedEndings++
		pl.plan.ending = false
	}

	if attack, ok := pl.attack(state, me, opponent, pl.plan.refusedEndings > 0); ok {
		pl.plan.attacked[attack.ID] = true
		return attack
	}

	if pl.plan.refusedEndings > maxRefusedEndings {
		pl.logger.Warn("the engine refuses to end the computer's turn, and it gives up the game", "turn", state.Turn)
		pl.view.finished = true
		return headerOnly{Header: "resign"}
	}
	pl.plan.ending, pl.plan.warningsAtEnd = true, pl.view.warnings
	return headerOnly{Header: "end_turn"}
}

// manaToCharge picks the card to put into the mana zone this turn, if any:
// the one it least needs to keep, favouring a civilization its mana lacks.
func (pl *Player) manaToCharge(state *server.MatchState) (string, bool) {
	hand := state.Me.Hand
	mana := len(state.Me.Manazone)
	if len(hand) == 0 || (mana >= pl.style.ManaTarget && len(hand) < 3) {
		return "", false
	}

	inMana := map[string]bool{}
	for _, card := range state.Me.Manazone {
		if known, ok := pl.library.Card(card.ImageID); ok {
			for _, civilization := range known.Civilizations {
				inMana[civilization] = true
			}
		}
	}

	// lacking reports whether none of the civilizations is in the mana zone,
	// so a card of them cannot be cast yet.
	lacking := func(civilizations []string) bool {
		return !slices.ContainsFunc(civilizations, func(civilization string) bool { return inMana[civilization] })
	}

	best, bestScore := "", 0.0
	for _, card := range hand {
		known, ok := pl.library.Card(card.ImageID)
		if !ok {
			continue
		}

		// Keeping a card it can play soon is worth more than one it cannot.
		keep := worth(known)
		switch {
		case lacking(known.Civilizations):
			keep *= 0.4
		case known.Cost <= mana+1:
			keep *= 1.5
		case known.Cost > mana+3:
			keep *= 0.5
		}

		// Charging a card of a civilization the mana zone lacks lets the rest
		// of the hand of it be cast.
		for _, other := range hand {
			if other.CardID == card.CardID {
				continue
			}
			if needing, ok := pl.library.Card(other.ImageID); ok && lacking(needing.Civilizations) &&
				slices.ContainsFunc(needing.Civilizations, func(c string) bool { return slices.Contains(known.Civilizations, c) }) {
				keep--
			}
		}

		if best == "" || keep < bestScore {
			best, bestScore = card.CardID, keep
		}
	}
	return best, best != ""
}

// cardToPlay picks the card to play from its hand, if any: the most worth
// among those it can pay for, leaving out spells with nothing to do.
func (pl *Player) cardToPlay(state *server.MatchState) (string, bool) {
	best, bestScore := "", 0.0
	for _, card := range state.Me.Hand {
		if card.Flags&uint8(match.PlayableFlag) == 0 || pl.plan.played[card.CardID] {
			continue
		}
		known, ok := pl.library.Card(card.ImageID)
		if !ok || (known.IsSpell() && !spellHasUse(known, state)) {
			continue
		}

		// Cost first, so the mana goes as far as it can.
		score := float64(known.Cost)*2 + worth(known)
		if best == "" || score > bestScore {
			best, bestScore = card.CardID, score
		}
	}
	return best, best != ""
}

// spellHasUse reports whether a spell would do anything: one that removes,
// taps or bounces creatures needs the opponent to have some, and one that
// makes the opponent discard needs them to hold cards.
func spellHasUse(spell Card, state *server.MatchState) bool {
	switch {
	case spell.Does("removal") || spell.Does("removal_bounce") || spell.Does("tap_opp"):
		return len(state.Opponent.Battlezone) > 0
	case spell.Does("discard_opp"):
		return state.Opponent.HandCount > 0
	}
	return true
}

// attack picks an attack to make, if any: first on a creature it can beat,
// then on the opponent. A creature that must attack attacks whatever the
// risk, and when forced, so does every creature that can.
func (pl *Player) attack(state *server.MatchState, me, opponent board, forced bool) (cardMove, bool) {
	ready := make([]server.CardState, 0)
	for _, creature := range state.Me.Battlezone {
		sick := me.has(creature.CardID, cnd.SummoningSickness) && !me.has(creature.CardID, cnd.SpeedAttacker)
		if creature.Flags&uint8(match.TappedFlag) == 0 && !sick && !pl.plan.attacked[creature.CardID] {
			ready = append(ready, creature)
		}
	}

	for _, attacker := range ready {
		known, _ := pl.library.Card(attacker.ImageID)
		if forced || known.Does("must_attack") {
			return pl.anyAttack(attacker, me, opponent), true
		}
	}

	// A creature that beats one the opponent can be attacked with attacks it.
	for _, attacker := range ready {
		if me.has(attacker.CardID, cnd.CantAttackCreatures) {
			continue
		}
		if _, ok := pl.targetFor(attacker, me, opponent); ok {
			return cardMove{Header: "attack_creature", ID: attacker.CardID}, true
		}
	}

	lethal := len(state.Opponent.Shieldzone) == 0
	blockers := opponent.untappedBlockers()
	for _, attacker := range ready {
		if me.has(attacker.CardID, cnd.CantAttackPlayers) {
			continue
		}
		if lethal {
			return cardMove{Header: "attack_player", ID: attacker.CardID}, true
		}
		if pl.style.KeepBlockers && me.has(attacker.CardID, cnd.Blocker) && len(state.Opponent.Battlezone) > 0 {
			continue
		}

		// A blocker that would win the battle makes the attack a loss.
		unblockable := me.has(attacker.CardID, cnd.CantBeBlocked)
		power := me.attackingPower(attacker.CardID)
		stopped := false
		for _, blocker := range blockers {
			if opponent.power(blocker.CardID) >= power {
				stopped = true
				break
			}
		}
		if stopped && !unblockable && pl.style.Aggression < 1 {
			continue
		}

		// Holding back, it attacks only while ahead on the battle zone.
		ahead := len(state.Me.Battlezone) >= len(state.Opponent.Battlezone)
		if pl.style.Aggression >= 0.5 || ahead {
			return cardMove{Header: "attack_player", ID: attacker.CardID}, true
		}
	}

	return cardMove{}, false
}

// anyAttack is the attack a creature that has to attack makes: on a creature
// it can beat if there is one, or else on the opponent, or else on any
// creature it may attack.
func (pl *Player) anyAttack(attacker server.CardState, me, opponent board) cardMove {
	if !me.has(attacker.CardID, cnd.CantAttackCreatures) {
		if _, ok := pl.targetFor(attacker, me, opponent); ok {
			return cardMove{Header: "attack_creature", ID: attacker.CardID}
		}
	}
	if !me.has(attacker.CardID, cnd.CantAttackPlayers) {
		return cardMove{Header: "attack_player", ID: attacker.CardID}
	}
	pl.plan.anyTarget[attacker.CardID] = true
	return cardMove{Header: "attack_creature", ID: attacker.CardID}
}

// targetFor picks the opponent's creature the attacker is best off
// attacking: the most worth among those it beats in battle.
func (pl *Player) targetFor(attacker server.CardState, me, opponent board) (server.CardState, bool) {
	power := me.attackingPower(attacker.CardID)
	slayer := me.has(attacker.CardID, cnd.Slayer)
	untapped := me.has(attacker.CardID, cnd.AttackUntapped)

	var best server.CardState
	bestScore, found := 0.0, false
	for _, target := range opponent.side.Battlezone {
		tapped := target.Flags&uint8(match.TappedFlag) != 0
		if (!tapped && !untapped) || opponent.has(target.CardID, cnd.CantBeAttacked) {
			continue
		}
		wins := power > opponent.power(target.CardID)
		trades := slayer || power == opponent.power(target.CardID)
		score := opponent.creatureWorth(pl.library, target)
		if !wins && !(trades && score > me.creatureWorth(pl.library, attacker)) {
			continue
		}
		if !found || score > bestScore {
			best, bestScore, found = target, score, true
		}
	}
	return best, found
}
