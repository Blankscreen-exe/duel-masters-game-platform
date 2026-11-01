package localplayer_test

import (
	"context"
	"encoding/json"
	"maps"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby/lobbytest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/localplayer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
)

// playWithin is how long a whole game played through the tools may take.
const playWithin = 3 * time.Minute

// mostMoves is how many tool calls a game may take before it counts as stuck.
const mostMoves = 5000

// world is a site with the live opponent, reached through the tools as Claude
// Code reaches them.
type world struct {
	*lobbytest.World
	challenges *opponents.Challenges
	opponents  *opponents.Service
	library    *computer.Library
	desk       *computer.Desk
	url        string
	session    *mcp.ClientSession
}

func newWorld(t *testing.T, db *gorm.DB) *world {
	t.Helper()
	ctx := context.Background()

	w := lobbytest.NewWorld(t, db, opponents.Settings()...)
	require.NoError(t, store.RunSeeds(db, opponents.Seeds(storetest.Logger())...))

	known, err := w.Catalogue.Cards(ctx, w.CardIDs)
	require.NoError(t, err)
	library := computer.NewLibrary(slices.Collect(maps.Values(known)))
	desk := computer.NewDesk(library, time.Now)
	service := opponents.NewService(db, w.Catalogue, w.Formats)

	site := httptest.NewServer(localplayer.NewHandler(localplayer.Dependencies{
		Desk:      desk,
		Opponents: service,
		Catalogue: w.Catalogue,
		Formats:   w.Formats,
		Settings:  w.Settings,
	}))
	t.Cleanup(site.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: site.URL + localplayer.Path}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return &world{
		World: w,
		challenges: opponents.NewChallenges(opponents.ChallengeDependencies{
			Opponents: service,
			Formats:   w.Formats,
			Lobby:     w.Lobby,
			Games:     w.Games,
			Library:   library,
			Settings:  w.Settings,
			Desk:      desk,
			Logger:    storetest.Logger(),
		}),
		opponents: service,
		library:   library,
		desk:      desk,
		url:       site.URL + localplayer.Path,
		session:   session,
	}
}

// join has Shobu join the game, as the duel board does, and waits until both
// players have, which the toss beginning shows.
func (w *world) join(t *testing.T, game games.Game) {
	t.Helper()
	shobu, _ := gamestest.Connect(w.Games, w.Shobu, game.ID)
	shobu.Say(map[string]any{"header": "join_match"})
	shobu.WaitForChat(t, "Shobu joined the game")
}

// closed waits until a game has closed. One given up before its coin toss is
// settled has no result to tell of.
func (w *world) closed(t *testing.T, game games.Game) {
	t.Helper()
	require.Eventually(t, func() bool { _, running := w.Games.Game(game.ID); return !running }, gamestest.Timeout, 5*time.Millisecond)
}

// switchOn turns on the setting.
func (w *world) switchOn(t *testing.T) {
	t.Helper()
	_, err := w.Settings.Set(context.Background(), opponents.LiveOpponent.Key, "true")
	require.NoError(t, err)
}

// call calls a tool, returning its text, and whether it reported an error.
func (w *world) call(t *testing.T, name string, arguments any) (string, bool) {
	t.Helper()
	result, err := w.session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	require.NoError(t, err)
	require.NotEmpty(t, result.Content)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	return text.Text, result.IsError
}

// callOK calls a tool that must work, reading its result as T.
func callOK[T any](t *testing.T, w *world, name string, arguments any) T {
	t.Helper()
	text, failed := w.call(t, name, arguments)
	require.False(t, failed, text)
	var result T
	require.NoError(t, json.Unmarshal([]byte(text), &result), text)
	return result
}

// saveDeck has the live opponent build its legal deck in the Open format.
func (w *world) saveDeck(t *testing.T) {
	t.Helper()
	entries := make([]map[string]any, 0, len(w.CardIDs))
	for _, entry := range gamestest.LegalDeck(w.CardIDs) {
		entries = append(entries, map[string]any{"card_id": entry.CardID, "count": entry.Count})
	}
	saved := callOK[map[string]any](t, w, "save_deck", map[string]any{"format_id": w.Open.ID, "cards": entries})
	require.Equal(t, true, saved["saved"], saved)
}

// offered reports whether players are offered the live opponent.
func (w *world) offered(t *testing.T) (string, bool) {
	t.Helper()
	offers, err := w.challenges.Offers(context.Background())
	require.NoError(t, err)
	for _, offer := range offers {
		if offer.IsLive() {
			return offer.ID, true
		}
	}
	return "", false
}

