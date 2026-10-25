package ratings

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Rankings limits.
const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// ErrNotFound is returned for a player who does not exist.
var ErrNotFound = errors.New("player not found")

// Standing is where a player stands: their rank, wins, losses and streak.
type Standing struct {
	UserID   string
	Username string
	Rank     Rank
	Wins     int
	Losses   int
	// Streak is the current run of wins, above zero, or of losses, below.
	Streak int
}

// Ranked reports whether the player is in the rankings: they have finished
// a game that counted.
func (s Standing) Ranked() bool {
	return s.Wins+s.Losses > 0
}

// Placed is a ranked player with their place in the rankings. Players with
// the same rating share a place, and the next place skips as many: 1, 2, 2,
// 4.
type Placed struct {
	Standing
	Place int
}

// RankingsQuery asks for one page of the rankings.
type RankingsQuery struct {
	// Search keeps the players whose name contains it, ignoring case. They
	// keep their places in the full rankings.
	Search string
	paging.Request
}

// Rankings is one page of the ranked players, the best first.
type Rankings struct {
	Players  []Placed
	Total    int
	Page     int
	PageSize int
}

// standingRow is a player's rating columns of the users table, with their
// name.
type standingRow struct {
	ID          string
	Username    string
	UsernameKey string
	Rating      int
	Wins        int
	Losses      int
	Streak      int
}

func (standingRow) TableName() string { return "users" }

func (row standingRow) standing() Standing {
	return Standing{UserID: row.ID, Username: row.Username, Rank: RankOf(row.Rating), Wins: row.Wins, Losses: row.Losses, Streak: row.Streak}
}

// rankedOnly keeps the ranked players.
func rankedOnly(db *gorm.DB) *gorm.DB {
	return db.Where("wins + losses > 0")
}

// Service keeps the ratings. It is safe for concurrent use.
type Service struct {
	db       *gorm.DB
	settings *settings.Service
}

var _ games.Rater = (*Service)(nil)

// NewService returns the ratings, moved at the speed the settings hold.
func NewService(db *gorm.DB, siteSettings *settings.Service) *Service {
	return &Service{db: db, settings: siteSettings}
}

// Rate applies a won game that counted to both players: the winner gains
// what the loser loses, and each gets a win or a loss on their record and
// streak. It runs in the transaction saving the game, which holds both
// players' rows until it ends, so no other change comes between reading
// the ratings and writing them.
func (s *Service) Rate(tx *gorm.DB, game games.Record) (host, guest games.RatingChange, err error) {
	if !game.Rated() {
		return games.RatingChange{}, games.RatingChange{}, fmt.Errorf("rating game %s: it does not count", game.ID)
	}

	var rows []standingRow
	err = tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id IN ?", []string{game.Host.ID, game.Guest.ID}).Find(&rows).Error
	if err != nil {
		return games.RatingChange{}, games.RatingChange{}, fmt.Errorf("reading the players' ratings: %w", err)
	}
	byID := make(map[string]standingRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	hostRow, hostFound := byID[game.Host.ID]
	guestRow, guestFound := byID[game.Guest.ID]
	if !hostFound || !guestFound {
		return games.RatingChange{}, games.RatingChange{}, fmt.Errorf("rating game %s: %w", game.ID, ErrNotFound)
	}

	winner, loser := &hostRow, &guestRow
	if game.WinnerID == game.Guest.ID {
		winner, loser = loser, winner
	}
	points := Change(winner.Rating, loser.Rating, s.settings.Number(Speed))
	winner.Rating += points
	winner.Wins++
	winner.Streak = NextStreak(winner.Streak, true)
	loser.Rating -= points
	loser.Losses++
	loser.Streak = NextStreak(loser.Streak, false)

	for _, row := range []*standingRow{winner, loser} {
		err := tx.Model(&standingRow{ID: row.ID}).Updates(map[string]any{
			"rating": row.Rating, "wins": row.Wins, "losses": row.Losses, "streak": row.Streak,
		}).Error
		if err != nil {
			return games.RatingChange{}, games.RatingChange{}, fmt.Errorf("saving %s's rating: %w", row.Username, err)
		}
	}

	change := func(row standingRow) games.RatingChange {
		if row.ID == winner.ID {
			return games.RatingChange{Change: points, Rating: row.Rating}
		}
		return games.RatingChange{Change: -points, Rating: row.Rating}
	}
	return change(hostRow), change(guestRow), nil
}

// Standing returns where a player stands. It returns ErrNotFound for a
// player who does not exist.
func (s *Service) Standing(ctx context.Context, userID string) (Standing, error) {
	var row standingRow
	err := s.db.WithContext(ctx).Where("id = ?", userID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Standing{}, ErrNotFound
	}
	if err != nil {
		return Standing{}, fmt.Errorf("reading a player's rating: %w", err)
	}
	return row.standing(), nil
}

// Standings returns where each of the players stands, by id. Ids of players
// who do not exist are left out.
func (s *Service) Standings(ctx context.Context, userIDs []string) (map[string]Standing, error) {
	standings := make(map[string]Standing, len(userIDs))
	if len(userIDs) == 0 {
		return standings, nil
	}

	var rows []standingRow
	if err := s.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("reading players' ratings: %w", err)
	}
	for _, row := range rows {
		standings[row.ID] = row.standing()
	}
	return standings, nil
}

