package games_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

// savedGame is a row of the games table as a test reads it.
type savedGame struct {
	ID           string
	Name         string
	FormatID     *string
	FormatName   string
	HostID       string
	HostDeckName string
	HostDeck     string
	// HostCivilizations is JSON.
	HostCivilizations string
	HostRatingChange  *int
	HostRating        *int
	GuestID           string
	GuestDeckName     string
	GuestRatingChange *int
	GuestRating       *int
	WinnerID          *string
	Ending            string
	Turns             int
	Counted           bool
	Visible           bool
	Matchmaking       bool
}

func saved(t *testing.T, db *gorm.DB) []savedGame {
	t.Helper()

	var rows []savedGame
	require.NoError(t, db.Table("games").Order("ended_at").Find(&rows).Error)
	return rows
}

func TestStartingAGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		game := f.start(t, true, false)

		assert.NotEmpty(t, game.ID)
		assert.Equal(t, "Friendly", game.Name)
		assert.Equal(t, f.open.ID, game.FormatID)
		assert.Equal(t, "Open", game.FormatName)
		assert.Equal(t, games.Player{ID: f.shobu.ID, Username: "Shobu", DeckID: "Fire Rush", DeckName: "Fire Rush", Civilizations: []string{"Fire"}}, game.Host)
		assert.Equal(t, games.Player{ID: f.kokujo.ID, Username: "Kokujo", DeckID: "Water Wall", DeckName: "Water Wall", Civilizations: []string{"Fire"}}, game.Guest)
		assert.True(t, game.Visible)
		assert.False(t, game.Spectators)

		running, ok := f.games.Game(game.ID)
		require.True(t, ok)
		assert.Equal(t, game, running)
		assert.Equal(t, []games.Game{game}, f.games.Games())

		playing, ok := f.games.PlayingIn(f.kokujo.ID)
		require.True(t, ok)
		assert.Equal(t, game.ID, playing.ID)
		_, ok = f.games.PlayingIn(f.watcher.ID)
		assert.False(t, ok)
	})
}

func TestADeckMustKeepToTheFormat(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		short := f.deck("Short")
		short.Cards = short.Cards[:9]

		_, err := f.games.Start(context.Background(), games.StartInput{
			Name:   "Friendly",
			Format: f.open,
			Host:   games.Entrant{User: f.shobu, Deck: f.deck("Fire Rush")},
			Guest:  games.Entrant{User: f.kokujo, Deck: short},
		})

		var illegal *games.IllegalDeckError
		require.True(t, errors.As(err, &illegal), "got %v", err)
		assert.Equal(t, "Kokujo", illegal.Username)
		assert.Equal(t, decks.ProblemSize, illegal.Problems[0].Kind)
		assert.Empty(t, f.games.Games(), "nothing started")
	})
}

func TestWhoMayConnect(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		watchable := f.start(t, true, true)
		unlisted := f.start(t, false, true)
		noSpectators := f.start(t, true, false)

		role, err := f.games.RoleIn(f.shobu.ID, noSpectators.ID)
		require.NoError(t, err)
		assert.Equal(t, games.RolePlayer, role)

		role, err = f.games.RoleIn(f.watcher.ID, watchable.ID)
		require.NoError(t, err)
		assert.Equal(t, games.RoleSpectator, role)

		for _, game := range []games.Game{unlisted, noSpectators} {
			_, err = f.games.RoleIn(f.watcher.ID, game.ID)
			assert.ErrorIs(t, err, games.ErrNotWatchable, "spectators watch only listed games that allow them")
		}

		_, err = f.games.RoleIn(f.shobu.ID, "no-such-game")
		assert.ErrorIs(t, err, games.ErrNotFound)

		refused, done := gamestest.Connect(f.games, f.watcher, noSpectators.ID)
		assert.ErrorIs(t, <-done, games.ErrNotWatchable)
		assert.True(t, refused.IsClosed(), "a refused connection is closed")
	})
}

