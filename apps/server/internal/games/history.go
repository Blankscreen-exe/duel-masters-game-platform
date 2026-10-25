package games

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// History limits.
const (
	HistoryPageSize    = 20
	MaxHistoryPageSize = 100
	// favouriteSample is how many of a player's latest games their favourite
	// civilizations are counted over.
	favouriteSample = 50
)

// ErrNotRecorded is returned for a game the history does not hold: one that
// never existed, never started, or is still running or being saved.
var ErrNotRecorded = errors.New("no finished game with this id")

// HistoryPage is one page of a player's finished games, the latest first.
type HistoryPage struct {
	Games    []Record
	Total    int
	Page     int
	PageSize int
}

// CivilizationCount is how many of a player's games they played a
// civilization in.
type CivilizationCount struct {
	Civilization string
	Games        int
}

// Recorded returns a finished game from the history. It returns
// ErrNotRecorded for a game the history does not hold.
func (s *Service) Recorded(ctx context.Context, gameID string) (Record, error) {
	var row gameRow
	err := s.db.WithContext(ctx).Where("id = ?", gameID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Record{}, ErrNotRecorded
	}
	if err != nil {
		return Record{}, fmt.Errorf("reading a finished game: %w", err)
	}

	records, err := s.withNames(ctx, []gameRow{row})
	if err != nil {
		return Record{}, err
	}
	return records[0], nil
}

// History returns one page of the games a player finished, every one of
// them, counted or not, the latest first. It returns a *validation.Error for
// a page out of range.
func (s *Service) History(ctx context.Context, userID string, page paging.Request) (HistoryPage, error) {
	if err := page.Normalize(HistoryPageSize, MaxHistoryPageSize); err != nil {
		return HistoryPage{}, err
	}

	played := s.played(ctx, userID)
	var total int64
	if err := played.Count(&total).Error; err != nil {
		return HistoryPage{}, fmt.Errorf("counting a player's games: %w", err)
	}

	var rows []gameRow
	if err := played.Order("ended_at DESC, id DESC").Limit(page.PageSize).Offset(page.Offset()).Find(&rows).Error; err != nil {
		return HistoryPage{}, fmt.Errorf("reading a player's games: %w", err)
	}

	records, err := s.withNames(ctx, rows)
	if err != nil {
		return HistoryPage{}, err
	}
	return HistoryPage{Games: records, Total: int(total), Page: page.Page, PageSize: page.PageSize}, nil
}

// FavouriteCivilizations counts, over a player's latest games, how many they
// played each civilization in: the most played first, then in the game's
// order.
func (s *Service) FavouriteCivilizations(ctx context.Context, userID string) ([]CivilizationCount, error) {
	var rows []gameRow
	err := s.played(ctx, userID).Select("host_id", "host_civilizations", "guest_civilizations").
		Order("ended_at DESC, id DESC").Limit(favouriteSample).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("reading a player's latest games: %w", err)
	}

	counts := map[string]int{}
	for _, row := range rows {
		civilizations := row.GuestCivilizations
		if row.HostID == userID {
			civilizations = row.HostCivilizations
		}
		for _, civilization := range civilizations {
			counts[civilization]++
		}
	}

	favourites := make([]CivilizationCount, 0, len(counts))
	for civilization, games := range counts {
		favourites = append(favourites, CivilizationCount{Civilization: civilization, Games: games})
	}
	slices.SortFunc(favourites, func(a, b CivilizationCount) int {
		return cmp.Or(cmp.Compare(b.Games, a.Games), decks.CompareCivilizations(a.Civilization, b.Civilization))
	})
	return favourites, nil
}

// withNames is the finished games rows keep, with their players' names.
func (s *Service) withNames(ctx context.Context, rows []gameRow) ([]Record, error) {
	ids := make([]string, 0, 2*len(rows))
	for _, row := range rows {
		ids = append(ids, row.HostID, row.GuestID)
	}

	var players []users.User
	if err := s.db.WithContext(ctx).Select("id", "username").Where("id IN ?", slices.Compact(slices.Sorted(slices.Values(ids)))).Find(&players).Error; err != nil {
		return nil, fmt.Errorf("reading the names of a game's players: %w", err)
	}
	names := make(map[string]string, len(players))
	for _, player := range players {
		names[player.ID] = player.Username
	}

	records := make([]Record, 0, len(rows))
	for _, row := range rows {
		finished := row.record()
		finished.Host.Username, finished.Guest.Username = names[row.HostID], names[row.GuestID]
		records = append(records, finished)
	}
	return records, nil
}

// played is the games a player played in.
func (s *Service) played(ctx context.Context, userID string) *gorm.DB {
	return s.db.WithContext(ctx).Model(&gameRow{}).Where("host_id = ? OR guest_id = ?", userID, userID)
}

// record is the finished game a row keeps, without its players' names.
func (row gameRow) record() Record {
	finished := Record{
		ID:         row.ID,
		Name:       row.Name,
		FormatName: row.FormatName,
		Host: RecordPlayer{
			ID:            row.HostID,
			DeckName:      row.HostDeckName,
			Deck:          entries(row.HostDeck),
			Civilizations: row.HostCivilizations,
			Rating:        ratingChange(row.HostRatingChange, row.HostRating),
		},
		Guest: RecordPlayer{
			ID:            row.GuestID,
			DeckName:      row.GuestDeckName,
			Deck:          entries(row.GuestDeck),
			Civilizations: row.GuestCivilizations,
			Rating:        ratingChange(row.GuestRatingChange, row.GuestRating),
		},
		Ending:      Ending(row.Ending),
		Turns:       row.Turns,
		StartedAt:   row.StartedAt.UTC(),
		EndedAt:     row.EndedAt.UTC(),
		Counted:     row.Counted,
		Visible:     row.Visible,
		Matchmaking: row.Matchmaking,
	}
	if row.FormatID != nil {
		finished.FormatID = *row.FormatID
	}
	if row.WinnerID != nil {
		finished.WinnerID = *row.WinnerID
	}
	return finished
}

func entries(stored []storedEntry) []decks.Entry {
	deck := make([]decks.Entry, 0, len(stored))
	for _, entry := range stored {
		deck = append(deck, decks.Entry{CardID: entry.CardID, Count: entry.Count})
	}
	return deck
}

func ratingChange(change, rating *int) *RatingChange {
	if change == nil || rating == nil {
		return nil
	}
	return &RatingChange{Change: *change, Rating: *rating}
}
