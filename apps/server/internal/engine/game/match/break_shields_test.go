package match

import (
	"slices"
	"testing"
)

const (
	breakShieldsTestShieldUID = "break-shields-test-shield"
	breakShieldsTestGuardUID  = "break-shields-test-guard"
)

// registerBreakShieldsTestConstructors registers a plain card and a guard
// whose effect, from any zone, takes the shields kept by keep out of every
// shield break without cancelling it, as a card that saves some shields might.
func registerBreakShieldsTestConstructors(keep func(shields []*Card) []*Card) {
	AddCard(breakShieldsTestShieldUID, func(card *Card) {
		card.Name = "Break Shields Test Shield"
	})
	AddCard(breakShieldsTestGuardUID, func(card *Card) {
		card.Name = "Break Shields Test Guard"
		card.Use(func(card *Card, ctx *Context) {
			if event, ok := ctx.Event.(*BreakShieldEvent); ok {
				event.Cards = keep(event.Cards)
			}
		})
	})
}

// newBreakShieldsTestMatch returns a match whose second player has the guard
// in hand and the given number of shields, and a card of the first player to
// break them with.
func newBreakShieldsTestMatch(t *testing.T, shieldCount int) (m *Match, shields []*Card, source *Card) {
	t.Helper()

	m = newSpawnCardTestMatch()
	defender := m.Player2.Player

	if _, err := defender.SpawnCard(breakShieldsTestGuardUID, HAND); err != nil {
		t.Fatal(err)
	}
	for range shieldCount {
		shield, err := defender.SpawnCard(breakShieldsTestShieldUID, SHIELDZONE)
		if err != nil {
			t.Fatal(err)
		}
		shields = append(shields, shield)
	}

	source, err := m.Player1.Player.SpawnCard(breakShieldsTestShieldUID, HAND)
	if err != nil {
		t.Fatal(err)
	}

	return m, shields, source
}

func TestBreakingShieldsAnEffectTookEveryOneOfChangesNothing(t *testing.T) {
	registerBreakShieldsTestConstructors(func([]*Card) []*Card { return nil })
	m, shields, source := newBreakShieldsTestMatch(t, 2)

	m.BreakShields(slices.Clone(shields), source)

	for _, shield := range shields {
		if shield.Zone != SHIELDZONE {
			t.Fatalf("shield %s moved to %s, want it left in the shield zone", shield.ID, shield.Zone)
		}
	}
	if len(m.brokenShields) != 0 {
		t.Fatalf("broken shields reported: %v, want none", m.brokenShields)
	}
}

func TestBreakingShieldsBreaksOnlyTheOnesAnEffectLeft(t *testing.T) {
	registerBreakShieldsTestConstructors(func(shields []*Card) []*Card { return shields[1:] })
	m, shields, source := newBreakShieldsTestMatch(t, 2)

	m.BreakShields(slices.Clone(shields), source)

	if shields[0].Zone != SHIELDZONE {
		t.Fatalf("the saved shield moved to %s, want it left in the shield zone", shields[0].Zone)
	}
	if shields[1].Zone != HAND {
		t.Fatalf("the other shield is in %s, want it broken into the hand", shields[1].Zone)
	}
	if !slices.Equal(m.brokenShields, []string{shields[1].ID}) {
		t.Fatalf("broken shields reported: %v, want only %s", m.brokenShields, shields[1].ID)
	}
}
