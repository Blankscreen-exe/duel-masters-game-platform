package match

import (
	"testing"
	"time"
)

func finishedTestMatch(minimum time.Duration) *Match {
	host := &PlayerReference{UID: "host-1", Username: "Alice", Player: &Player{}}
	guest := &PlayerReference{UID: "guest-2", Username: "Bob", Player: &Player{}}

	return &Match{
		ID:                     "duel-123",
		MatchName:              "Friendly",
		FormatID:               "format-open",
		FormatName:             "Open",
		HostID:                 host.UID,
		HostDeck:               []string{"card-a"},
		GuestID:                guest.UID,
		GuestDeck:              []string{"card-b"},
		hostUsername:           "Alice",
		guestUsername:          "Bob",
		Player1:                host,
		Player2:                guest,
		turnsPlayed:            14,
		startedAt:              time.Unix(100, 0),
		minimumCountedDuration: minimum,
	}
}

func TestResultDescribesTheFinishedGame(t *testing.T) {
	m := finishedTestMatch(time.Minute)

	result := m.result(m.Player2.Player, EndingSurrender, time.Unix(190, 0))

	if result.MatchID != "duel-123" || result.Name != "Friendly" || result.FormatID != "format-open" || result.FormatName != "Open" {
		t.Fatalf("expected the match's details, got %+v", result)
	}
	if result.Host.ID != "host-1" || result.Host.Username != "Alice" || len(result.Host.Deck) != 1 {
		t.Fatalf("expected the host, got %+v", result.Host)
	}
	if result.Guest.ID != "guest-2" || result.Guest.Username != "Bob" {
		t.Fatalf("expected the guest, got %+v", result.Guest)
	}
	if result.WinnerID != "guest-2" {
		t.Fatalf("expected Bob to win, got %q", result.WinnerID)
	}
	if result.Ending != EndingSurrender || result.Turns != 14 {
		t.Fatalf("expected a surrender after 14 turns, got %+v", result)
	}
	if result.Duration() != 90*time.Second || !result.Counted {
		t.Fatalf("expected a counted 90-second game, got %v counted %v", result.Duration(), result.Counted)
	}
}

func TestResultWithoutAWinner(t *testing.T) {
	m := finishedTestMatch(time.Minute)

	if winner := m.result(nil, EndingPlay, time.Unix(190, 0)).WinnerID; winner != "" {
		t.Fatalf("expected no winner, got %q", winner)
	}
}

func TestAGameShorterThanTheMinimumDoesNotCount(t *testing.T) {
	m := finishedTestMatch(time.Minute)

	if m.result(nil, EndingPlay, time.Unix(159, 0)).Counted {
		t.Fatal("a 59-second game must not count")
	}
	if !m.result(nil, EndingPlay, time.Unix(160, 0)).Counted {
		t.Fatal("a 60-second game counts")
	}
}

func TestDuelFinishedMessageTellsHowTheGameEnded(t *testing.T) {
	m := finishedTestMatch(time.Minute)

	message := m.duelFinishedMessage(m.result(m.Player1.Player, EndingDisconnect, time.Unix(190, 0)))

	if message.Header != "duel_finished" || message.DuelID != "duel-123" {
		t.Fatalf("expected a duel_finished message for duel-123, got %+v", message)
	}
	if message.Winner == nil || message.Winner.UID != "host-1" || message.Winner.Username != "Alice" {
		t.Fatalf("expected Alice to win, got %+v", message.Winner)
	}
	if message.Ending != "disconnect" || !message.Counted || message.Turns != 14 || message.DurationSeconds != 90 {
		t.Fatalf("expected a counted 90-second disconnect after 14 turns, got %+v", message)
	}

	if none := m.duelFinishedMessage(m.result(nil, EndingPlay, time.Unix(190, 0))); none.Winner != nil {
		t.Fatalf("expected no winner, got %+v", none.Winner)
	}
}

func TestADurationIsNeverNegative(t *testing.T) {
	result := Result{StartedAt: time.Unix(200, 0), EndedAt: time.Unix(100, 0)}

	if result.Duration() != 0 {
		t.Fatalf("expected 0, got %v", result.Duration())
	}
}
