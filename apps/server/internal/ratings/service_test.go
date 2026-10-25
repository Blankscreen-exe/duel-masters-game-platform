package ratings_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

type fixture struct {
	db       *gorm.DB
	ratings  *ratings.Service
	settings *settings.Service
	accounts *users.Service
}

func newFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()

	siteSettings, err := settings.NewService(context.Background(), db, storetest.Logger(), ratings.Settings()...)
	require.NoError(t, err)

	return fixture{
		db:       db,
		ratings:  ratings.NewService(db, siteSettings),
		settings: siteSettings,
		accounts: users.NewService(db, storetest.Logger()),
	}
}

func (f fixture) player(t *testing.T, name string) users.User {
	t.Helper()

	player, err := f.accounts.Create(context.Background(), name, "hash")
	require.NoError(t, err)
	return player
}

// play rates a counted game between host and guest that winner won, as the
// games service does when it saves it.
func (f fixture) play(t *testing.T, host, guest, winner users.User) (games.RatingChange, games.RatingChange) {
	t.Helper()

	hostChange, guestChange, err := f.rate(host, guest, winner)
	require.NoError(t, err)
	return hostChange, guestChange
}

// rate is play for any goroutine: it returns the error.
func (f fixture) rate(host, guest, winner users.User) (hostChange, guestChange games.RatingChange, err error) {
	game := games.Record{
		ID:       fmt.Sprintf("game-%s-%s", host.Username, guest.Username),
		Host:     games.RecordPlayer{ID: host.ID, Username: host.Username},
		Guest:    games.RecordPlayer{ID: guest.ID, Username: guest.Username},
		WinnerID: winner.ID,
		Counted:  true,
	}

	err = f.db.Transaction(func(tx *gorm.DB) error {
		var err error
		hostChange, guestChange, err = f.ratings.Rate(tx, game)
		return err
	})
	return hostChange, guestChange, err
}

func (f fixture) standing(t *testing.T, player users.User) ratings.Standing {
	t.Helper()

	standing, err := f.ratings.Standing(context.Background(), player.ID)
	require.NoError(t, err)
	return standing
}

func TestNewPlayersStartUnranked(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		shobu := f.player(t, "Shobu")

		standing := f.standing(t, shobu)
		assert.Equal(t, ratings.Standing{UserID: shobu.ID, Username: "Shobu", Rank: ratings.RankOf(ratings.StartingRating)}, standing)
		assert.False(t, standing.Ranked())

		place, err := f.ratings.Place(context.Background(), standing)
		require.NoError(t, err)
		assert.Zero(t, place, "not in the rankings")
	})
}

func TestRatingAGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		shobu, kokujo := f.player(t, "Shobu"), f.player(t, "Kokujo")

		host, guest := f.play(t, shobu, kokujo, kokujo)

		assert.Equal(t, games.RatingChange{Change: -16, Rating: 984}, host)
		assert.Equal(t, games.RatingChange{Change: 16, Rating: 1016}, guest)
		assert.Equal(t, ratings.Standing{UserID: shobu.ID, Username: "Shobu", Rank: ratings.RankOf(984), Losses: 1, Streak: -1}, f.standing(t, shobu))
		assert.Equal(t, ratings.Standing{UserID: kokujo.ID, Username: "Kokujo", Rank: ratings.RankOf(1016), Wins: 1, Streak: 1}, f.standing(t, kokujo))
	})
}

func TestRatingsBuildUpOverGames(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		shobu, kokujo := f.player(t, "Shobu"), f.player(t, "Kokujo")

		f.play(t, shobu, kokujo, shobu)
		_, second := f.play(t, kokujo, shobu, shobu)
		_, third := f.play(t, kokujo, shobu, kokujo)

		assert.Equal(t, 15, second.Change, "beating a weaker player is worth less")
		assert.Equal(t, -19, third.Change, "losing to one costs more")
		assert.Equal(t, ratings.Standing{UserID: shobu.ID, Username: "Shobu", Rank: ratings.RankOf(1012), Wins: 2, Losses: 1, Streak: -1}, f.standing(t, shobu))
		assert.Equal(t, ratings.Standing{UserID: kokujo.ID, Username: "Kokujo", Rank: ratings.RankOf(988), Wins: 1, Losses: 2, Streak: 1}, f.standing(t, kokujo))
	})
}

