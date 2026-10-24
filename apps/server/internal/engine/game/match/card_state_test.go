package match

import "testing"

const (
	cardStateTestCreatureUID = "card-state-test-creature"
	cardStateTestSpellUID    = "card-state-test-spell"
)

func TestDenormalizedCardsNameTheirType(t *testing.T) {
	AddCard(cardStateTestCreatureUID, func(card *Card) {
		card.Name = "Card State Test Creature"
		card.Family = []string{"Liquid People"}
	})
	AddCard(cardStateTestSpellUID, func(card *Card) {
		card.Name = "Card State Test Spell"
	})

	m := newSpawnCardTestMatch()
	defer m.Dispose()

	creature, err := m.Player1.Player.SpawnCard(cardStateTestCreatureUID, HAND)
	if err != nil {
		t.Fatalf("expected the creature to spawn: %v", err)
	}
	spell, err := m.Player1.Player.SpawnCard(cardStateTestSpellUID, HAND)
	if err != nil {
		t.Fatalf("expected the spell to spawn: %v", err)
	}

	visible := denormalizeCards([]*Card{creature, spell}, false)
	if visible[0].Type != "creature" {
		t.Errorf("expected a card with a race to be a creature, got %q", visible[0].Type)
	}
	if visible[1].Type != "spell" {
		t.Errorf("expected a card without a race to be a spell, got %q", visible[1].Type)
	}

	hidden := denormalizeCards([]*Card{creature}, true)
	if hidden[0].Type != "" {
		t.Errorf("expected a face down card not to reveal its type, got %q", hidden[0].Type)
	}
}
