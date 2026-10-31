package match

import "github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"

// PromptKind is what a question is about, told with it
// (server.PromptAbout.Kind) for programs that answer questions as a player
// does, such as the computer opponent.
type PromptKind string

const (
	// PromptEffect is a question a card's effect asks, such as which
	// creature to destroy; its source tells which card asks.
	PromptEffect PromptKind = "effect"
	// PromptPayMana asks which mana pays for the card being played.
	PromptPayMana PromptKind = "pay_mana"
	// PromptAttackTarget asks which creature an attacker attacks.
	PromptAttackTarget PromptKind = "attack_target"
	// PromptConfirmAttack asks an attacker's player to confirm an attack that
	// cannot be taken back once confirmed.
	PromptConfirmAttack PromptKind = "confirm_attack"
	// PromptBreakShields asks which of the opponent's shields an attack
	// breaks.
	PromptBreakShields PromptKind = "break_shields"
	// PromptBlock asks the attacked player whether to block, and with which
	// creature.
	PromptBlock PromptKind = "block"
	// PromptShieldTrigger asks which shield trigger, if any, to play for
	// free.
	PromptShieldTrigger PromptKind = "shield_trigger"
	// PromptEvolve asks which creature an evolution creature evolves from.
	PromptEvolve PromptKind = "evolve"
	// PromptTapAbility asks which of several tap abilities to use.
	PromptTapAbility PromptKind = "tap_ability"
)

// NewBlockAction asks the attacked player whether to block an attack, and with
// which of the blockers. The target is the creature attacked, or nil when the
// player is attacked. Both creatures are in play, so telling them gives nothing
// away.
func (m *Match) NewBlockAction(player *Player, blockers []*Card, attacker *Card, target *Card) {
	msg := m.actionMessage(player, PromptBlock, blockers, 1, 1, "Choose a creature to block the attack with or close to not block the attack.", true, nil)
	msg.Attack = &server.PromptAttack{
		Attacker: server.PromptFighter{Name: attacker.Name, Power: m.GetPower(attacker, true)},
	}

	if target != nil {
		msg.Attack.Target = &server.PromptFighter{Name: target.Name, Power: m.GetPower(target, false)}
	}

	m.ask(player, msg)
}

// promptAbout says what a question to player is about, offering cards.
func (m *Match) promptAbout(player *Player, kind PromptKind, cards []*Card) server.PromptAbout {
	about := server.PromptAbout{Kind: string(kind)}

	// The card whose effect asks is told only when the player may know it,
	// so a hidden card of the opponent's is never given away.
	if source := m.effectSource(); source != nil && (source.Player == player || publicZones[source.Zone]) {
		about.Source = &server.PromptSource{CardID: source.ID, ImageID: source.ImageID, Name: source.Name}
	}

	for _, card := range cards {
		if card.Player != player {
			about.OpponentCards = append(about.OpponentCards, card.ID)
		}
	}

	return about
}