func TestAGameIsPlayedAndSaved(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		game := f.start(t, true, true)

		shobu, kokujo := gamestest.Play(t, f.games, game)
		watcher, _ := gamestest.Connect(f.games, f.watcher, game.ID)
		watcher.WaitFor(t, "state_update")

		shobu.Say(map[string]any{"header": "resign"})
		finished := kokujo.WaitFor(t, "duel_finished")
		assert.Equal(t, "surrender", finished["ending"])
		watcher.WaitFor(t, "duel_finished")

		require.Eventually(t, func() bool { return len(f.heard.records()) == 1 }, waitTimeout, 5*time.Millisecond)
		record := f.heard.records()[0]
		assert.Equal(t, game.ID, record.ID)
		assert.Equal(t, f.kokujo.ID, record.WinnerID)
		assert.Equal(t, games.EndingSurrender, record.Ending)
		assert.Equal(t, "Fire Rush", record.Host.DeckName)
		assert.Equal(t, f.deck("Fire Rush").Cards, record.Host.Deck)
		assert.False(t, record.Counted, "a game of a few seconds does not count")
		assert.Nil(t, record.Host.Rating, "nor moves ratings")
		assert.Nil(t, record.Guest.Rating)

		rows := saved(t, db)
		require.Len(t, rows, 1)
		row := rows[0]
		assert.Equal(t, game.ID, row.ID)
		assert.Equal(t, "Friendly", row.Name)
		assert.Equal(t, f.open.ID, *row.FormatID)
		assert.Equal(t, "Open", row.FormatName)
		assert.Equal(t, f.shobu.ID, row.HostID)
		assert.Equal(t, "Fire Rush", row.HostDeckName)
		assert.Contains(t, row.HostDeck, `{"cardId":"A001","count":4}`)
		assert.JSONEq(t, `["Fire"]`, row.HostCivilizations)
		assert.Equal(t, f.kokujo.ID, row.GuestID)
		assert.Equal(t, f.kokujo.ID, *row.WinnerID)
		assert.Equal(t, "surrender", row.Ending)
		assert.False(t, row.Counted)
		assert.True(t, row.Visible)
		assert.Nil(t, row.HostRatingChange)
		assert.Nil(t, row.GuestRatingChange)
		assert.Equal(t, standing{Rating: 1000}, f.standing(t, f.shobu.ID), "ratings stay as they were")
		assert.Equal(t, standing{Rating: 1000}, f.standing(t, f.kokujo.ID))

		require.Eventually(t, func() bool { return len(f.heard.closedGames()) == 1 }, waitTimeout, 5*time.Millisecond)
		ended := f.heard.endedGames()
		require.Len(t, ended, 1)
		assert.Equal(t, game.ID, ended[0].Game.ID)
		assert.Equal(t, f.kokujo.ID, ended[0].WinnerID)
		assert.Equal(t, f.shobu.ID, ended[0].LoserID())
		assert.Equal(t, games.EndingSurrender, ended[0].Ending)
		assert.Equal(t, "Fire Rush", ended[0].Game.Host.DeckID)
		assert.Equal(t, []string{"ended", "closed"}, slices.DeleteFunc(f.heard.heardInOrder(), func(heard string) bool { return heard == "finished" }),
			"the end is heard before the game closes")
		assert.Empty(t, f.games.Games(), "no longer running")
		_, ok := f.games.PlayingIn(f.shobu.ID)
		assert.False(t, ok)
	})
}

