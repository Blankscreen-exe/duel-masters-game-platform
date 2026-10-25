package games_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// finishedGame is a game for the history, as a test seeds it.
type finishedGame struct {
	id           string
	host, guest  users.User
	winner       *users.User
	hostCivs     string
	guestCivs    string
	endedMinutes int
	ratingChange *int
}

// seed saves finished games straight into the history.
func (f fixture) seed(t *testing.T, seeded ...finishedGame) {
	t.Helper()

	start := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	for _, game := range seeded {
		row := map[string]any{
			"id": game.id, "name": "Seeded", "format_name": "Open",
			"host_id": game.host.ID, "host_deck_name": "Host deck", "host_deck": "[]", "host_civilizations": game.hostCivs,
			"guest_id": game.guest.ID, "guest_deck_name": "Guest deck", "guest_deck": "[]", "guest_civilizations": game.guestCivs,
			"ending": "play", "turns": 7,
			"started_at": start, "ended_at": start.Add(time.Duration(game.endedMinutes) * time.Minute),
			"counted": game.ratingChange != nil, "visible": true, "matchmaking": false,
		}
		if game.winner != nil {
			row["winner_id"] = game.winner.ID
		}
		if game.ratingChange != nil {
			row["host_rating_change"], row["host_rating"] = *game.ratingChange, 1000+*game.ratingChange
			row["guest_rating_change"], row["guest_rating"] = -*game.ratingChange, 1000-*game.ratingChange
		}
		require.NoError(t, f.db.Table("games").Create(row).Error)
	}
}

func ids(page games.HistoryPage) []string {
	shown := make([]string, 0, len(page.Games))
	for _, game := range page.Games {
		shown = append(shown, game.ID)
	}
	return shown
}

func TestAPlayersHistory(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		sixteen := 16
		f.seed(t,
			finishedGame{id: "first", host: f.shobu, guest: f.kokujo, winner: &f.shobu, hostCivs: "[]", guestCivs: "[]", endedMinutes: 1, ratingChange: &sixteen},
			finishedGame{id: "second", host: f.kokujo, guest: f.shobu, winner: &f.kokujo, hostCivs: "[]", guestCivs: "[]", endedMinutes: 2},
			finishedGame{id: "others", host: f.kokujo, guest: f.watcher, hostCivs: "[]", guestCivs: "[]", endedMinutes: 3},
			finishedGame{id: "third", host: f.shobu, guest: f.watcher, hostCivs: "[]", guestCivs: "[]", endedMinutes: 4},
		)

		first, err := f.games.History(ctx, f.shobu.ID, paging.Request{PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, []string{"third", "second"}, ids(first), "the latest first, only the player's")
		assert.Equal(t, 3, first.Total)
		assert.Equal(t, 1, first.Page)
		assert.Equal(t, 2, first.PageSize)

		second, err := f.games.History(ctx, f.shobu.ID, paging.Request{Page: 2, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, []string{"first"}, ids(second))

		game := second.Games[0]
		assert.Equal(t, "Shobu", game.Host.Username, "with the players' names")
		assert.Equal(t, "Kokujo", game.Guest.Username)
		assert.Equal(t, f.shobu.ID, game.WinnerID)
		assert.Equal(t, &games.RatingChange{Change: 16, Rating: 1016}, game.Host.Rating)
		assert.Equal(t, &games.RatingChange{Change: -16, Rating: 984}, game.Guest.Rating)
		assert.Equal(t, 7, game.Turns)
		assert.Equal(t, time.Minute, game.Duration())

		assert.Nil(t, first.Games[1].Host.Rating, "a game that did not count moved no rating")

		defaults, err := f.games.History(ctx, f.kokujo.ID, paging.Request{})
		require.NoError(t, err)
		assert.Equal(t, []string{"others", "second", "first"}, ids(defaults))
		assert.Equal(t, games.HistoryPageSize, defaults.PageSize)
	})
}

func TestHistoryRefusesPagesOutOfRange(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		_, err := f.games.History(context.Background(), f.shobu.ID, paging.Request{PageSize: games.MaxHistoryPageSize + 1})

		invalid, ok := validation.As(err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "pageSize", invalid.Field)
	})
}

func TestAPlayedGameIsInTheHistory(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		game := f.start(t, false, false)

		shobu, kokujo := gamestest.Play(t, f.games, game)
		shobu.Say(map[string]any{"header": "resign"})
		kokujo.WaitFor(t, "duel_finished")
		require.Eventually(t, func() bool { return len(f.heard.records()) == 1 }, waitTimeout, 5*time.Millisecond)

		recorded, err := f.games.Recorded(ctx, game.ID)
		require.NoError(t, err)
		heard := f.heard.records()[0]
		// Databases keep times to the microsecond.
		assert.WithinDuration(t, heard.StartedAt, recorded.StartedAt, time.Microsecond)
		assert.WithinDuration(t, heard.EndedAt, recorded.EndedAt, time.Microsecond)
		heard.StartedAt, heard.EndedAt = recorded.StartedAt, recorded.EndedAt
		assert.Equal(t, heard, recorded, "the history keeps the game as it finished")
		assert.Equal(t, "Shobu", recorded.Host.Username)
		assert.Equal(t, []string{"Fire"}, recorded.Guest.Civilizations)
		assert.Equal(t, f.deck("Water Wall").Cards, recorded.Guest.Deck)
		assert.False(t, recorded.Visible)

		_, err = f.games.Recorded(ctx, "no-such-game")
		assert.ErrorIs(t, err, games.ErrNotRecorded)
	})
}

func TestFavouriteCivilizations(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		f.seed(t,
			finishedGame{id: "g1", host: f.shobu, guest: f.kokujo, hostCivs: `["Fire","Nature"]`, guestCivs: `["Darkness"]`, endedMinutes: 1},
			finishedGame{id: "g2", host: f.kokujo, guest: f.shobu, hostCivs: `["Light"]`, guestCivs: `["Water","Fire"]`, endedMinutes: 2},
			finishedGame{id: "g3", host: f.shobu, guest: f.watcher, hostCivs: `[]`, guestCivs: `["Light"]`, endedMinutes: 3},
		)

		favourites, err := f.games.FavouriteCivilizations(context.Background(), f.shobu.ID)
		require.NoError(t, err)

		assert.Equal(t, []games.CivilizationCount{
			{Civilization: "Fire", Games: 2},
			{Civilization: "Water", Games: 1},
			{Civilization: "Nature", Games: 1},
		}, favourites, "the most played first, then in the game's order; only the player's own decks")

		none, err := f.games.FavouriteCivilizations(context.Background(), "no-games-yet")
		require.NoError(t, err)
		assert.Empty(t, none)
	})
}