func TestTheLiveOpponentIsOfferedOnlyWhenItCanPlay(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newWorld(t, db)
		_, offered := w.offered(t)
		assert.False(t, offered, "at first it is off, away, and has no deck")

		w.switchOn(t)
		w.desk.Touch()
		_, offered = w.offered(t)
		assert.False(t, offered, "it has no deck")

		w.saveDeck(t)
		id, offered := w.offered(t)
		require.True(t, offered)

		_, err := w.Settings.Set(context.Background(), opponents.LiveOpponent.Key, "false")
		require.NoError(t, err)
		_, offered = w.offered(t)
		assert.False(t, offered, "switched off")
		_, err = w.challenges.Start(context.Background(), w.Shobu, id, w.Legal[w.Shobu.ID].ID)
		assert.ErrorIs(t, err, opponents.ErrUnavailable)

		w.switchOn(t)
		game, err := w.challenges.Start(context.Background(), w.Shobu, id, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)
		_, offered = w.offered(t)
		assert.False(t, offered, "it is in a game")
		_, err = w.challenges.Start(context.Background(), w.Kokujo, id, w.Legal[w.Kokujo.ID].ID)
		assert.ErrorIs(t, err, opponents.ErrUnavailable)

		// Turning it off leaves the game under way able to finish.
		_, err = w.Settings.Set(context.Background(), opponents.LiveOpponent.Key, "false")
		require.NoError(t, err)
		_, failed := w.call(t, "status", struct{}{})
		assert.False(t, failed)

		w.join(t, game)
		_, failed = w.call(t, "resign", struct{}{})
		assert.False(t, failed)
		w.closed(t, game)
	})
}

func TestTheToolsTellWhenSwitchedOff(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newWorld(t, db)

		for _, name := range []string{"status", "list_formats", "wait_for_turn"} {
			text, failed := w.call(t, name, struct{}{})
			assert.True(t, failed, name)
			assert.Contains(t, text, opponents.LiveOpponent.Key)
		}
		assert.False(t, w.desk.Online(), "being refused is not being there")
	})
}

func TestTheLiveOpponentsDeckMustBePlayable(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newWorld(t, db)
		w.switchOn(t)

		short := map[string]any{"format_id": w.Open.ID, "cards": []map[string]any{{"card_id": w.CardIDs[0], "count": 4}}}
		saved := callOK[map[string]any](t, w, "save_deck", short)
		assert.Equal(t, false, saved["saved"])
		assert.NotEmpty(t, saved["problems"])
		live, err := w.opponents.Live(context.Background())
		require.NoError(t, err)
		assert.Empty(t, live.Cards, "nothing changed")
		assert.False(t, live.Enabled)

		text, failed := w.call(t, "save_deck", map[string]any{"format_id": "nowhere", "cards": []map[string]any{}})
		assert.True(t, failed, text)

		w.saveDeck(t)
		status := callOK[map[string]any](t, w, "status", struct{}{})
		assert.Len(t, status["deck"], len(w.CardIDs))
		assert.Nil(t, status["deck_problems"])
	})
}

func TestAdminsDoNotManageTheLiveOpponent(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newWorld(t, db)
		ctx := context.Background()

		live, err := w.opponents.Live(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Claude", live.Name)

		all, err := w.opponents.All(ctx)
		require.NoError(t, err)
		assert.False(t, slices.ContainsFunc(all, opponents.Opponent.IsLive))
		_, err = w.opponents.ByID(ctx, live.ID)
		assert.ErrorIs(t, err, opponents.ErrNotFound)
		assert.NotContains(t, computer.StyleNames(), opponents.LiveStyle)
	})
}

func TestFindCardsKeepsToTheFormatAndTheSearch(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newWorld(t, db)
		w.switchOn(t)

		found := callOK[struct {
			Total int
			Cards []struct {
				ID   string
				Name string
				Text string
			}
		}](t, w, "find_cards", map[string]any{"format_id": w.Open.ID})
		require.NotZero(t, found.Total)
		require.NotEmpty(t, found.Cards)

		name := found.Cards[0].Name
		byName := callOK[struct{ Total int }](t, w, "find_cards", map[string]any{"words": name})
		assert.GreaterOrEqual(t, byName.Total, 1)
		nothing := callOK[struct{ Total int }](t, w, "find_cards", map[string]any{"words": "no card says this at all"})
		assert.Zero(t, nothing.Total)
	})
}

