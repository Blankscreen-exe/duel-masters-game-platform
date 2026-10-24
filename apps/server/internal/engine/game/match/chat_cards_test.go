package match

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"reflect"
	"testing"
)

const (
	chatCardsTestDragonUID     = "chat-cards-test-dragon"
	chatCardsTestDragonKingUID = "chat-cards-test-dragon-king"
)

func newChatCardsTestMatch(t *testing.T) *Match {
	t.Helper()

	AddCard(chatCardsTestDragonUID, func(card *Card) {
		card.Name = "Test Dragon"
		card.Family = []string{"Armored Dragon"}
	})
	AddCard(chatCardsTestDragonKingUID, func(card *Card) {
		card.Name = "Test Dragon King"
		card.Family = []string{"Armored Dragon"}
	})

	m := newSpawnCardTestMatch()
	t.Cleanup(m.Dispose)

	return m
}

// spawnForChat puts a card in a zone. A card cannot be spawned into the battle
// zone, so it is moved there from the hand, which notes it as recently moved.
func spawnForChat(t *testing.T, player *Player, uid string, zone string) *Card {
	t.Helper()

	spawnZone := zone
	if zone == BATTLEZONE {
		spawnZone = HAND
	}

	card, err := player.SpawnCard(uid, spawnZone)
	if err != nil {
		t.Fatalf("expected the card to spawn: %v", err)
	}

	if zone == BATTLEZONE {
		if card, err = player.MoveCard(card.ID, HAND, BATTLEZONE, card.ID); err != nil || card.Zone != BATTLEZONE {
			t.Fatalf("expected the card to move into the battle zone: %v", err)
		}
	}

	return card
}

func TestChatLinesNameTheirCards(t *testing.T) {
	t.Run("a card in play is named with its owner and where it is", func(t *testing.T) {
		m := newChatCardsTestMatch(t)
		dragon := spawnForChat(t, m.Player1.Player, chatCardsTestDragonUID, BATTLEZONE)

		got := m.mentionedCards("Player 1 summoned Test Dragon to the battle zone")
		want := []server.ChatCard{{Name: "Test Dragon", ImageID: chatCardsTestDragonUID, Owner: "Player 1", CardID: dragon.ID}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("a card no one can see is not given away", func(t *testing.T) {
		m := newChatCardsTestMatch(t)
		spawnForChat(t, m.Player2.Player, chatCardsTestDragonUID, HAND)

		got := m.mentionedCards("Test Dragon was discarded from Player 2's hand")
		want := []server.ChatCard{{Name: "Test Dragon", ImageID: chatCardsTestDragonUID, Owner: "Player 2"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("of copies both players own, the one that last moved is meant", func(t *testing.T) {
		m := newChatCardsTestMatch(t)
		spawnForChat(t, m.Player1.Player, chatCardsTestDragonUID, MANAZONE)
		theirs := spawnForChat(t, m.Player2.Player, chatCardsTestDragonUID, MANAZONE)
		m.noteRecent(theirs.ID)

		got := m.mentionedCards("Test Dragon was added to Player 2's manazone")
		if len(got) != 1 || got[0].Owner != "Player 2" || got[0].CardID != theirs.ID {
			t.Errorf("expected player 2's copy, got %+v", got)
		}
	})

	t.Run("of copies both players own, none of which moved, the owner is unknown", func(t *testing.T) {
		m := newChatCardsTestMatch(t)
		spawnForChat(t, m.Player1.Player, chatCardsTestDragonUID, MANAZONE)
		spawnForChat(t, m.Player2.Player, chatCardsTestDragonUID, MANAZONE)

		got := m.mentionedCards("Test Dragon was tapped")
		want := []server.ChatCard{{Name: "Test Dragon", ImageID: chatCardsTestDragonUID}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("a card named twice in a line is two different copies", func(t *testing.T) {
		m := newChatCardsTestMatch(t)
		loser := spawnForChat(t, m.Player1.Player, chatCardsTestDragonUID, BATTLEZONE)
		winner := spawnForChat(t, m.Player2.Player, chatCardsTestDragonUID, BATTLEZONE)
		m.noteRecent(winner.ID)
		m.noteRecent(loser.ID)

		got := m.mentionedCards("Test Dragon (2000) was destroyed by Test Dragon (3000)")
		if len(got) != 2 || got[0].CardID != loser.ID || got[1].CardID != winner.ID {
			t.Errorf("expected the loser then the winner, got %+v", got)
		}
	})

	t.Run("the longest name is found rather than a name inside it", func(t *testing.T) {
		m := newChatCardsTestMatch(t)
		spawnForChat(t, m.Player1.Player, chatCardsTestDragonUID, BATTLEZONE)
		spawnForChat(t, m.Player1.Player, chatCardsTestDragonKingUID, BATTLEZONE)

		got := m.mentionedCards("Test Dragon King was given +2000 by Test Dragon")
		if len(got) != 2 || got[0].Name != "Test Dragon King" || got[1].Name != "Test Dragon" {
			t.Errorf("expected both names in order, got %+v", got)
		}
	})

	t.Run("a name is only found as a whole word", func(t *testing.T) {
		m := newChatCardsTestMatch(t)
		spawnForChat(t, m.Player1.Player, chatCardsTestDragonUID, BATTLEZONE)

		if got := m.mentionedCards("Test Dragons were shuffled"); got != nil {
			t.Errorf("expected no mention, got %+v", got)
		}
	})

	t.Run("a line naming no card has none", func(t *testing.T) {
		m := newChatCardsTestMatch(t)
		spawnForChat(t, m.Player1.Player, chatCardsTestDragonUID, BATTLEZONE)

		if got := m.mentionedCards("Your turn, Player 1"); got != nil {
			t.Errorf("expected no mention, got %+v", got)
		}
	})
}

func TestRecentCardsKeepOnlyTheLatest(t *testing.T) {
	m := newChatCardsTestMatch(t)

	for i := 0; i < recentCardsKept+4; i++ {
		m.noteRecent(string(rune('a' + i)))
	}
	m.noteRecent("c")

	rank := m.recency()
	if len(rank) != recentCardsKept {
		t.Fatalf("expected %d cards kept, got %d", recentCardsKept, len(rank))
	}
	if rank["c"] != recentCardsKept {
		t.Errorf("expected a card noted again to be the most recent, got rank %d", rank["c"])
	}
	if _, kept := rank["a"]; kept {
		t.Error("expected the oldest card to be forgotten")
	}
}
