package rematch_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby/lobbytest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/rematch"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

const waitTimeout = gamestest.Timeout

// ending is a rematch the players were told will not be played.
type ending struct {
	rematch rematch.Rematch
	reason  rematch.Reason
	by      string
}

// heard keeps what rematches tell their listener.
type heard struct {
	mu      sync.Mutex
	changed []rematch.Rematch
	ended   []ending
	started []games.Game
}

func (h *heard) RematchChanged(r rematch.Rematch) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.changed = append(h.changed, r)
}

func (h *heard) RematchEnded(r rematch.Rematch, reason rematch.Reason, byUserID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ended = append(h.ended, ending{rematch: r, reason: reason, by: byUserID})
}

func (h *heard) GameStarted(game games.Game) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.started = append(h.started, game)
}

func (h *heard) latest() rematch.Rematch {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.changed[len(h.changed)-1]
}

func (h *heard) endings() []ending {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.ended)
}

func (h *heard) startedGames() []games.Game {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.started)
}

// fixture is the lobby's world with rematches, whose sweeps run when run is
// set.
type fixture struct {
	*lobbytest.World
	rematches *rematch.Service
	heard     *heard
}

func newFixture(t *testing.T, db *gorm.DB, run bool) fixture {
	t.Helper()

	w := lobbytest.NewWorld(t, db, rematch.Settings()...)
	listener := &heard{}
	service := rematch.NewService(rematch.Dependencies{
		Lobby: w.Lobby, Games: w.Games, Directory: w.Directory, Settings: w.Settings, Listener: listener,
		Logger: storetest.Logger(), Now: w.Clock.Now, SweepEvery: 10 * time.Millisecond,
	})

	if run {
		ctx, stop := context.WithCancel(context.Background())
		running := make(chan struct{})
		go func() {
			defer close(running)
			service.Run(ctx)
		}()
		t.Cleanup(func() {
			stop()
			<-running
		})
	}

	return fixture{World: w, rematches: service, heard: listener}
}

// playAndSurrender plays a game Shobu hosts against Kokujo, each with their
// legal deck, until Shobu surrenders, and returns it.
func (f fixture) playAndSurrender(t *testing.T) games.Game {
	t.Helper()
	ctx := context.Background()

	open, err := f.Lobby.Create(ctx, f.Shobu, lobby.CreateInput{
		Name: "Friendly", DeckID: f.Legal[f.Shobu.ID].ID, FormatID: f.Open.ID, Visible: true, Spectators: false,
	})
	require.NoError(t, err)
	game, err := f.Lobby.Join(ctx, f.Kokujo, open.ID, f.Legal[f.Kokujo.ID].ID)
	require.NoError(t, err)

	shobu, kokujo := gamestest.Play(t, f.Games, game)
	shobu.Say(map[string]any{"header": "resign"})
	kokujo.WaitFor(t, "duel_finished")
	shobu.WaitFor(t, "duel_finished")

	// Played out: the game has closed.
	require.Eventually(t, func() bool { _, running := f.Games.Game(game.ID); return !running }, waitTimeout, 5*time.Millisecond)
	return game
}

// anotherDeck gives a player a second legal deck.
func (f fixture) anotherDeck(t *testing.T, player users.User, name string) decks.Deck {
	t.Helper()

	deck, err := f.Decks.Create(context.Background(), player.ID, decks.Input{Name: name, Cards: gamestest.LegalDeck(f.CardIDs)})
	require.NoError(t, err)
	return deck
}