func TestChattingInAGameTooFastIsRefused(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		_, err := f.settings.Set(ctx, chat.PostsAllowed.Key, "2")
		require.NoError(t, err)
		_, err = f.settings.Set(ctx, chat.PostWindow.Key, "60")
		require.NoError(t, err)

		shobu, kokujo := gamestest.Play(t, f.games, f.start(t, true, true))
		spam := []string{"gg 1", "gg 2", "gg 3"}
		for _, text := range spam {
			shobu.Say(map[string]any{"header": "chat", "message": text})
		}

		shobu.WaitForChat(t, "You are posting too fast. Wait 60 seconds.")
		shown := func() []string {
			return slices.DeleteFunc(kokujo.Chats(), func(text string) bool { return !slices.Contains(spam, text) })
		}
		require.Eventually(t, func() bool { return len(shown()) == 2 }, waitTimeout, 5*time.Millisecond,
			"the opponent sees the two messages allowed")
		assert.NotContains(t, kokujo.Chats(), "You are posting too fast. Wait 60 seconds.", "only the sender is told")

		kokujo.Say(map[string]any{"header": "chat", "message": "calm down"})
		shobu.WaitForChat(t, "calm down")
		assert.Len(t, shown(), 2, "the refused message never reached the opponent")
	})
}

func TestALongEnoughGameCounts(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		_, err := f.settings.Set(context.Background(), games.MinimumCountedDuration.Key, "0")
		require.NoError(t, err)
		game := f.start(t, true, false)

		shobu, kokujo := gamestest.Play(t, f.games, game)
		kokujo.Say(map[string]any{"header": "resign"})
		shobu.WaitFor(t, "duel_finished")

		require.Eventually(t, func() bool { return len(saved(t, db)) == 1 }, waitTimeout, 5*time.Millisecond)
		row := saved(t, db)[0]
		assert.True(t, row.Counted)
		assert.Equal(t, f.shobu.ID, *row.WinnerID)

		// Equally rated, so each rating moves by half the speed.
		assert.Equal(t, 16, *row.HostRatingChange)
		assert.Equal(t, 1016, *row.HostRating)
		assert.Equal(t, -16, *row.GuestRatingChange)
		assert.Equal(t, 984, *row.GuestRating)
		assert.Equal(t, standing{Rating: 1016, Wins: 1, Streak: 1}, f.standing(t, f.shobu.ID))
		assert.Equal(t, standing{Rating: 984, Losses: 1, Streak: -1}, f.standing(t, f.kokujo.ID))

		require.Len(t, f.heard.records(), 1)
		record := f.heard.records()[0]
		assert.Equal(t, &games.RatingChange{Change: 16, Rating: 1016}, record.Host.Rating)
		assert.Equal(t, &games.RatingChange{Change: -16, Rating: 984}, record.Guest.Rating)
	})
}

func TestAGameThatCannotBeRatedIsNotSaved(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		_, err := f.settings.Set(context.Background(), games.MinimumCountedDuration.Key, "0")
		require.NoError(t, err)
		f.rater.fail(errors.New("the database went away"))
		game := f.start(t, true, false)

		shobu, kokujo := gamestest.Play(t, f.games, game)
		kokujo.Say(map[string]any{"header": "resign"})
		shobu.WaitFor(t, "duel_finished")

		require.Eventually(t, func() bool { return f.rater.timesAsked() == 1 }, waitTimeout, 5*time.Millisecond)

		// Games are saved in order, so once the next one is, the first was
		// dealt with.
		f.rater.fail(nil)
		next := f.start(t, true, false)
		shobu, kokujo = gamestest.Play(t, f.games, next)
		kokujo.Say(map[string]any{"header": "resign"})
		shobu.WaitFor(t, "duel_finished")
		require.Eventually(t, func() bool { return len(saved(t, db)) == 1 }, waitTimeout, 5*time.Millisecond)

		assert.Equal(t, next.ID, saved(t, db)[0].ID, "a game is never saved without its ratings")
		require.Len(t, f.heard.records(), 1)
		assert.Equal(t, next.ID, f.heard.records()[0].ID)
		assert.Equal(t, standing{Rating: 1016, Wins: 1, Streak: 1}, f.standing(t, f.shobu.ID), "the first game moved nothing")
	})
}

