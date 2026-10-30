package opponents_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby/lobbytest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents/opponentstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// challengeWorld is a lobby's world with a computer opponent to play.
type challengeWorld struct {
	*lobbytest.World
	challenges *opponents.Challenges
	grim       opponentstest.Added
}

func newChallengeWorld(t *testing.T, db *gorm.DB) challengeWorld {
	t.Helper()

	w := lobbytest.NewWorld(t, db, opponents.Settings()...)
	return challengeWorld{
		World:      w,
		challenges: opponentstest.NewChallenges(t, w),
		grim:       opponentstest.Add(t, db, "Grim", w.Open.ID, gamestest.LegalDeck(w.CardIDs)),
	}
}

// start starts a game between Shobu and the opponent.
func (w challengeWorld) start(t *testing.T) games.Game {
	t.Helper()

	game, err := w.challenges.Start(context.Background(), w.Shobu, w.grim.ID, w.Legal[w.Shobu.ID].ID)
	require.NoError(t, err)
	return game
}

// closed waits until the game has closed.
func (w challengeWorld) closed(t *testing.T, gameID string) {
	t.Helper()
	require.Eventually(t, func() bool { _, running := w.Games.Game(gameID); return !running }, gamestest.Timeout, 5*time.Millisecond)
}

// resign has Shobu join the game, settle the toss with the computer and give
// up, and returns what the game told him of its end.
func (w challengeWorld) resign(t *testing.T, game games.Game) map[string]any {
	t.Helper()

	shobu, _ := gamestest.Connect(w.Games, w.Shobu, game.ID)
	shobu.SettleToss(t)
	shobu.Say(map[string]any{"header": "resign"})
	return shobu.WaitFor(t, "duel_finished")
}

func TestAGameAgainstTheComputerIsPrivate(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newChallengeWorld(t, db)
		game := w.start(t)

		assert.True(t, game.AgainstComputer())
		assert.True(t, game.Guest.Computer)
		assert.Equal(t, "Grim", game.Guest.Username)
		assert.Equal(t, "Grim", game.Guest.DeckName)
		assert.False(t, game.Host.Computer)

		_, err := w.Games.RoleIn(w.Kokujo.ID, game.ID)
		assert.ErrorIs(t, err, games.ErrNotWatchable, "nobody else may watch it")
		assert.Empty(t, w.Lobby.Snapshot().Running, "it is not listed")
		assert.Equal(t, 1, w.Lobby.Snapshot().GamesRunning, "but it is counted")

		activity, ok := w.Lobby.Activities().Of(w.Shobu.ID)
		require.True(t, ok)
		assert.Equal(t, lobby.Activity{Kind: lobby.Playing, GameID: game.ID, Computer: true}, activity)
		_, busy := w.Lobby.Activities().Of(w.grim.Account.ID)
		assert.False(t, busy, "the computer may play many games at once")
	})
}

func TestAGameAgainstTheComputerIsKeptInHistoryAndNeverRated(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w := newChallengeWorld(t, db)
		// However short or long it is, it is the computer that stops it counting.
		_, err := w.Settings.Set(ctx, games.MinimumCountedDuration.Key, "0")
		require.NoError(t, err)

		game := w.start(t)
		finished := w.resign(t, game)
		assert.Equal(t, false, finished["counted"])
		assert.Equal(t, true, finished["practice"])
		w.closed(t, game.ID)

		var history games.HistoryPage
		require.Eventually(t, func() bool {
			history, err = w.Games.History(ctx, w.Shobu.ID, paging.Request{})
			return err == nil && history.Total == 1
		}, gamestest.Timeout, 5*time.Millisecond)

		saved := history.Games[0]
		assert.Equal(t, game.ID, saved.ID)
		assert.False(t, saved.Counted)
		assert.Equal(t, w.grim.Account.ID, saved.WinnerID)
		assert.False(t, saved.Host.Computer)
		assert.True(t, saved.Guest.Computer)
		assert.Equal(t, "Grim", saved.Guest.Username)
		assert.Nil(t, saved.Host.Rating)
		assert.Nil(t, saved.Guest.Rating)

		for _, player := range []string{w.Shobu.ID, w.grim.Account.ID} {
			standing, err := w.Ratings.Standing(ctx, player)
			require.NoError(t, err)
			assert.Zero(t, standing.Wins+standing.Losses, "nothing on anyone's record")
			assert.False(t, standing.Ranked())
		}
	})
}