func TestAGameThatEndsOffersARematch(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		game := f.playAndSurrender(t)

		offered, err := f.rematches.Get(f.Kokujo.ID, game.ID)
		require.NoError(t, err)
		assert.Equal(t, game.ID, offered.GameID)
		assert.Equal(t, "Friendly", offered.Name)
		assert.Equal(t, f.Open.ID, offered.FormatID)
		assert.Equal(t, "Open", offered.FormatName)
		assert.True(t, offered.Visible)
		assert.False(t, offered.Spectators)
		assert.False(t, offered.Matchmaking)
		assert.Equal(t, rematch.Seat{
			UserID: f.Shobu.ID, Username: "Shobu",
			Deck: rematch.Deck{ID: f.Legal[f.Shobu.ID].ID, Name: "Shobu's deck", Civilizations: []string{"Fire"}},
		}, offered.Host, "each player starts with the deck they played")
		assert.Equal(t, f.Legal[f.Kokujo.ID].ID, offered.Guest.Deck.ID)
		assert.Equal(t, f.Shobu.ID, offered.CallerID, "the loser calls the next coin toss")
		assert.Equal(t, f.Clock.Now().Add(2*time.Minute), offered.ExpiresAt)

		assert.Equal(t, offered, f.heard.latest(), "both players are shown it")
		assert.Equal(t, lobby.Rematching, f.Doing(f.Shobu), "deciding, not free, once the game has closed")
		assert.Equal(t, lobby.Rematching, f.Doing(f.Kokujo))

		_, err = f.Lobby.Create(context.Background(), f.Shobu, lobby.CreateInput{DeckID: f.Legal[f.Shobu.ID].ID, FormatID: f.Open.ID})
		assert.True(t, lobby.IsBusy(err), "a player deciding on a rematch does nothing else")

		_, err = f.rematches.Get(f.Mimi.ID, game.ID)
		assert.ErrorIs(t, err, rematch.ErrNotFound, "only its players see a rematch")
		_, _, err = f.rematches.Ready(context.Background(), f.Mimi, game.ID)
		assert.ErrorIs(t, err, rematch.ErrNotFound)
	})
}

func TestBothAgreeingStartsTheNewGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		ctx := context.Background()
		last := f.playAndSurrender(t)

		started, offered, err := f.rematches.Ready(ctx, f.Kokujo, last.ID)
		require.NoError(t, err)
		assert.Nil(t, started, "waiting for the other player")
		assert.True(t, offered.Guest.Ready)
		assert.False(t, offered.Host.Ready)
		assert.Equal(t, offered, f.heard.latest())

		started, _, err = f.rematches.Ready(ctx, f.Shobu, last.ID)
		require.NoError(t, err)
		require.NotNil(t, started)
		assert.NotEqual(t, last.ID, started.ID, "a new game")
		assert.Equal(t, "Friendly", started.Name)
		assert.Equal(t, f.Open.ID, started.FormatID)
		assert.True(t, started.Visible)
		assert.False(t, started.Spectators)
		assert.Equal(t, f.Shobu.ID, started.Host.ID, "the players keep their seats")
		assert.Equal(t, []games.Game{*started}, f.heard.startedGames(), "both are told to go to it")

		for _, player := range []users.User{f.Shobu, f.Kokujo} {
			activity, ok := f.Lobby.Activities().Of(player.ID)
			require.True(t, ok)
			assert.Equal(t, lobby.Activity{Kind: lobby.Playing, GameID: started.ID}, activity)
		}
		_, err = f.rematches.Get(f.Shobu.ID, last.ID)
		assert.ErrorIs(t, err, rematch.ErrNotFound, "decided")

		// The loser of the last game calls the toss.
		shobu, _ := gamestest.Connect(f.Games, f.Shobu, started.ID)
		kokujo, _ := gamestest.Connect(f.Games, f.Kokujo, started.ID)
		shobu.WaitFor(t, "choose_toss")
		kokujo.WaitFor(t, "toss_being_chosen")
	})
}

func TestAgreeingAtOnceStartsOneGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		last := f.playAndSurrender(t)

		var wg sync.WaitGroup
		for _, player := range []users.User{f.Shobu, f.Kokujo} {
			wg.Go(func() {
				_, _, err := f.rematches.Ready(context.Background(), player, last.ID)
				assert.NoError(t, err)
			})
		}
		wg.Wait()

		assert.Len(t, f.heard.startedGames(), 1)
		assert.Len(t, f.Games.Games(), 1)
	})
}