func TestMovesTheGameHasNoPlaceForAreRefused(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newWorld(t, db)
		w.switchOn(t)
		w.saveDeck(t)

		for _, attempt := range []struct {
			name      string
			arguments map[string]any
		}{
			{"make_move", map[string]any{"move": "end_turn"}},
			{"answer_question", map[string]any{"cancel": true}},
			{"coin_toss", map[string]any{"choice": "heads"}},
			{"say", map[string]any{"message": "hello"}},
			{"resign", map[string]any{}},
		} {
			text, failed := w.call(t, attempt.name, attempt.arguments)
			assert.True(t, failed, attempt.name)
			assert.Contains(t, text, "not in a game", attempt.name)
		}

		id, offered := w.offered(t)
		require.True(t, offered)
		game, err := w.challenges.Start(context.Background(), w.Shobu, id, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)
		w.join(t, game)

		text, failed := w.call(t, "coin_toss", map[string]any{"choice": "sideways"})
		assert.True(t, failed, text)
		text, failed = w.call(t, "make_move", map[string]any{"move": "cheat", "card": "x"})
		assert.True(t, failed, text)
		text, failed = w.call(t, "answer_question", map[string]any{"cards": []string{"x"}})
		assert.True(t, failed, text)

		callOK[map[string]any](t, w, "resign", struct{}{})
		w.closed(t, game)
	})
}

func TestOtherSitesPagesCannotReachTheTools(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newWorld(t, db)

		request, err := http.NewRequest(http.MethodPost, w.url, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		request.Header.Set("Origin", "https://evil.example")
		request.Header.Set("Sec-Fetch-Site", "cross-site")

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, http.StatusForbidden, response.StatusCode)
	})
}

// situation is a game as the tools tell it, as far as the stand-in needs.
type situation struct {
	Status string `json:"status"`
	Turn   int    `json:"turn"`
	You    *struct {
		Hand   []card `json:"hand"`
		Battle []card `json:"battle_zone"`
	} `json:"you"`
	Opponent *struct {
		HandCount int    `json:"hand_count"`
		Hand      []card `json:"hand"`
	} `json:"opponent"`
	ManaCharged   bool `json:"mana_charged"`
	CanChargeMana bool `json:"can_charge_mana"`
	Question      *struct {
		HowToAnswer string   `json:"how_to_answer"`
		Min         int      `json:"min_cards"`
		Max         int      `json:"max_cards"`
		Cancellable bool     `json:"cancellable"`
		Cards       []card   `json:"cards"`
		Choices     []string `json:"choices"`
	} `json:"question"`
	Refused []string `json:"refused"`
	Result  *struct {
		Won    bool   `json:"won"`
		Winner string `json:"winner"`
		Ending string `json:"ending"`
	} `json:"result"`
}

type card struct {
	ID        string `json:"id"`
	CanPlay   bool   `json:"can_play"`
	CanAttack bool   `json:"can_attack"`
}

// standIn plays in Claude's place through the tools, carelessly but by the
// rules: what is being tested is that a whole game can be played with them.
type standIn struct {
	w *world
	// tried are the moves made this turn, to try each once.
	turn  int
	tried map[string]bool
	rand  *rand.Rand
	// refusals counts refusals in a row of an answer to a question.
	refusals int
}

// next makes the next call the situation asks for.
func (s *standIn) next(t *testing.T, now situation) situation {
	t.Helper()

	switch now.Status {
	case computer.StatusNoGame, computer.StatusOpponentMove:
		return callOK[situation](t, s.w, "wait_for_turn", struct{}{})
	case computer.StatusCallCoin:
		return callOK[situation](t, s.w, "coin_toss", map[string]any{"choice": "heads"})
	case computer.StatusChooseOrder:
		return callOK[situation](t, s.w, "coin_toss", map[string]any{"choice": "first"})
	case computer.StatusQuestion:
		return callOK[situation](t, s.w, "answer_question", s.answer(t, now))
	case computer.StatusYourTurn:
		return callOK[situation](t, s.w, "make_move", s.move(now))
	}
	t.Fatalf("unknown status %q", now.Status)
	return now
}