func TestAPlayerPlaysAWholeGameAgainstTheComputer(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w := newChallengeWorld(t, db)
		_, err := w.Settings.Set(ctx, games.MessagesPerSecond.Key, "100")
		require.NoError(t, err)
		_, err = w.Settings.Set(ctx, games.MessageBurst.Key, "200")
		require.NoError(t, err)

		game := w.start(t)

		// The player's side is played by another computer player, over the
		// connection a browser's would come in by.
		known, err := w.Catalogue.Cards(ctx, w.CardIDs)
		require.NoError(t, err)
		library := computer.NewLibrary(slices.Collect(maps.Values(known)))
		player := computer.New(computer.Options{Library: library, Style: computer.Balanced, Seed: 7})
		connected := make(chan error, 1)
		go func() { connected <- w.Games.Connect(player.Connection(), w.Shobu, game.ID) }()
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			player.Play()
		}()

		select {
		case <-stopped:
		case <-time.After(2 * time.Minute):
			t.Fatal("the game did not end")
		}
		require.NoError(t, <-connected)

		var record games.Record
		require.Eventually(t, func() bool {
			record, err = w.Games.Recorded(ctx, game.ID)
			return err == nil
		}, gamestest.Timeout, 5*time.Millisecond)
		assert.Equal(t, games.EndingPlay, record.Ending, "won by the game's rules")
		assert.NotEmpty(t, record.WinnerID)
		assert.False(t, record.Counted)
	})
}

func TestAGameAgainstTheComputerIsRefusedFor(t *testing.T) {
	cases := map[string]func(t *testing.T, w challengeWorld){
		"an opponent not offered": func(t *testing.T, w challengeWorld) {
			ctx := context.Background()
			_, err := w.challenges.Start(ctx, w.Shobu, "nobody", w.Legal[w.Shobu.ID].ID)
			assert.ErrorIs(t, err, opponents.ErrNotFound)

			opponentstest.Retire(t, w.DB, w.grim.ID)
			_, err = w.challenges.Start(ctx, w.Shobu, w.grim.ID, w.Legal[w.Shobu.ID].ID)
			assert.ErrorIs(t, err, opponents.ErrNotFound)
			_, busy := w.Lobby.Activities().Of(w.Shobu.ID)
			assert.False(t, busy)
		},
		"a deck that is not legal": func(t *testing.T, w challengeWorld) {
			_, err := w.challenges.Start(context.Background(), w.Shobu, w.grim.ID, w.Short[w.Shobu.ID].ID)
			var illegal *games.IllegalDeckError
			assert.ErrorAs(t, err, &illegal)
		},
		"someone else's deck": func(t *testing.T, w challengeWorld) {
			_, err := w.challenges.Start(context.Background(), w.Shobu, w.grim.ID, w.Legal[w.Kokujo.ID].ID)
			_, invalid := validation.As(err)
			assert.True(t, invalid, "got %v", err)
		},
		"a player already doing something": func(t *testing.T, w challengeWorld) {
			first := w.start(t)
			_, err := w.challenges.Start(context.Background(), w.Shobu, w.grim.ID, w.Legal[w.Shobu.ID].ID)
			assert.True(t, lobby.IsBusy(err), "got %v", err)
			activity, _ := w.Lobby.Activities().Of(w.Shobu.ID)
			assert.Equal(t, first.ID, activity.GameID, "still playing the first")
		},
		"new games being paused": func(t *testing.T, w challengeWorld) {
			_, err := w.Settings.Set(context.Background(), lobby.NewGamesOpen.Key, "false")
			require.NoError(t, err)
			_, err = w.challenges.Start(context.Background(), w.Shobu, w.grim.ID, w.Legal[w.Shobu.ID].ID)
			assert.ErrorIs(t, err, lobby.ErrPaused)
		},
		"an opponent whose deck can no longer be played": func(t *testing.T, w challengeWorld) {
			require.NoError(t, w.DB.Table("computer_opponent_cards").
				Where("computer_opponent_id = ? AND card_id = ?", w.grim.ID, w.CardIDs[0]).Delete(nil).Error)
			_, err := w.challenges.Start(context.Background(), w.Shobu, w.grim.ID, w.Legal[w.Shobu.ID].ID)
			assert.ErrorIs(t, err, opponents.ErrUnavailable)
			_, busy := w.Lobby.Activities().Of(w.Shobu.ID)
			assert.False(t, busy)
		},
	}

	for name, check := range cases {
		t.Run(name, func(t *testing.T) {
			storetest.Each(t, func(t *testing.T, db *gorm.DB) {
				check(t, newChallengeWorld(t, db))
			})
		})
	}
}