func TestSwitchingDecks(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		ctx := context.Background()
		last := f.playAndSurrender(t)
		otherDeck := f.anotherDeck(t, f.Shobu, "Water Wall")

		_, _, err := f.rematches.Ready(ctx, f.Kokujo, last.ID)
		require.NoError(t, err)

		switched, err := f.rematches.ChooseDeck(ctx, f.Shobu, last.ID, otherDeck.ID)
		require.NoError(t, err)
		assert.Equal(t, rematch.Deck{ID: otherDeck.ID, Name: "Water Wall", Civilizations: []string{"Fire"}}, switched.Host.Deck)
		assert.False(t, switched.Guest.Ready, "Kokujo agreed to the deck he saw, so must agree again")
		assert.Equal(t, switched, f.heard.latest())

		_, _, err = f.rematches.Ready(ctx, f.Shobu, last.ID)
		require.NoError(t, err)
		again, err := f.rematches.ChooseDeck(ctx, f.Kokujo, last.ID, f.Legal[f.Kokujo.ID].ID)
		require.NoError(t, err)
		assert.True(t, again.Host.Ready, "choosing the same deck again changes nothing")

		_, err = f.rematches.ChooseDeck(ctx, f.Shobu, last.ID, f.Short[f.Shobu.ID].ID)
		var illegal *games.IllegalDeckError
		require.ErrorAs(t, err, &illegal, "a deck must keep to the format")
		_, err = f.rematches.ChooseDeck(ctx, f.Shobu, last.ID, f.Legal[f.Kokujo.ID].ID)
		var invalid *validation.Error
		require.ErrorAs(t, err, &invalid, "and be the player's own")
		assert.Equal(t, "deckId", invalid.Field)

		kept, err := f.rematches.Get(f.Shobu.ID, last.ID)
		require.NoError(t, err)
		assert.Equal(t, otherDeck.ID, kept.Host.Deck.ID, "a refused deck changes nothing")
		assert.True(t, kept.Host.Ready)

		started, _, err := f.rematches.Ready(ctx, f.Kokujo, last.ID)
		require.NoError(t, err)
		require.NotNil(t, started)
		assert.Equal(t, "Water Wall", started.Host.DeckName, "the new game is played with the decks chosen")
	})
}

func TestADeckThatStoppedKeepingToTheFormat(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		ctx := context.Background()
		last := f.playAndSurrender(t)

		_, _, err := f.rematches.Ready(ctx, f.Kokujo, last.ID)
		require.NoError(t, err)
		kokujoDeck := f.Legal[f.Kokujo.ID]
		_, err = f.Decks.Update(ctx, f.Kokujo.ID, kokujoDeck.ID, decks.Input{Name: kokujoDeck.Name, Cards: kokujoDeck.Cards[:3]})
		require.NoError(t, err)

		_, _, err = f.rematches.Ready(ctx, f.Shobu, last.ID)
		require.ErrorIs(t, err, rematch.ErrOpponentDeckIllegal)
		assert.Empty(t, f.heard.startedGames())

		open, err := f.rematches.Get(f.Shobu.ID, last.ID)
		require.NoError(t, err, "still to be decided")
		assert.False(t, open.Host.Ready)
		assert.False(t, open.Guest.Ready, "nobody has agreed to decks that can be played")

		_, _, err = f.rematches.Ready(ctx, f.Kokujo, last.ID)
		var illegal *games.IllegalDeckError
		require.ErrorAs(t, err, &illegal, "Kokujo is told his own deck cannot be played")

		_, err = f.Decks.Update(ctx, f.Shobu.ID, f.Legal[f.Shobu.ID].ID, decks.Input{Name: "Shobu's deck", Cards: kokujoDeck.Cards[:3]})
		require.NoError(t, err)
		_, _, err = f.rematches.Ready(ctx, f.Shobu, last.ID)
		require.ErrorAs(t, err, &illegal, "a player's own deck is checked as they agree")
	})
}

func TestNewGamesPaused(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		last := f.playAndSurrender(t)
		_, err := f.Settings.Set(context.Background(), lobby.NewGamesOpen.Key, "false")
		require.NoError(t, err)

		_, _, err = f.rematches.Ready(context.Background(), f.Shobu, last.ID)
		assert.ErrorIs(t, err, lobby.ErrPaused)
	})
}

