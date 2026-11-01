package computer

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

const liveTestTimeout = 5 * time.Second

// seatedLive is a live game with nothing but this test at the engine's end.
type seatedLive struct {
	desk *Desk
	live *Live
	// ahead moves the desk's clock on from the real one.
	ahead *atomic.Int64
	done  chan struct{}
}

func newSeatedLive(t *testing.T) seatedLive {
	t.Helper()

	ahead := &atomic.Int64{}
	desk := NewDesk(NewLibrary(nil), func() time.Time { return time.Now().Add(time.Duration(ahead.Load())) })
	desk.abandonCheck = 5 * time.Millisecond
	live, ok := desk.Seat(slog.New(slog.DiscardHandler)).(*Live)
	require.True(t, ok)

	done := make(chan struct{})
	go func() {
		defer close(done)
		live.Play()
	}()
	t.Cleanup(func() {
		_ = live.conn.Close()
		<-done
	})

	s := seatedLive{desk: desk, live: live, ahead: ahead, done: done}
	assert.Equal(t, "join_match", s.nextMove(t)["header"])
	return s
}

// nextMove reads the next move the seat makes, as the engine would.
func (s seatedLive) nextMove(t *testing.T) map[string]any {
	t.Helper()

	type read struct {
		data []byte
		err  error
	}
	got := make(chan read, 1)
	go func() {
		_, data, err := s.live.conn.ReadMessage()
		got <- read{data, err}
	}()

	select {
	case r := <-got:
		require.NoError(t, r.err)
		var move map[string]any
		require.NoError(t, json.Unmarshal(r.data, &move))
		return move
	case <-time.After(liveTestTimeout):
		t.Fatal("no move came")
		return nil
	}
}

// tell has the engine tell the seat something.
func (s seatedLive) tell(t *testing.T, message any) {
	t.Helper()
	require.NoError(t, s.live.conn.WriteJSON(message))
}

func myTurn() server.MatchStateMessage {
	return server.MatchStateMessage{Header: "state_update", State: server.MatchState{
		MyTurn: true,
		Turn:   3,
		Me:     server.PlayerState{Username: "Claude", Hand: []server.CardState{{CardID: "in-hand", Name: "A Card"}}},
	}}
}

func TestLiveWaitsForTheGameToSettle(t *testing.T) {
	s := newSeatedLive(t)
	s.tell(t, myTurn())

	shown := s.live.Wait(context.Background(), liveTestTimeout)
	assert.Equal(t, StatusYourTurn, shown.Status)
	assert.Equal(t, 3, shown.Turn)
	require.NotNil(t, shown.You)
	require.Len(t, shown.You.Hand, 1)
}

func TestLiveTellsWhenTheEngineIgnoresAMove(t *testing.T) {
	s := newSeatedLive(t)
	s.tell(t, myTurn())
	require.Equal(t, StatusYourTurn, s.live.Wait(context.Background(), liveTestTimeout).Status)

	require.NoError(t, moveAsync(s, MovePlay, "in-hand"))
	assert.Equal(t, "add_to_playzone", s.nextMove(t)["header"])

	shown := s.live.Wait(context.Background(), liveTestTimeout)
	assert.Equal(t, StatusYourTurn, shown.Status, "still its turn")
	assert.Contains(t, shown.Refused, "The game did not take that move, and nothing changed.")
	assert.Empty(t, s.live.Situation().Refused, "told once")
}

func TestLivePassesOnTheEnginesRefusal(t *testing.T) {
	s := newSeatedLive(t)
	s.tell(t, myTurn())
	require.Equal(t, StatusYourTurn, s.live.Wait(context.Background(), liveTestTimeout).Status)

	require.NoError(t, moveAsync(s, MovePlay, "in-hand"))
	s.nextMove(t)
	s.tell(t, server.WarningMessage{Header: "warn", Message: "You cannot play that."})

	assert.Contains(t, s.live.Wait(context.Background(), liveTestTimeout).Refused, "You cannot play that.")
}

func TestLiveChecksAMoveBeforeTheEngineDoes(t *testing.T) {
	s := newSeatedLive(t)

	assert.ErrorIs(t, s.live.Move(MoveEndTurn, ""), ErrNotNow, "not its turn")
	assert.ErrorIs(t, s.live.Answer(nil, 0, true), ErrNotNow, "nothing asked")
	assert.ErrorIs(t, s.live.CallCoin(true), ErrNotNow)
	assert.ErrorIs(t, s.live.ChooseOrder(true), ErrNotNow)

	s.tell(t, myTurn())
	require.Equal(t, StatusYourTurn, s.live.Wait(context.Background(), liveTestTimeout).Status)
	assert.ErrorIs(t, s.live.Move(MovePlay, "not-in-hand"), ErrNotNow)
	assert.ErrorIs(t, s.live.Move(MoveAttackPlayer, "in-hand"), ErrNotNow, "attacks come from the battle zone")
	assert.ErrorIs(t, s.live.Move("dance", "in-hand"), ErrNotNow)
}