func TestAGameThatNeverStartsIsNotSaved(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		_, err := f.settings.Set(context.Background(), games.DisconnectGrace.Key, "15")
		require.NoError(t, err)
		game := f.start(t, true, false)

		// Shobu joins and leaves before Kokujo arrives.
		shobu, _ := gamestest.Connect(f.games, f.shobu, game.ID)
		require.Eventually(t, func() bool {
			_, ok := f.games.Game(game.ID)
			return ok
		}, waitTimeout, 5*time.Millisecond)
		_ = shobu.Close()

		closeCtx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()
		require.NoError(t, f.games.Close(closeCtx))

		assert.Empty(t, saved(t, db))
		assert.Empty(t, f.heard.records())
	})
}

func TestStoppingTheServerSavesRunningGamesAsAborted(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		game := f.start(t, true, false)
		shobu, _ := gamestest.Play(t, f.games, game)

		closeCtx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()
		require.NoError(t, f.games.Close(closeCtx))

		finished := shobu.WaitFor(t, "duel_finished")
		assert.Equal(t, "aborted", finished["ending"])

		require.Eventually(t, func() bool { return len(saved(t, db)) == 1 }, waitTimeout, 5*time.Millisecond)
		row := saved(t, db)[0]
		assert.Equal(t, "aborted", row.Ending)
		assert.Nil(t, row.WinnerID)
		assert.False(t, row.Counted)

		_, err := f.games.Start(context.Background(), games.StartInput{
			Name:   "Too late",
			Format: f.open,
			Host:   games.Entrant{User: f.shobu, Deck: f.deck("Fire Rush")},
			Guest:  games.Entrant{User: f.kokujo, Deck: f.deck("Water Wall")},
		})
		assert.ErrorIs(t, err, games.ErrClosed, "no new games once stopping")
	})
}

func TestAnAdminCanStopAGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		game := f.start(t, true, false)
		shobu, kokujo := gamestest.Play(t, f.games, game)

		stopped, err := f.games.Stop(game.ID)
		require.NoError(t, err)
		assert.Equal(t, game.ID, stopped.ID)

		for _, player := range []*gamestest.Pipe{shobu, kokujo} {
			assert.Equal(t, "aborted", player.WaitFor(t, "duel_finished")["ending"])
			assert.Contains(t, player.WaitFor(t, "error")["message"], "An admin stopped this game")
		}

		require.Eventually(t, func() bool { return len(saved(t, db)) == 1 }, waitTimeout, 5*time.Millisecond)
		row := saved(t, db)[0]
		assert.Equal(t, "aborted", row.Ending)
		assert.Nil(t, row.WinnerID)
		assert.False(t, row.Counted, "a stopped game never counts")

		require.Eventually(t, func() bool {
			_, err := f.games.Stop(game.ID)
			return errors.Is(err, games.ErrNotFound)
		}, waitTimeout, 5*time.Millisecond, "a game no longer running cannot be stopped")
	})
}

func TestExpelPutsAPlayerOutOfTheGamesTheyPlayAndWatch(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		game := f.start(t, true, true)
		shobu, _ := gamestest.Play(t, f.games, game)
		watching, done := gamestest.Connect(f.games, f.watcher, game.ID)
		watching.WaitFor(t, "state_update")

		f.games.Expel(f.watcher.ID)
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(waitTimeout):
			t.Fatal("the spectator's connection should close")
		}
		assert.True(t, watching.IsClosed())
		_, running := f.games.Game(game.ID)
		assert.True(t, running, "the game goes on without its spectator")

		f.games.Expel(f.shobu.ID)
		assert.Equal(t, "aborted", shobu.WaitFor(t, "duel_finished")["ending"])
		require.Eventually(t, func() bool { return len(saved(t, db)) == 1 }, waitTimeout, 5*time.Millisecond)

		f.games.Expel(f.kokujo.ID) // in no game any more: nothing happens
	})
}