func TestTheSpeedSettingScalesChanges(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		_, err := f.settings.Set(context.Background(), ratings.Speed.Key, "10")
		require.NoError(t, err)
		shobu, kokujo := f.player(t, "Shobu"), f.player(t, "Kokujo")

		host, _ := f.play(t, shobu, kokujo, shobu)

		assert.Equal(t, 5, host.Change)
	})
}

func TestOnlyGamesThatCountAreRated(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		shobu, kokujo := f.player(t, "Shobu"), f.player(t, "Kokujo")

		for _, game := range []games.Record{
			{ID: "short", Host: games.RecordPlayer{ID: shobu.ID}, Guest: games.RecordPlayer{ID: kokujo.ID}, WinnerID: shobu.ID},
			{ID: "aborted", Host: games.RecordPlayer{ID: shobu.ID}, Guest: games.RecordPlayer{ID: kokujo.ID}, Counted: true},
		} {
			_, _, err := f.ratings.Rate(db, game)
			assert.Error(t, err, game.ID)
		}
		assert.False(t, f.standing(t, shobu).Ranked())
	})
}

func TestStandingsOfSeveralPlayers(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		shobu, kokujo := f.player(t, "Shobu"), f.player(t, "Kokujo")
		f.play(t, shobu, kokujo, shobu)

		standings, err := f.ratings.Standings(context.Background(), []string{shobu.ID, kokujo.ID, "no-such-player"})
		require.NoError(t, err)

		assert.Len(t, standings, 2)
		assert.Equal(t, 1016, standings[shobu.ID].Rank.Rating)
		assert.Equal(t, 984, standings[kokujo.ID].Rank.Rating)

		_, err = f.ratings.Standing(context.Background(), "no-such-player")
		assert.ErrorIs(t, err, ratings.ErrNotFound)
	})
}

// names lists the players of a page of the rankings with their places.
func names(rankings ratings.Rankings) []string {
	shown := make([]string, 0, len(rankings.Players))
	for _, player := range rankings.Players {
		shown = append(shown, fmt.Sprintf("%d %s", player.Place, player.Username))
	}
	return shown
}

func TestTheRankings(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		shobu, kokujo, mimi, hakuoh := f.player(t, "Shobu"), f.player(t, "Kokujo"), f.player(t, "Mimi"), f.player(t, "Hakuoh")
		f.player(t, "Newcomer")

		f.play(t, shobu, kokujo, shobu) // Shobu 1016, Kokujo 984
		f.play(t, mimi, hakuoh, mimi)   // Mimi 1016, Hakuoh 984

		all, err := f.ratings.Rankings(ctx, ratings.RankingsQuery{})
		require.NoError(t, err)
		assert.Equal(t, []string{"1 Mimi", "1 Shobu", "3 Hakuoh", "3 Kokujo"}, names(all),
			"the best first, sharing a place when level, by name; a player who never played is not ranked")
		assert.Equal(t, 4, all.Total)
		assert.Equal(t, 1, all.Page)
		assert.Equal(t, ratings.DefaultPageSize, all.PageSize)
		assert.Equal(t, 1, all.Players[0].Wins)

		second, err := f.ratings.Rankings(ctx, ratings.RankingsQuery{Request: paging.Request{Page: 2, PageSize: 3}})
		require.NoError(t, err)
		assert.Equal(t, []string{"3 Kokujo"}, names(second), "places count the whole rankings")

		found, err := f.ratings.Rankings(ctx, ratings.RankingsQuery{Search: " KOKU "})
		require.NoError(t, err)
		assert.Equal(t, []string{"3 Kokujo"}, names(found), "a search ignores case, and keeps each player's place")
		assert.Equal(t, 1, found.Total)

		place, err := f.ratings.Place(ctx, f.standing(t, kokujo))
		require.NoError(t, err)
		assert.Equal(t, 3, place)
	})
}