func TestTheSettingLimitsGamesAgainstTheComputer(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w := newChallengeWorld(t, db)
		_, err := w.Settings.Set(ctx, games.MaxComputerGames.Key, "1")
		require.NoError(t, err)

		first := w.start(t)

		_, err = w.challenges.Start(ctx, w.Kokujo, w.grim.ID, w.Legal[w.Kokujo.ID].ID)
		require.ErrorIs(t, err, games.ErrComputerGamesFull)
		_, busy := w.Lobby.Activities().Of(w.Kokujo.ID)
		assert.False(t, busy, "a refused player is free to do something else")

		// Games between people are not limited.
		_, err = w.Games.Start(ctx, games.StartInput{
			Name:   "Friendly",
			Format: w.Open,
			Host:   games.Entrant{User: w.Kokujo, Deck: w.Legal[w.Kokujo.ID]},
			Guest:  games.Entrant{User: w.Mimi, Deck: w.Legal[w.Mimi.ID]},
		})
		require.NoError(t, err)

		// Its end makes room.
		w.resign(t, first)
		w.closed(t, first.ID)
		_, err = w.challenges.Start(ctx, w.Kokujo, w.grim.ID, w.Legal[w.Kokujo.ID].ID)
		assert.NoError(t, err)
	})
}

func TestAStoppedGameFreesThePlayerAndTheComputer(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w := newChallengeWorld(t, db)
		_, err := w.Settings.Set(ctx, games.MaxComputerGames.Key, "1")
		require.NoError(t, err)

		// Nobody joins it. Stopping it ends the computer's seat too: the
		// service's Close, when the test ends, waits for the computer.
		game := w.start(t)
		_, err = w.Games.Stop(game.ID)
		require.NoError(t, err)
		w.closed(t, game.ID)

		_, busy := w.Lobby.Activities().Of(w.Shobu.ID)
		assert.False(t, busy, "the player is free again")
		w.start(t)
	})
}

func TestComputerOpponentsAreOfferedWithTheirFormatAndCivilizations(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w := newChallengeWorld(t, db)
		retired := opponentstest.Add(t, db, "Retired", w.Open.ID, gamestest.LegalDeck(w.CardIDs))
		opponentstest.Retire(t, db, retired.ID)

		offers, err := w.challenges.Offers(ctx)
		require.NoError(t, err)
		require.Len(t, offers, 1)
		assert.Equal(t, "Grim", offers[0].Name)
		assert.Equal(t, w.Open.Name, offers[0].FormatName)
		assert.NotEmpty(t, offers[0].Civilizations)
	})
}
