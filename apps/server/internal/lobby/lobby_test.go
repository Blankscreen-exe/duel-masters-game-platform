package lobby_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/clocktest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby/lobbytest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// fixture is the test world, as the tests name it.
type fixture struct {
	lobby     *lobby.Service
	games     *games.Service
	settings  *settings.Service
	directory *lobbytest.Directory
	heard     *lobbytest.Heard
	clock     *clocktest.Clock
	open      formats.Format
	classic   formats.Format
	shobu     users.User
	kokujo    users.User
	mimi      users.User
	// decks are each player's legal deck, by user id; short their unfinished
	// one.
	decks map[string]decks.Deck
	short map[string]decks.Deck
	world *lobbytest.World
}

func newFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()

	w := lobbytest.NewWorld(t, db)
	return fixture{
		lobby: w.Lobby, games: w.Games, settings: w.Settings, directory: w.Directory, heard: w.Heard, clock: w.Clock,
		open: w.Open, classic: w.Classic, shobu: w.Shobu, kokujo: w.Kokujo, mimi: w.Mimi,
		decks: w.Legal, short: w.Short, world: w,
	}
}

// host opens a listed game for Shobu with his legal deck.
func (f fixture) host(t *testing.T, input lobby.CreateInput) lobby.OpenGame {
	t.Helper()

	if input.DeckID == "" {
		input.DeckID = f.decks[f.shobu.ID].ID
	}
	if input.FormatID == "" {
		input.FormatID = f.open.ID
	}
	open, err := f.lobby.Create(context.Background(), f.shobu, input)
	require.NoError(t, err)
	return open
}

func activityOf(t *testing.T, f fixture, user users.User) lobby.Kind {
	t.Helper()
	return f.world.Doing(user)
}

func TestHostingAGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		open := f.host(t, lobby.CreateInput{Name: "  Friday night  ", Visible: true, Spectators: true})

		assert.NotEmpty(t, open.ID)
		assert.Equal(t, "Friday night", open.Name, "trimmed")
		assert.Equal(t, lobby.Person{ID: f.shobu.ID, Username: "Shobu"}, open.Host)
		assert.Equal(t, f.open.ID, open.FormatID)
		assert.Equal(t, "Open", open.FormatName)
		assert.True(t, open.Visible)
		assert.True(t, open.Spectators)
		assert.Equal(t, lobby.Hosting, activityOf(t, f, f.shobu))

		snapshot := f.heard.Latest()
		assert.Equal(t, []lobby.OpenGame{open}, snapshot.OpenGames, "everyone sees the listed game")
		assert.Equal(t, open, f.lobby.Snapshot().OpenGames[0])

		var shobu lobby.OnlinePlayer
		for _, player := range snapshot.Online {
			if player.ID == f.shobu.ID {
				shobu = player
			}
		}
		require.NotNil(t, shobu.Activity)
		assert.Equal(t, lobby.Hosting, shobu.Activity.Kind, "the online list says what he is doing")
	})
}

func TestAnInviteOnlyGameIsNotListed(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		open := f.host(t, lobby.CreateInput{Visible: false})

		assert.Empty(t, f.lobby.Snapshot().OpenGames)
		found, err := f.lobby.OpenGame(open.ID)
		require.NoError(t, err)
		assert.Equal(t, open, found, "the invite link leads to it")
	})
}

func TestAnUnnamedGameIsGivenAName(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)

		open := f.host(t, lobby.CreateInput{Name: "   "})

		words := strings.Fields(open.Name)
		assert.Len(t, words, 2, "such as Tidal Showdown, got %q", open.Name)
	})
}