func TestTheRankingsSearchMatchesWildcardsLiterally(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		rock, rocky := f.player(t, "Rock_Fan"), f.player(t, "RockyFan")
		f.play(t, rock, rocky, rock)

		found, err := f.ratings.Rankings(context.Background(), ratings.RankingsQuery{Search: "k_f"})
		require.NoError(t, err)
		assert.Equal(t, []string{"1 Rock_Fan"}, names(found))
	})
}

func TestTheRankingsRefuseQueriesOutOfRange(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		for _, test := range []struct {
			query ratings.RankingsQuery
			field string
		}{
			{ratings.RankingsQuery{Search: "a name far too long to be one"}, "search"},
			{ratings.RankingsQuery{Request: paging.Request{Page: -1}}, "page"},
			{ratings.RankingsQuery{Request: paging.Request{PageSize: ratings.MaxPageSize + 1}}, "pageSize"},
		} {
			_, err := f.ratings.Rankings(context.Background(), test.query)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%+v: got %v", test.query, err)
			assert.Equal(t, test.field, invalid.Field)
		}
	})
}

func TestGamesRatedAtOnceBothCount(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		shobu, kokujo, mimi := f.player(t, "Shobu"), f.player(t, "Kokujo"), f.player(t, "Mimi")

		opponents := []users.User{kokujo, mimi}
		errs := make([]error, len(opponents))
		var wg sync.WaitGroup
		for i, opponent := range opponents {
			wg.Go(func() { _, _, errs[i] = f.rate(shobu, opponent, shobu) })
		}
		wg.Wait()
		require.NoError(t, errors.Join(errs...))

		standing := f.standing(t, shobu)
		assert.Equal(t, 2, standing.Wins, "neither game's result is lost")
		assert.Equal(t, 2, standing.Streak)
		assert.Equal(t, 1031, standing.Rank.Rating, "the second game was rated from the first's result")
	})
}

func TestBannedPlayersLeaveTheRankingsWhileTheBanLasts(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		shobu, kokujo, mimi := f.player(t, "Shobu"), f.player(t, "Kokujo"), f.player(t, "Mimi")
		f.play(t, shobu, kokujo, shobu) // Shobu 1016, Kokujo 984
		f.play(t, mimi, kokujo, mimi)   // Mimi 1015, Kokujo 969

		now := db.NowFunc()
		require.NoError(t, users.SetBanIn(db, shobu.ID, users.Ban{Penalty: users.Penalty{Since: now}, Reason: "Cheating"}))

		all, err := f.ratings.Rankings(ctx, ratings.RankingsQuery{})
		require.NoError(t, err)
		assert.Equal(t, []string{"1 Mimi", "2 Kokujo"}, names(all), "the others move up")
		assert.Equal(t, 2, all.Total)

		banned := f.standing(t, shobu)
		assert.True(t, banned.Banned)
		assert.False(t, banned.Ranked())
		place, err := f.ratings.Place(ctx, banned)
		require.NoError(t, err)
		assert.Zero(t, place)

		ended := now.Add(-time.Second)
		require.NoError(t, users.SetBanIn(db, shobu.ID, users.Ban{Penalty: users.Penalty{Since: now.Add(-time.Hour), Until: &ended}, Reason: "Over"}))
		all, err = f.ratings.Rankings(ctx, ratings.RankingsQuery{})
		require.NoError(t, err)
		assert.Equal(t, []string{"1 Shobu", "2 Mimi", "3 Kokujo"}, names(all), "back once the ban is over")
	})
}
