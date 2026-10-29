package match

import (
	"slices"
	"testing"
)

const (
	effectSourceOuterUID = "effect-source-test-outer"
	effectSourceInnerUID = "effect-source-test-inner"
)

// The events the effect source test cards answer.
type (
	effectSourceOuterEvent     struct{}
	effectSourceInnerEvent     struct{}
	effectSourceScheduleEvent  struct{}
	effectSourcePersistedEvent struct{}
	effectSourceLaterEvent     struct{}
)

// effectSourceSeen records which card each effect source test card saw as the
// effect source, in order.
type effectSourceSeen struct {
	match *Match
	seen  []string
}

func (s *effectSourceSeen) note(label string) {
	source := s.match.effectSource()
	name := "none"
	if source != nil {
		name = source.Name
	}
	s.seen = append(s.seen, label+":"+name)
}

// newEffectSourceTestMatch returns a match whose first player holds the
// outer test card and the inner one, which note what they see.
func newEffectSourceTestMatch(t *testing.T) (*Match, *effectSourceSeen) {
	t.Helper()

	m := newSpawnCardTestMatch()
	seen := &effectSourceSeen{match: m}

	AddCard(effectSourceOuterUID, func(card *Card) {
		card.Name = "Outer"
		card.Use(func(card *Card, ctx *Context) {
			switch ctx.Event.(type) {
			case *effectSourceOuterEvent:
				seen.note("outer before")
				ctx.Match.HandleFx(NewContext(ctx.Match, &effectSourceInnerEvent{}))
				seen.note("outer after")
			case *effectSourceScheduleEvent:
				ctx.ScheduleAfter(func() { seen.note("scheduled") })
			case *effectSourcePersistedEvent:
				ctx.Match.ApplyPersistentEffect(func(ctx *Context, exit func()) {
					if _, ok := ctx.Event.(*effectSourceLaterEvent); ok {
						seen.note("persistent")
						exit()
					}
				})
			}
		})
	})
	AddCard(effectSourceInnerUID, func(card *Card) {
		card.Name = "Inner"
		card.Use(func(card *Card, ctx *Context) {
			switch ctx.Event.(type) {
			case *effectSourceInnerEvent:
				seen.note("inner")
			case *effectSourceScheduleEvent:
				seen.note("inner handler")
			}
		})
	})

	for _, uid := range []string{effectSourceOuterUID, effectSourceInnerUID} {
		if _, err := m.Player1.Player.SpawnCard(uid, HAND); err != nil {
			t.Fatal(err)
		}
	}
	// A player with no deck loses at the end of the first event handled.
	registerSpawnCardTestConstructor()
	for _, player := range []*Player{m.Player1.Player, m.Player2.Player} {
		if _, err := player.SpawnCard(spawnCardTestUID, DECK); err != nil {
			t.Fatal(err)
		}
	}

	return m, seen
}

func assertSeen(t *testing.T, seen *effectSourceSeen, want ...string) {
	t.Helper()

	if !slices.Equal(seen.seen, want) {
		t.Fatalf("effect sources seen: %v, want %v", seen.seen, want)
	}
}

func TestAnEffectRunsAsItsCardsAndNestedEffectsAsTheirs(t *testing.T) {
	m, seen := newEffectSourceTestMatch(t)

	m.HandleFx(NewContext(m, &effectSourceOuterEvent{}))
	seen.note("outside")

	assertSeen(t, seen, "outer before:Outer", "inner:Inner", "outer after:Outer", "outside:none")
}

func TestAScheduledCallbackRunsAsTheCardThatScheduledIt(t *testing.T) {
	m, seen := newEffectSourceTestMatch(t)

	m.HandleFx(NewContext(m, &effectSourceScheduleEvent{}))

	assertSeen(t, seen, "inner handler:Inner", "scheduled:Outer")
}

func TestAPersistentEffectRunsAsTheCardThatAddedIt(t *testing.T) {
	m, seen := newEffectSourceTestMatch(t)

	m.HandleFx(NewContext(m, &effectSourcePersistedEvent{}))
	m.HandleFx(NewContext(m, &effectSourceLaterEvent{}))

	assertSeen(t, seen, "persistent:Outer")
}

func TestAnEffectAbandonedByAPanicLeavesNoSourceBehind(t *testing.T) {
	m, _ := newEffectSourceTestMatch(t)
	AddCard(effectSourceOuterUID, func(card *Card) {
		card.Name = "Outer"
		card.Use(func(*Card, *Context) { panic("abandoned") })
	})
	if _, err := m.Player1.Player.SpawnCard(effectSourceOuterUID, HAND); err != nil {
		t.Fatal(err)
	}

	func() {
		defer func() { _ = recover() }()
		m.HandleFx(NewContext(m, &effectSourceOuterEvent{}))
	}()

	if source := m.effectSource(); source != nil {
		t.Fatalf("effect source after the panic: %s, want none", source.Name)
	}
}

func TestAQuestionNamesItsSourceOnlyWhenThePlayerMayKnowIt(t *testing.T) {
	m := newSpawnCardTestMatch()
	registerSpawnCardTestConstructor()
	asker, opponent := m.Player1.Player, m.Player2.Player

	theirs, err := opponent.SpawnCard(spawnCardTestUID, HAND)
	if err != nil {
		t.Fatal(err)
	}
	mine, err := asker.SpawnCard(spawnCardTestUID, HAND)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		what   string
		source *Card
		zone   string
		named  bool
	}{
		{what: "the opponent's card in hand", source: theirs, zone: HAND, named: false},
		{what: "the opponent's card in a shield", source: theirs, zone: SHIELDZONE, named: false},
		{what: "the opponent's card in the mana zone", source: theirs, zone: MANAZONE, named: true},
		{what: "the opponent's card in the battle zone", source: theirs, zone: BATTLEZONE, named: true},
		{what: "the player's own card in hand", source: mine, zone: HAND, named: true},
	} {
		tc.source.Zone = tc.zone
		depth := m.enterEffect(tc.source)
		about := m.promptAbout(asker, PromptEffect, []*Card{mine, theirs})
		m.leaveEffect(depth)

		if named := about.Source != nil; named != tc.named {
			t.Errorf("%s: named %v, want %v", tc.what, named, tc.named)
		}
		if !slices.Equal(about.OpponentCards, []string{theirs.ID}) {
			t.Errorf("%s: opponent cards %v, want only %s", tc.what, about.OpponentCards, theirs.ID)
		}
	}
}