// Place returns a ranked player's place in the rankings, and 0 for a player
// not ranked.
func (s *Service) Place(ctx context.Context, standing Standing) (int, error) {
	if !standing.Ranked() {
		return 0, nil
	}
	places, err := s.places(ctx, []int{standing.Rank.Rating})
	if err != nil {
		return 0, err
	}
	return places[standing.Rank.Rating], nil
}

// Rankings returns one page of the ranked players, the best first, and those
// with the same rating by name. It returns a *validation.Error for a query
// breaking the limits.
func (s *Service) Rankings(ctx context.Context, query RankingsQuery) (Rankings, error) {
	query.Search = strings.TrimSpace(query.Search)
	if utf8.RuneCountInString(query.Search) > users.UsernameMaxLength {
		return Rankings{}, validation.New("search", fmt.Sprintf("must be at most %d characters", users.UsernameMaxLength))
	}
	if err := query.Normalize(DefaultPageSize, MaxPageSize); err != nil {
		return Rankings{}, err
	}

	matching := rankedOnly(s.db.WithContext(ctx).Model(&standingRow{}))
	if query.Search != "" {
		matching = matching.Where(store.Contains("username_key", users.Key(query.Search)))
	}

	var total int64
	if err := matching.Count(&total).Error; err != nil {
		return Rankings{}, fmt.Errorf("counting ranked players: %w", err)
	}

	var rows []standingRow
	err := matching.Order("rating DESC").Order("username_key").
		Limit(query.PageSize).Offset(query.Offset()).Find(&rows).Error
	if err != nil {
		return Rankings{}, fmt.Errorf("reading the rankings: %w", err)
	}

	ratings := make([]int, 0, len(rows))
	for _, row := range rows {
		ratings = append(ratings, row.Rating)
	}
	places, err := s.places(ctx, ratings)
	if err != nil {
		return Rankings{}, err
	}

	placed := make([]Placed, 0, len(rows))
	for _, row := range rows {
		placed = append(placed, Placed{Standing: row.standing(), Place: places[row.Rating]})
	}
	return Rankings{Players: placed, Total: int(total), Page: query.Page, PageSize: query.PageSize}, nil
}

// places returns the place in the full rankings of each rating: one more
// than the ranked players rated above it.
func (s *Service) places(ctx context.Context, ratings []int) (map[int]int, error) {
	places := make(map[int]int, len(ratings))
	for _, rating := range slices.Compact(slices.Sorted(slices.Values(ratings))) {
		var above int64
		err := rankedOnly(s.db.WithContext(ctx).Model(&standingRow{})).Where("rating > ?", rating).Count(&above).Error
		if err != nil {
			return nil, fmt.Errorf("placing a rating in the rankings: %w", err)
		}
		places[rating] = int(above) + 1
	}
	return places, nil
}
