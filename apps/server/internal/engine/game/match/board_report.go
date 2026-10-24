package match

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// boardReport is what the event loop works out for the duel interface beyond
// where the cards are: the creatures in both battle zones, by card id, the
// attack under way, and the turn the duel is on.
type boardReport struct {
	powers   map[string]server.CreaturePower
	statuses map[string][]server.CreatureStatus
	attack   attackReport
	turn     int
}

// attackReport is what the duel interface is told about the attack under way.
type attackReport struct {
	// resolving is set while an attack or a tap ability resolves.
	resolving bool
	// attacker is the creature attacking, once its attack has gone ahead.
	attacker string
}

// shownStatuses are the conditions the interface shows on a creature, in the
// order it shows them, after summoning sickness.
var shownStatuses = []string{
	cnd.PowerAttacker,
	cnd.DoubleBreaker,
	cnd.TripleBreaker,
	cnd.ShieldBreakModifier,
	cnd.SpeedAttacker,
	cnd.Slayer,
	cnd.CantBeBlocked,
	cnd.Stealth,
	cnd.AttackUntapped,
	cnd.Blocker,
	cnd.Survivor,
	cnd.CantBeAttacked,
	cnd.TreatedAsTapped,
	cnd.CantBeSelectedByOpp,
	cnd.CantBeTappedByOpp,
	cnd.CantAttackCreatures,
	cnd.CantAttackPlayers,
	cnd.DoesntUntap,
	cnd.DestroyAfterBattle,
	cnd.TapAbility,
	cnd.SilentSkill,
	cnd.WaveStriker,
}

// countedStatuses carry an amount, which adds up across the effects giving it:
// the power a power attacker gains, and the extra shields a creature breaks.
var countedStatuses = map[string]bool{
	cnd.PowerAttacker:       true,
	cnd.ShieldBreakModifier: true,
}

// knownBoard is what the event loop last worked out about the board.
func (m *Match) knownBoard() boardReport {
	m.boardMutex.Lock()
	defer m.boardMutex.Unlock()

	return m.board
}

// keepBoard keeps what the event loop worked out about the board for
// broadcastKnownState, which cannot work it out itself.
func (m *Match) keepBoard(report boardReport) {
	m.boardMutex.Lock()
	defer m.boardMutex.Unlock()

	m.board = report
}

// workOutCreatures finds the power and statuses of every creature in both
// battle zones. Only the event loop may call it, because working out power
// runs card effects.
func (m *Match) workOutCreatures(report *boardReport) {
	report.powers = make(map[string]server.CreaturePower)
	report.statuses = make(map[string][]server.CreatureStatus)

	for _, p := range []*PlayerReference{m.Player1, m.Player2} {
		battlezone, err := p.Player.Container(BATTLEZONE)
		if err != nil {
			continue
		}

		// GetPower dispatches an event, so work from a snapshot of the zone.
		for _, card := range append([]*Card(nil), battlezone...) {
			report.powers[card.ID] = server.CreaturePower{
				Printed:   card.Power,
				Current:   m.GetPower(card, false),
				Attacking: m.GetPower(card, true),
			}
			report.statuses[card.ID] = m.creatureStatuses(card)
		}
	}
}

// workOutAttack finds the attack under way. Only the event loop may call it,
// because only the event loop changes the attack.
func (m *Match) workOutAttack() attackReport {
	attack := m.resolvingAttack
	if attack == nil {
		return attackReport{}
	}

	report := attackReport{resolving: true}

	// An attacker taps once its attack is beyond cancelling, and one that goes
	// ahead without tapping has its attack confirmed instead.
	if attack.attacks && (attack.announced || attackWentAhead(attack.event)) {
		report.attacker = attack.cardID
	}

	return report
}

// creatureStatuses lists the shown statuses a creature has right now. Stealth
// is only shown while it keeps the creature from being blocked.
func (m *Match) creatureStatuses(card *Card) []server.CreatureStatus {
	statuses := make([]server.CreatureStatus, 0)

	if card.HasSummoningSickness() {
		statuses = append(statuses, server.CreatureStatus{ID: cnd.SummoningSickness})
	}

	for _, id := range shownStatuses {
		if id == cnd.Stealth && !card.stealthActiveIn(m) {
			continue
		}

		status := server.CreatureStatus{ID: id}
		found := false

		for _, condition := range card.Conditions() {
			if condition.ID != id {
				continue
			}

			found = true

			if value, ok := condition.Val.(int); ok && countedStatuses[id] {
				status.Value += value
			}

			if grantedTo(card, condition.Src) {
				status.Granted = true
			}
		}

		if found {
			statuses = append(statuses, status)
		}
	}

	return statuses
}

// grantedTo reports whether a condition's source is another card. A card's own
// abilities name the card itself as their source, and the rules name none.
func grantedTo(card *Card, source interface{}) bool {
	switch src := source.(type) {
	case string:
		return src != "" && src != card.ID
	case *Card:
		return src != nil && src != card
	default:
		return false
	}
}

// ownedBy picks the entries for the creatures in one battle zone.
func ownedBy[T any](battlezone []server.CardState, byCard map[string]T) map[string]T {
	own := make(map[string]T, len(battlezone))

	for _, card := range battlezone {
		if entry, ok := byCard[card.CardID]; ok {
			own[card.CardID] = entry
		}
	}

	return own
}