func TestDecliningEndsItForBoth(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		last := f.playAndSurrender(t)
		_, _, err := f.rematches.Ready(context.Background(), f.Shobu, last.ID)
		require.NoError(t, err)

		require.NoError(t, f.rematches.Decline(f.Kokujo.ID, last.ID))

		endings := f.heard.endings()
		require.Len(t, endings, 1)
		assert.Equal(t, rematch.Declined, endings[0].reason)
		assert.Equal(t, f.Kokujo.ID, endings[0].by)
		assert.Equal(t, last.ID, endings[0].rematch.GameID)
		assert.Empty(t, f.Doing(f.Shobu), "both are free again")
		assert.Empty(t, f.Doing(f.Kokujo))

		assert.ErrorIs(t, f.rematches.Decline(f.Kokujo.ID, last.ID), rematch.ErrNotFound)
		_, _, err = f.rematches.Ready(context.Background(), f.Shobu, last.ID)
		assert.ErrorIs(t, err, rematch.ErrNotFound)
	})
}

func TestOnlyItsPlayersDecline(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		last := f.playAndSurrender(t)

		assert.ErrorIs(t, f.rematches.Decline(f.Mimi.ID, last.ID), rematch.ErrNotFound)
		_, err := f.rematches.Get(f.Shobu.ID, last.ID)
		assert.NoError(t, err, "still to be decided")
	})
}

func TestLeavingTheSiteEndsIt(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, false)
		last := f.playAndSurrender(t)

		f.Directory.Set(f.Shobu, false)
		f.rematches.PresenceChanged(f.Shobu.ID)

		endings := f.heard.endings()
		require.Len(t, endings, 1)
		assert.Equal(t, rematch.Left, endings[0].reason)
		assert.Equal(t, f.Shobu.ID, endings[0].by)
		assert.Empty(t, f.Doing(f.Kokujo))
		_, err := f.rematches.Get(f.Kokujo.ID, last.ID)
		assert.ErrorIs(t, err, rematch.ErrNotFound)
	})
}

func TestTheSweepEndsItForAPlayerWhoLeft(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		f.playAndSurrender(t)

		// Gone without the presence change arriving.
		f.Directory.Set(f.Kokujo, false)

		require.Eventually(t, func() bool { return len(f.heard.endings()) == 1 }, waitTimeout, 5*time.Millisecond)
		assert.Equal(t, rematch.Left, f.heard.endings()[0].reason)
		assert.Equal(t, f.Kokujo.ID, f.heard.endings()[0].by)
	})
}

func TestItIsTooLateToAgreeOnceTheOfferRunsOut(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, false)
		last := f.playAndSurrender(t)

		f.Clock.Advance(2 * time.Minute)
		_, _, err := f.rematches.Ready(context.Background(), f.Shobu, last.ID)
		assert.ErrorIs(t, err, rematch.ErrNotFound, "too late to agree, even before the sweep")
	})
}

func TestTheOfferRunsOut(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		f.playAndSurrender(t)

		f.Clock.Advance(2 * time.Minute)

		require.Eventually(t, func() bool { return len(f.heard.endings()) == 1 }, waitTimeout, 5*time.Millisecond)
		assert.Equal(t, rematch.Expired, f.heard.endings()[0].reason)
		assert.Empty(t, f.heard.endings()[0].by)
		assert.Empty(t, f.Doing(f.Shobu))
	})
}

func TestTheOfferTimeIsASetting(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, false)
		_, err := f.Settings.Set(context.Background(), rematch.OfferTime.Key, "30")
		require.NoError(t, err)

		last := f.playAndSurrender(t)

		offered, err := f.rematches.Get(f.Shobu.ID, last.ID)
		require.NoError(t, err)
		assert.Equal(t, f.Clock.Now().Add(30*time.Second), offered.ExpiresAt)
	})
}