func TestHostingChecksTheRules(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		legal := f.decks[f.shobu.ID].ID

		for name, test := range map[string]struct {
			input lobby.CreateInput
			field string
		}{
			"a long name":       {lobby.CreateInput{Name: strings.Repeat("a", lobby.NameMaxLength+1), DeckID: legal, FormatID: f.open.ID}, "name"},
			"two lines":         {lobby.CreateInput{Name: "one\ntwo", DeckID: legal, FormatID: f.open.ID}, "name"},
			"an unknown format": {lobby.CreateInput{DeckID: legal, FormatID: "nope"}, "formatId"},
			"someone's deck":    {lobby.CreateInput{DeckID: f.decks[f.kokujo.ID].ID, FormatID: f.open.ID}, "deckId"},
		} {
			_, err := f.lobby.Create(ctx, f.shobu, test.input)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%s: got %v", name, err)
			assert.Equal(t, test.field, invalid.Field, name)
		}

		_, err := f.lobby.Create(ctx, f.shobu, lobby.CreateInput{DeckID: f.short[f.shobu.ID].ID, FormatID: f.open.ID})
		var illegal *games.IllegalDeckError
		require.True(t, errors.As(err, &illegal), "got %v", err)
		assert.Equal(t, decks.ProblemSize, illegal.Problems[0].Kind)

		assert.Empty(t, f.lobby.Snapshot().OpenGames, "nothing was opened")
		assert.Empty(t, activityOf(t, f, f.shobu))
	})
}

func TestAPlayerDoesOneThingAtATime(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		first := f.host(t, lobby.CreateInput{Visible: true})

		_, err := f.lobby.Create(context.Background(), f.shobu, lobby.CreateInput{DeckID: f.decks[f.shobu.ID].ID, FormatID: f.open.ID})
		var busy *lobby.BusyError
		require.True(t, errors.As(err, &busy), "got %v", err)
		assert.Equal(t, lobby.Hosting, busy.Current.Kind)
		assert.Equal(t, first.ID, busy.Current.GameID)
		assert.Len(t, f.lobby.Snapshot().OpenGames, 1)
	})
}

func TestJoiningStartsTheGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		open := f.host(t, lobby.CreateInput{Name: "Friendly", Visible: true, Spectators: true})

		game, err := f.lobby.Join(ctx, f.kokujo, open.ID, f.decks[f.kokujo.ID].ID)
		require.NoError(t, err)

		assert.Equal(t, "Friendly", game.Name)
		assert.Equal(t, f.shobu.ID, game.Host.ID)
		assert.Equal(t, f.kokujo.ID, game.Guest.ID)
		assert.Equal(t, "Kokujo's deck", game.Guest.DeckName)
		assert.True(t, game.Watchable())
		assert.Equal(t, []games.Game{game}, f.heard.Started(), "both players are told to go to it")

		assert.Equal(t, lobby.Playing, activityOf(t, f, f.shobu))
		assert.Equal(t, lobby.Playing, activityOf(t, f, f.kokujo))

		snapshot := f.lobby.Snapshot()
		assert.Empty(t, snapshot.OpenGames, "no longer open")
		assert.Equal(t, []games.Game{game}, snapshot.Running, "listed as running")

		_, err = f.lobby.Join(ctx, f.mimi, open.ID, f.decks[f.mimi.ID].ID)
		assert.ErrorIs(t, err, lobby.ErrNotFound, "someone else got there first")

		// The game plays on the engine and, when it ends, frees its players.
		host, guest := gamestest.Play(t, f.games, game)
		guest.Say(map[string]any{"header": "resign"})
		host.WaitFor(t, "duel_finished")
		require.Eventually(t, func() bool { return activityOf(t, f, f.shobu) == "" && activityOf(t, f, f.kokujo) == "" },
			gamestest.Timeout, 5*time.Millisecond, "both players are free again")
		require.Eventually(t, func() bool { return len(f.heard.Latest().Running) == 0 }, gamestest.Timeout, 5*time.Millisecond)
	})
}