func (s *standIn) answer(t *testing.T, now situation) map[string]any {
	t.Helper()

	q := now.Question
	if len(now.Refused) == 0 {
		s.refusals = 0
	} else {
		s.refusals++
		require.Less(t, s.refusals, 200, "the game keeps refusing every answer: %v", now.Refused)
	}

	switch {
	case strings.Contains(q.HowToAnswer, "close them"), strings.Contains(q.HowToAnswer, "no cards for yes"):
		return map[string]any{"cancel": strings.Contains(q.HowToAnswer, "close them")}
	case len(q.Choices) > 0:
		return map[string]any{"count": s.rand.IntN(len(q.Choices))}
	case strings.Contains(q.HowToAnswer, "a number from"):
		return map[string]any{"count": q.Min}
	}

	ids := make([]string, 0, len(q.Cards))
	for _, offered := range q.Cards {
		ids = append(ids, offered.ID)
	}
	if strings.Contains(q.HowToAnswer, "the order you want") {
		return map[string]any{"cards": ids}
	}
	if len(now.Refused) > 0 && q.Cancellable {
		return map[string]any{"cancel": true}
	}

	// A wrong choice, such as mana of the wrong civilization, is tried again
	// with others.
	s.rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	return map[string]any{"cards": ids[:min(max(q.Min, 1), q.Max, len(ids))]}
}

func (s *standIn) move(now situation) map[string]any {
	if s.turn != now.Turn {
		s.turn, s.tried = now.Turn, map[string]bool{}
	}
	once := func(move, id string) bool {
		key := move + " " + id
		if s.tried[key] {
			return false
		}
		s.tried[key] = true
		return true
	}

	if !now.ManaCharged && now.CanChargeMana && len(now.You.Hand) > 0 && once("charge_mana", "") {
		return map[string]any{"move": "charge_mana", "card": now.You.Hand[0].ID}
	}
	for _, inHand := range now.You.Hand {
		if inHand.CanPlay && once("play", inHand.ID) {
			return map[string]any{"move": "play", "card": inHand.ID}
		}
	}
	for _, creature := range now.You.Battle {
		if creature.CanAttack && once("attack_player", creature.ID) {
			return map[string]any{"move": "attack_player", "card": creature.ID}
		}
	}
	return map[string]any{"move": "end_turn"}
}

func TestAWholeGameCanBePlayedThroughTheTools(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		w := newWorld(t, db)
		ctx := context.Background()
		w.switchOn(t)

		// It builds its deck, and waits for a game.
		w.saveDeck(t)

		id, offered := w.offered(t)
		require.True(t, offered)
		game, err := w.challenges.Start(ctx, w.Shobu, id, w.Legal[w.Shobu.ID].ID)
		require.NoError(t, err)
		assert.True(t, game.Guest.Computer)
		assert.Equal(t, "Claude", game.Guest.Username)

		// The player's side is played by the computer, as a browser's would be.
		player := computer.New(computer.Options{Library: w.library, Style: computer.Balanced, Seed: 7})
		connected := make(chan error, 1)
		go func() { connected <- w.Games.Connect(player.Connection(), w.Shobu, game.ID) }()
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			player.Play()
		}()

		stand := &standIn{w: w, rand: rand.New(rand.NewPCG(1, 2))}
		deadline := time.Now().Add(playWithin)
		now := situation{Status: computer.StatusNoGame}
		for calls := 0; now.Status != computer.StatusFinished; calls++ {
			require.Less(t, calls, mostMoves, "the game did not end")
			require.True(t, time.Now().Before(deadline), "the game did not end in time")

			now = stand.next(t, now)
			if now.Opponent != nil {
				assert.Empty(t, now.Opponent.Hand, "the opponent's hand stays hidden")
			}
		}

		require.NotNil(t, now.Result)
		assert.Equal(t, "play", now.Result.Ending)
		assert.NotEmpty(t, now.Result.Winner)
		assert.Equal(t, now.Result.Winner == "Claude", now.Result.Won)

		select {
		case <-stopped:
		case <-time.After(gamestest.Timeout):
			t.Fatal("the player's side did not stop")
		}
		require.NoError(t, <-connected)

		var record games.Record
		require.Eventually(t, func() bool {
			record, err = w.Games.Recorded(ctx, game.ID)
			return err == nil
		}, gamestest.Timeout, 5*time.Millisecond)
		assert.False(t, record.Counted, "it never counts")
		assert.Equal(t, games.EndingPlay, record.Ending)

		// Told once, the desk is free for the next game.
		assert.True(t, w.desk.Free())
		_, stillThere := w.desk.Current()
		assert.False(t, stillThere)
	})
}