func TestNoRematchWithAPlayerWhoHasLeft(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, false)
		ctx := context.Background()

		open, err := f.Lobby.Create(ctx, f.Shobu, lobby.CreateInput{DeckID: f.Legal[f.Shobu.ID].ID, FormatID: f.Open.ID, Visible: true})
		require.NoError(t, err)
		game, err := f.Lobby.Join(ctx, f.Kokujo, open.ID, f.Legal[f.Kokujo.ID].ID)
		require.NoError(t, err)
		shobu, kokujo := gamestest.Play(t, f.Games, game)

		f.Directory.Set(f.Kokujo, false)
		shobu.Say(map[string]any{"header": "resign"})
		kokujo.WaitFor(t, "duel_finished")
		// The game closing frees one player, then the other; neither is
		// kept deciding on a rematch.
		require.Eventually(t, func() bool { return f.Doing(f.Shobu) == "" && f.Doing(f.Kokujo) == "" }, waitTimeout, 5*time.Millisecond, "both freed as the game closes")

		_, err = f.rematches.Get(f.Shobu.ID, game.ID)
		assert.ErrorIs(t, err, rematch.ErrNotFound)
	})
}

func TestNoRematchOfAStoppedGame(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, false)
		ctx := context.Background()

		open, err := f.Lobby.Create(ctx, f.Shobu, lobby.CreateInput{DeckID: f.Legal[f.Shobu.ID].ID, FormatID: f.Open.ID, Visible: true})
		require.NoError(t, err)
		game, err := f.Lobby.Join(ctx, f.Kokujo, open.ID, f.Legal[f.Kokujo.ID].ID)
		require.NoError(t, err)
		_, kokujo := gamestest.Play(t, f.Games, game)

		closeCtx, cancel := context.WithTimeout(ctx, waitTimeout)
		defer cancel()
		require.NoError(t, f.Games.Close(closeCtx))
		finished := kokujo.WaitFor(t, "duel_finished")
		assert.Equal(t, "aborted", finished["ending"])

		_, err = f.rematches.Get(f.Shobu.ID, game.ID)
		assert.ErrorIs(t, err, rematch.ErrNotFound)
		assert.Empty(t, f.Doing(f.Shobu))
	})
}

func TestAQuickMatchRematchIsAQuickMatch(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		ctx := context.Background()

		game, err := f.Games.Start(ctx, games.StartInput{
			Name: "Quick match", Format: f.Open, Visible: true, Spectators: true, Matchmaking: true,
			Host:  games.Entrant{User: f.Shobu, Deck: f.Legal[f.Shobu.ID]},
			Guest: games.Entrant{User: f.Kokujo, Deck: f.Legal[f.Kokujo.ID]},
		})
		require.NoError(t, err)
		for _, player := range []users.User{f.Shobu, f.Kokujo} {
			require.NoError(t, f.Lobby.Activities().Claim(player.ID, lobby.Activity{Kind: lobby.Playing, GameID: game.ID}))
		}
		shobu, kokujo := gamestest.Play(t, f.Games, game)
		kokujo.Say(map[string]any{"header": "resign"})
		shobu.WaitFor(t, "duel_finished")

		_, _, err = f.rematches.Ready(ctx, f.Shobu, game.ID)
		require.NoError(t, err)
		started, _, err := f.rematches.Ready(ctx, f.Kokujo, game.ID)
		require.NoError(t, err)
		require.NotNil(t, started)
		assert.True(t, started.Matchmaking, "a rematch keeps the game's options")
		assert.True(t, started.Spectators)
		assert.Equal(t, "Quick match", started.Name)
	})
}

func TestADeckDeletedSinceItWasChosen(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db, true)
		ctx := context.Background()
		last := f.playAndSurrender(t)

		_, _, err := f.rematches.Ready(ctx, f.Kokujo, last.ID)
		require.NoError(t, err)
		require.NoError(t, f.Decks.Delete(ctx, f.Kokujo.ID, f.Legal[f.Kokujo.ID].ID))

		_, _, err = f.rematches.Ready(ctx, f.Shobu, last.ID)
		require.ErrorIs(t, err, rematch.ErrOpponentDeckIllegal, "a deleted deck cannot be played")

		_, _, err = f.rematches.Ready(ctx, f.Kokujo, last.ID)
		var illegal *games.IllegalDeckError
		require.ErrorAs(t, err, &illegal, "Kokujo is told his deck can no longer be played")
		assert.Empty(t, illegal.Problems)
		assert.Empty(t, f.heard.startedGames())
	})
}