func TestJoiningIsRefusedWhenItCannotWork(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		open := f.host(t, lobby.CreateInput{Visible: true})

		_, err := f.lobby.Join(ctx, f.shobu, open.ID, f.decks[f.shobu.ID].ID)
		assert.ErrorIs(t, err, lobby.ErrOwnGame)

		_, err = f.lobby.Join(ctx, f.kokujo, open.ID, f.short[f.kokujo.ID].ID)
		var illegal *games.IllegalDeckError
		require.True(t, errors.As(err, &illegal), "got %v", err)
		assert.Equal(t, "Kokujo", illegal.Username)

		_, err = f.lobby.Join(ctx, f.kokujo, open.ID, f.decks[f.shobu.ID].ID)
		invalid, ok := validation.As(err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, "deckId", invalid.Field, "only their own decks")

		_, err = f.lobby.Join(ctx, f.kokujo, "no-such-game", f.decks[f.kokujo.ID].ID)
		assert.ErrorIs(t, err, lobby.ErrNotFound)

		assert.Equal(t, []lobby.OpenGame{open}, f.lobby.Snapshot().OpenGames, "the game waits for the next player")
		assert.Empty(t, activityOf(t, f, f.kokujo))
		assert.Empty(t, f.heard.Started())
	})
}

func TestAGameWhoseHostLeftCannotBeJoined(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		open := f.host(t, lobby.CreateInput{Visible: true})
		f.directory.Set(f.shobu, false)

		_, err := f.lobby.Join(context.Background(), f.kokujo, open.ID, f.decks[f.kokujo.ID].ID)

		assert.ErrorIs(t, err, lobby.ErrNotFound)
		assert.Empty(t, f.lobby.Snapshot().OpenGames)
		assert.Empty(t, activityOf(t, f, f.shobu))
	})
}

func TestTheHostLeavingClosesTheirGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		f.host(t, lobby.CreateInput{Visible: true})

		f.directory.Set(f.shobu, false)
		f.lobby.PresenceChanged(f.shobu.ID)

		assert.Empty(t, f.heard.Latest().OpenGames)
		assert.Empty(t, activityOf(t, f, f.shobu))
		for _, player := range f.heard.Latest().Online {
			assert.NotEqual(t, f.shobu.ID, player.ID, "no longer online")
		}
	})
}

func TestCancellingAGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		open := f.host(t, lobby.CreateInput{Visible: true})

		assert.ErrorIs(t, f.lobby.Cancel(ctx, f.kokujo, open.ID), lobby.ErrNotHost)
		require.NoError(t, f.lobby.Cancel(ctx, f.shobu, open.ID))
		assert.ErrorIs(t, f.lobby.Cancel(ctx, f.shobu, open.ID), lobby.ErrNotFound)

		assert.Empty(t, f.heard.Latest().OpenGames)
		assert.Empty(t, activityOf(t, f, f.shobu))
	})
}

func TestAGameNobodyJoinsCloses(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		open := f.host(t, lobby.CreateInput{Visible: true})

		ctx, stop := context.WithCancel(context.Background())
		defer stop()
		go f.lobby.Run(ctx)

		f.clock.Advance(lobby.OpenGameLifetime - time.Second)
		assert.Never(t, func() bool { return len(f.lobby.Snapshot().OpenGames) == 0 }, 200*time.Millisecond, 10*time.Millisecond,
			"still waiting just before its time")

		f.clock.Advance(time.Second)
		require.Eventually(t, func() bool { return len(f.heard.Expired()) == 1 }, gamestest.Timeout, 5*time.Millisecond, "the host is told")
		assert.Equal(t, open, f.heard.Expired()[0])
		assert.Empty(t, f.lobby.Snapshot().OpenGames)
		assert.Empty(t, activityOf(t, f, f.shobu))
	})
}

func TestPausedNewGames(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		f := newFixture(t, db)
		open := f.host(t, lobby.CreateInput{Visible: true})
		_, err := f.settings.Set(ctx, lobby.NewGamesOpen.Key, "false")
		require.NoError(t, err)

		_, err = f.lobby.Create(ctx, f.kokujo, lobby.CreateInput{DeckID: f.decks[f.kokujo.ID].ID, FormatID: f.open.ID})
		assert.ErrorIs(t, err, lobby.ErrPaused)
		_, err = f.lobby.Join(ctx, f.kokujo, open.ID, f.decks[f.kokujo.ID].ID)
		assert.ErrorIs(t, err, lobby.ErrPaused)
		assert.True(t, f.lobby.Paused())
	})
}