func TestLiveAnswersAQuestionOnlyWithCardsOffered(t *testing.T) {
	s := newSeatedLive(t)
	s.tell(t, server.ActionMessage{
		Prompt: server.Prompt{ID: 9}, Header: "action", ActionType: "card", Text: "Choose one",
		Cards:             []server.CardState{{CardID: "offered", Name: "A Card"}, {CardID: "shown", Name: "B Card"}},
		UnselectableCards: []server.CardState{{CardID: "shown", Name: "B Card"}},
		MinSelections:     1, MaxSelections: 1,
	})

	shown := s.live.Wait(context.Background(), liveTestTimeout)
	require.Equal(t, StatusQuestion, shown.Status)
	require.NotNil(t, shown.Question)
	assert.Equal(t, "Choose one", shown.Question.Text)
	require.Len(t, shown.Question.Cards, 1, "only what can be chosen")

	assert.ErrorIs(t, s.live.Answer([]string{"shown"}, 0, false), ErrNotNow)
	assert.ErrorIs(t, s.live.Answer([]string{"elsewhere"}, 0, false), ErrNotNow)

	require.NoError(t, answerAsync(s, []string{"offered"}))
	reply := s.nextMove(t)
	assert.Equal(t, "action", reply["header"])
	assert.EqualValues(t, 9, reply["prompt"])
	assert.Equal(t, []any{"offered"}, reply["cards"])
}

func TestLiveGivesUpForAPlayerWhoHasGone(t *testing.T) {
	s := newSeatedLive(t)
	s.desk.Touch()

	s.ahead.Store(int64(DeskAbandonAfter - time.Minute))
	select {
	case <-time.After(50 * time.Millisecond):
	case move := <-s.live.conn.moves:
		t.Fatalf("gave up too early: %s", move)
	}

	s.ahead.Store(int64(DeskAbandonAfter + time.Minute))
	assert.Equal(t, "resign", s.nextMove(t)["header"])
}

func TestLiveGameEndingFreesTheDesk(t *testing.T) {
	s := newSeatedLive(t)
	assert.False(t, s.desk.Free())

	s.tell(t, myTurn())
	s.tell(t, server.DuelFinishedMessage{Header: "duel_finished", Ending: "play", Winner: &server.DuelFinishedPlayer{Username: "Claude"}, Turns: 7})

	shown := s.live.Wait(context.Background(), liveTestTimeout)
	assert.Equal(t, StatusFinished, shown.Status)
	require.NotNil(t, shown.Result)
	assert.True(t, shown.Result.Won)
	assert.Equal(t, 7, shown.Result.Turns)
	assert.True(t, s.desk.Free())

	_, ok := s.desk.Current()
	assert.False(t, ok, "told once")
}

func TestLiveGameStoppedByTheServerIsTold(t *testing.T) {
	s := newSeatedLive(t)
	require.NoError(t, s.live.conn.Close())
	<-s.done

	current, ok := s.desk.Current()
	require.True(t, ok)
	shown := current.Situation()
	assert.Equal(t, StatusFinished, shown.Status)
	require.NotNil(t, shown.Result)
	assert.Equal(t, "stopped", shown.Result.Ending)
	assert.False(t, shown.Result.Won)
}

func TestLiveHearsTheResultSentJustBeforeTheGameCloses(t *testing.T) {
	// Which of the two it notices first is down to chance, so it is tried often.
	for range 50 {
		desk := NewDesk(NewLibrary(nil), time.Now)
		live, ok := desk.Seat(slog.New(slog.DiscardHandler)).(*Live)
		require.True(t, ok)

		require.NoError(t, live.conn.WriteJSON(server.DuelFinishedMessage{Header: "duel_finished", Ending: "play", Winner: &server.DuelFinishedPlayer{Username: "Shobu"}}))
		require.NoError(t, live.conn.Close())
		live.Play()

		shown := live.Situation()
		require.NotNil(t, shown.Result)
		require.Equal(t, "play", shown.Result.Ending)
		require.Equal(t, "Shobu", shown.Result.Winner)
	}
}

func TestDeskCountsSomeoneAsThereWhileInTouch(t *testing.T) {
	ahead := &atomic.Int64{}
	desk := NewDesk(NewLibrary(nil), func() time.Time { return time.Now().Add(time.Duration(ahead.Load())) })
	assert.False(t, desk.Online(), "never in touch")

	desk.Touch()
	assert.True(t, desk.Online())

	ahead.Store(int64(DeskOnlineWindow + time.Second))
	assert.False(t, desk.Online())
}

func TestDeskWaitForGameGivesUpInTime(t *testing.T) {
	desk := NewDesk(NewLibrary(nil), time.Now)

	_, ok := desk.WaitForGame(context.Background(), 10*time.Millisecond)
	assert.False(t, ok)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, ok = desk.WaitForGame(ctx, liveTestTimeout)
	assert.False(t, ok)
}

// moveAsync makes a move, which only returns once the engine reads it.
func moveAsync(s seatedLive, kind, cardID string) error {
	// The move is checked before it is sent, so a refusal comes back at once.
	go func() { _ = s.live.Move(kind, cardID) }()
	return nil
}

func answerAsync(s seatedLive, cards []string) error {
	go func() { _ = s.live.Answer(cards, 0, false) }()
	return nil
}
