package games

import (
	"context"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// saveTimeout bounds saving one finished game, so a stuck database cannot
// hold up the games after it for ever.
const saveTimeout = 30 * time.Second

// Ending is how a game ended.
type Ending string

const (
	EndingPlay       Ending = Ending(match.EndingPlay)
	EndingSurrender  Ending = Ending(match.EndingSurrender)
	EndingDisconnect Ending = Ending(match.EndingDisconnect)
	EndingAborted    Ending = Ending(match.EndingAborted)
)

// RecordPlayer is a player as a finished game keeps them.
type RecordPlayer struct {
	ID       string
	Username string
	DeckName string
	// Deck is the deck as played.
	Deck []decks.Entry
}

// Record is a finished game, as match history keeps it.
type Record struct {
	ID         string
	Name       string
	FormatID   string
	FormatName string
	Host       RecordPlayer
	Guest      RecordPlayer
	// WinnerID is empty for a game without a winner.
	WinnerID string
	Ending   Ending
	Turns    int
	// StartedAt is when the coin toss was settled and play began.
	StartedAt time.Time
	EndedAt   time.Time
	// Counted is false for a game too short to count, or aborted.
	Counted     bool
	Visible     bool
	Matchmaking bool
}

// Duration is how long the game lasted.
func (r Record) Duration() time.Duration {
	return max(r.EndedAt.Sub(r.StartedAt), 0)
}

// gameRow is a row of the games table.
type gameRow struct {
	ID            string `gorm:"primaryKey"`
	Name          string
	FormatID      *string
	FormatName    string
	HostID        string
	HostDeckName  string
	HostDeck      []storedEntry `gorm:"serializer:json"`
	GuestID       string
	GuestDeckName string
	GuestDeck     []storedEntry `gorm:"serializer:json"`
	WinnerID      *string
	Ending        string
	Turns         int
	StartedAt     time.Time
	EndedAt       time.Time
	Counted       bool
	Visible       bool
	Matchmaking   bool
}

func (gameRow) TableName() string { return "games" }

// storedEntry is a deck entry as the games table keeps it, in JSON. It is a
// type of its own so the stored format never changes with the decks package.
type storedEntry struct {
	CardID string `json:"cardId"`
	Count  int    `json:"count"`
}

func storedDeck(entries []decks.Entry) []storedEntry {
	stored := make([]storedEntry, 0, len(entries))
	for _, entry := range entries {
		stored = append(stored, storedEntry{CardID: entry.CardID, Count: entry.Count})
	}
	return stored
}

// record is a finished game as history keeps it.
func record(done finished) Record {
	result, game := done.result, done.live.game

	return Record{
		ID:          result.MatchID,
		Name:        game.Name,
		FormatID:    game.FormatID,
		FormatName:  game.FormatName,
		Host:        RecordPlayer{ID: game.Host.ID, Username: game.Host.Username, DeckName: game.Host.DeckName, Deck: done.live.hostDeck},
		Guest:       RecordPlayer{ID: game.Guest.ID, Username: game.Guest.Username, DeckName: game.Guest.DeckName, Deck: done.live.guestDeck},
		WinnerID:    result.WinnerID,
		Ending:      Ending(result.Ending),
		Turns:       result.Turns,
		StartedAt:   result.StartedAt.UTC(),
		EndedAt:     result.EndedAt.UTC(),
		Counted:     result.Counted,
		Visible:     game.Visible,
		Matchmaking: game.Matchmaking,
	}
}

// save writes a finished game to the history and tells the listener. A game
// that cannot be saved is logged with everything needed to save it by hand.
func (s *Service) save(done finished) {
	finishedGame := record(done)

	row := gameRow{
		ID:            finishedGame.ID,
		Name:          finishedGame.Name,
		FormatName:    finishedGame.FormatName,
		HostID:        finishedGame.Host.ID,
		HostDeckName:  finishedGame.Host.DeckName,
		HostDeck:      storedDeck(finishedGame.Host.Deck),
		GuestID:       finishedGame.Guest.ID,
		GuestDeckName: finishedGame.Guest.DeckName,
		GuestDeck:     storedDeck(finishedGame.Guest.Deck),
		Ending:        string(finishedGame.Ending),
		Turns:         finishedGame.Turns,
		StartedAt:     finishedGame.StartedAt,
		EndedAt:       finishedGame.EndedAt,
		Counted:       finishedGame.Counted,
		Visible:       finishedGame.Visible,
		Matchmaking:   finishedGame.Matchmaking,
	}
	if finishedGame.FormatID != "" {
		row.FormatID = &finishedGame.FormatID
	}
	if finishedGame.WinnerID != "" {
		row.WinnerID = &finishedGame.WinnerID
	}

	ctx, cancel := context.WithTimeout(context.Background(), saveTimeout)
	defer cancel()

	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		s.logger.Error("saving a finished game failed", "error", err, "game", finishedGame)
		return
	}

	s.logger.Info("a game finished",
		"game", finishedGame.ID, "winner", finishedGame.WinnerID, "ending", finishedGame.Ending,
		"turns", finishedGame.Turns, "duration", finishedGame.Duration().Round(time.Second), "counted", finishedGame.Counted)

	s.notify(func(l Listener) { l.GameFinished(finishedGame) })
}
