package games

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

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
	// Deck is the deck as played, and Civilizations its civilizations, in
	// the game's order.
	Deck          []decks.Entry
	Civilizations []string
	// Computer marks a seat the computer played.
	Computer bool
	// Rating is what the game did to the player's rating: nil for a game
	// that did not count.
	Rating *RatingChange
}

// RatingChange is what a game did to a player's rating.
type RatingChange struct {
	// Change is the points won, above zero, or lost, below.
	Change int
	// Rating is the rating after the game.
	Rating int
}

// Rater rates the games that count, in the transaction that saves them, so
// a game is never saved without its rating changes, nor rated unsaved.
type Rater interface {
	// Rate applies a won game that counted to both players' ratings and
	// returns what it did to the host's and to the guest's.
	Rate(tx *gorm.DB, game Record) (host, guest RatingChange, err error)
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
	// Counted is false for a game too short to count, aborted, or against the
	// computer.
	Counted     bool
	Visible     bool
	Matchmaking bool
}

// Duration is how long the game lasted.
func (r Record) Duration() time.Duration {
	return max(r.EndedAt.Sub(r.StartedAt), 0)
}

// Rated reports whether the game moves its players' ratings: it counted, and
// one of them won.
func (r Record) Rated() bool {
	return r.Counted && r.WinnerID != ""
}

// gameRow is a row of the games table.
type gameRow struct {
	ID           string `gorm:"primaryKey"`
	Name         string
	FormatID     *string
	FormatName   string
	HostID       string
	HostDeckName string
	HostDeck     []storedEntry `gorm:"serializer:json"`
	// HostCivilizations and GuestCivilizations are empty for games finished
	// before history kept them.
	HostCivilizations  []string `gorm:"serializer:json"`
	HostRatingChange   *int
	HostRating         *int
	GuestID            string
	GuestDeckName      string
	GuestDeck          []storedEntry `gorm:"serializer:json"`
	GuestCivilizations []string      `gorm:"serializer:json"`
	GuestRatingChange  *int
	GuestRating        *int
	WinnerID           *string
	Ending             string
	Turns              int
	StartedAt          time.Time
	EndedAt            time.Time
	Counted            bool
	Visible            bool
	Matchmaking        bool
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

// record is a finished game as history keeps it, with its times as the
// database stores them, so the players are told of it as history shows it.
func record(done finished) Record {
	result, game := done.result, done.live.game

	return Record{
		ID:          result.MatchID,
		Name:        game.Name,
		FormatID:    game.FormatID,
		FormatName:  game.FormatName,
		Host:        recordPlayer(game.Host, done.live.hostDeck),
		Guest:       recordPlayer(game.Guest, done.live.guestDeck),
		WinnerID:    result.WinnerID,
		Ending:      Ending(result.Ending),
		Turns:       result.Turns,
		StartedAt:   result.StartedAt.UTC().Truncate(time.Microsecond),
		EndedAt:     result.EndedAt.UTC().Truncate(time.Microsecond),
		Counted:     result.Counted,
		Visible:     game.Visible,
		Matchmaking: game.Matchmaking,
	}
}

func recordPlayer(player Player, deck []decks.Entry) RecordPlayer {
	return RecordPlayer{
		ID:            player.ID,
		Username:      player.Username,
		DeckName:      player.DeckName,
		Deck:          deck,
		Civilizations: player.Civilizations,
		Computer:      player.Computer,
	}
}

// save writes a finished game to the history, rating it if it counts, and
// tells the listeners. A game that cannot be saved is logged with everything
// needed to save it by hand.
func (s *Service) save(done finished) {
	finishedGame := record(done)

	ctx, cancel := context.WithTimeout(context.Background(), saveTimeout)
	defer cancel()

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if finishedGame.Rated() {
			host, guest, err := s.rater.Rate(tx, finishedGame)
			if err != nil {
				return fmt.Errorf("rating the game: %w", err)
			}
			finishedGame.Host.Rating, finishedGame.Guest.Rating = &host, &guest
		}

		row := newGameRow(finishedGame)
		return tx.Create(&row).Error
	})
	if err != nil {
		s.logger.Error("saving a finished game failed", "error", err, "game", finishedGame)
		return
	}

	s.logger.Info("a game finished",
		"game", finishedGame.ID, "winner", finishedGame.WinnerID, "ending", finishedGame.Ending,
		"turns", finishedGame.Turns, "duration", finishedGame.Duration().Round(time.Second), "counted", finishedGame.Counted)

	s.notify(func(l Listener) { l.GameFinished(finishedGame) })
}

// newGameRow is a finished game as the games table keeps it.
func newGameRow(finishedGame Record) gameRow {
	row := gameRow{
		ID:                 finishedGame.ID,
		Name:               finishedGame.Name,
		FormatName:         finishedGame.FormatName,
		HostID:             finishedGame.Host.ID,
		HostDeckName:       finishedGame.Host.DeckName,
		HostDeck:           storedDeck(finishedGame.Host.Deck),
		HostCivilizations:  finishedGame.Host.Civilizations,
		GuestID:            finishedGame.Guest.ID,
		GuestDeckName:      finishedGame.Guest.DeckName,
		GuestDeck:          storedDeck(finishedGame.Guest.Deck),
		GuestCivilizations: finishedGame.Guest.Civilizations,
		Ending:             string(finishedGame.Ending),
		Turns:              finishedGame.Turns,
		StartedAt:          finishedGame.StartedAt,
		EndedAt:            finishedGame.EndedAt,
		Counted:            finishedGame.Counted,
		Visible:            finishedGame.Visible,
		Matchmaking:        finishedGame.Matchmaking,
	}
	if finishedGame.FormatID != "" {
		row.FormatID = &finishedGame.FormatID
	}
	if finishedGame.WinnerID != "" {
		row.WinnerID = &finishedGame.WinnerID
	}
	if rating := finishedGame.Host.Rating; rating != nil {
		row.HostRatingChange, row.HostRating = &rating.Change, &rating.Rating
	}
	if rating := finishedGame.Guest.Rating; rating != nil {
		row.GuestRatingChange, row.GuestRating = &rating.Change, &rating.Rating
	}
	return row
}
