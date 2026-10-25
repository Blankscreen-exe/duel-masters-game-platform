package tests

import (
	"encoding/json"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
)

// lifecycleTimeout bounds every wait for something a match does on its own
// goroutines.
const lifecycleTimeout = 5 * time.Second

const (
	hostID  = "host-1"
	guestID = "guest-2"
)

// recordingObserver keeps what matches report.
type recordingObserver struct {
	mu       sync.Mutex
	finished []match.Result
	closed   []string
}

func (o *recordingObserver) MatchFinished(result match.Result) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.finished = append(o.finished, result)
}

func (o *recordingObserver) MatchClosed(matchID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.closed = append(o.closed, matchID)
}

func (o *recordingObserver) results() []match.Result {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.finished)
}

func (o *recordingObserver) closedMatches() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.closed)
}

// lifecycleOptions are a match between Alice (the host) and Bob, whose toss
// Alice calls.
func lifecycleOptions(grace time.Duration) match.Options {
	return match.Options{
		Name:            "Friendly",
		Host:            match.Seat{ID: hostID, Username: "Alice", Deck: scenario.DefaultDeck()},
		Guest:           match.Seat{ID: guestID, Username: "Bob", Deck: scenario.DefaultDeck()},
		Format:          match.FormatDescriptor{ID: "open", Name: "Open"},
		TossCaller:      hostID,
		DisconnectGrace: grace,
	}
}

func newLifecycleMatch(t *testing.T, options match.Options) (*match.Match, *recordingObserver) {
	t.Helper()

	scenario.RegisterCards()
	observer := &recordingObserver{}
	m, err := match.NewSystem(match.SystemOptions{Observer: observer}).NewMatch(options)
	require.NoError(t, err)
	t.Cleanup(m.Dispose)

	return m, observer
}

// client is someone connected to a match.
type client struct {
	socket *server.Socket
	conn   *scenario.MockConnection
}

// connect opens a connection to the match and sends the join message, as the
// duel board does.
func connect(m *match.Match, uid, username string) client {
	conn := scenario.NewMockConnection()
	c := client{socket: server.NewSocket(conn, m, uid, username), conn: conn}
	c.send(m, map[string]any{"header": "join_match"})
	return c
}

func (c client) send(m *match.Match, message map[string]any) {
	data, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	m.Parse(c.socket, data)
}

// mark is how many messages the client has received so far.
func (c client) mark() int {
	return c.conn.JSONWriteCount()
}

// waitFor waits for a message with the header received since mark, and
// returns it.
func (c client) waitFor(t *testing.T, since int, header string) map[string]any {
	t.Helper()

	var found map[string]any
	require.Eventually(t, func() bool {
		for _, raw := range c.conn.JSONMessagesSince(since) {
			var message map[string]any
			if json.Unmarshal([]byte(raw), &message) == nil && message["header"] == header {
				found = message
				return true
			}
		}
		return false
	}, lifecycleTimeout, 5*time.Millisecond, "expected a %q message", header)

	return found
}

// headers are the headers of the messages received since mark.
func (c client) headers(since int) []string {
	var headers []string
	for _, raw := range c.conn.JSONMessagesSince(since) {
		var message server.Message
		if json.Unmarshal([]byte(raw), &message) == nil {
			headers = append(headers, message.Header)
		}
	}
	return headers
}

// startedMatch is a match whose toss Alice called and won or lost, and whose
// winner chose to play first.
func startedMatch(t *testing.T, options match.Options) (*match.Match, *recordingObserver, client, client, string) {
	t.Helper()

	m, observer := newLifecycleMatch(t, options)
	alice := connect(m, hostID, "Alice")
	bob := connect(m, guestID, "Bob")
	alice.waitFor(t, 0, "choose_toss")

	aliceMark := alice.mark()
	alice.send(m, map[string]any{"header": "toss_chosen", "prediction": 1})
	result := alice.waitFor(t, aliceMark, "toss_result")
	winner := result["winner"].(map[string]any)["uid"].(string)

	chooser := alice
	if winner == guestID {
		chooser = bob
	}
	chooser.waitFor(t, 0, "choose_turn")
	chooser.send(m, map[string]any{"header": "turn_chosen", "player": 1})
	alice.waitFor(t, aliceMark, "state_update")

	return m, observer, alice, bob, winner
}

func TestANewMatchNeedsTwoPlayersAndADisconnectGrace(t *testing.T) {
	system := match.NewSystem(match.SystemOptions{})

	for name, change := range map[string]func(*match.Options){
		"no guest":    func(o *match.Options) { o.Guest.ID = "" },
		"one player":  func(o *match.Options) { o.Guest.ID = hostID },
		"no grace":    func(o *match.Options) { o.DisconnectGrace = 0 },
		"minus grace": func(o *match.Options) { o.DisconnectGrace = -time.Second },
	} {
		options := lifecycleOptions(time.Minute)
		change(&options)
		_, err := system.NewMatch(options)
		assert.Error(t, err, name)
	}
}

func TestBothPlayersJoiningStartsTheCoinToss(t *testing.T) {
	m, _ := newLifecycleMatch(t, lifecycleOptions(time.Minute))

	alice := connect(m, hostID, "Alice")
	assert.Never(t, func() bool { return slices.Contains(alice.headers(0), "choose_toss") }, 100*time.Millisecond, 10*time.Millisecond,
		"the toss waits for both players")

	bob := connect(m, guestID, "Bob")

	ask := alice.waitFor(t, 0, "choose_toss")
	assert.Equal(t, map[string]any{"uid": hostID, "username": "Alice"}, ask["player"])
	wait := bob.waitFor(t, 0, "toss_being_chosen")
	assert.Equal(t, map[string]any{"uid": hostID, "username": "Alice"}, wait["player"], "Bob sees who is calling")
	assert.NotContains(t, bob.headers(0), "state_update", "the game has not started")
}

func TestTheTossCallerIsPickedAtRandomWhenNotChosen(t *testing.T) {
	callers := map[string]bool{}
	for range 40 {
		options := lifecycleOptions(time.Minute)
		options.TossCaller = ""
		m, _ := newLifecycleMatch(t, options)
		alice := connect(m, hostID, "Alice")
		connect(m, guestID, "Bob")

		require.Eventually(t, func() bool {
			headers := alice.headers(0)
			return slices.Contains(headers, "choose_toss") || slices.Contains(headers, "toss_being_chosen")
		}, lifecycleTimeout, 5*time.Millisecond)

		caller := guestID
		if slices.Contains(alice.headers(0), "choose_toss") {
			caller = hostID
		}
		callers[caller] = true
		m.Dispose()
	}

	assert.Len(t, callers, 2, "both players should get to call the toss sometimes")
}

func TestTheTossWinnerChoosesWhoPlaysFirst(t *testing.T) {
	m, _ := newLifecycleMatch(t, lifecycleOptions(time.Minute))
	alice := connect(m, hostID, "Alice")
	bob := connect(m, guestID, "Bob")
	alice.waitFor(t, 0, "choose_toss")

	// Only the caller calls, and only heads or tails.
	bobMark := bob.mark()
	bob.send(m, map[string]any{"header": "toss_chosen", "prediction": 1})
	alice.send(m, map[string]any{"header": "toss_chosen", "prediction": 7})
	assert.Never(t, func() bool { return slices.Contains(bob.headers(bobMark), "toss_result") }, 100*time.Millisecond, 10*time.Millisecond)

	alice.send(m, map[string]any{"header": "toss_chosen", "prediction": -1})
	result := bob.waitFor(t, bobMark, "toss_result")
	assert.Equal(t, "tails", result["call"])
	assert.Contains(t, []string{"heads", "tails"}, result["outcome"])
	assert.Equal(t, map[string]any{"uid": hostID, "username": "Alice"}, result["caller"])

	winner := result["winner"].(map[string]any)["uid"].(string)
	if result["outcome"] == "tails" {
		assert.Equal(t, hostID, winner, "Alice called it right")
	} else {
		assert.Equal(t, guestID, winner, "Alice called it wrong")
	}

	chooser, other, otherID := alice, bob, guestID
	if winner == guestID {
		chooser, other, otherID = bob, alice, hostID
	}
	chooser.waitFor(t, 0, "choose_turn")
	other.waitFor(t, 0, "turn_being_chosen")

	// Only the winner chooses: the other player's choice is ignored.
	otherMark := other.mark()
	other.send(m, map[string]any{"header": "turn_chosen", "player": 1})
	assert.Never(t, func() bool { return slices.Contains(other.headers(otherMark), "state_update") }, 100*time.Millisecond, 10*time.Millisecond)

	// The winner plays second.
	chooser.send(m, map[string]any{"header": "turn_chosen", "player": -1})
	other.waitFor(t, otherMark, "state_update")
	require.Eventually(t, func() bool { return !m.HasPendingEvents() }, lifecycleTimeout, 5*time.Millisecond)
	assert.Equal(t, otherID, m.CurrentPlayer().UID, "the other player plays first")

	// A second choice changes nothing.
	chooser.send(m, map[string]any{"header": "turn_chosen", "player": 1})
	require.Eventually(t, func() bool { return !m.HasPendingEvents() }, lifecycleTimeout, 5*time.Millisecond)
	assert.Equal(t, otherID, m.CurrentPlayer().UID)
}

func TestAPlayerComingBackDuringTheTossIsAskedAgain(t *testing.T) {
	m, observer := newLifecycleMatch(t, lifecycleOptions(time.Minute))
	alice := connect(m, hostID, "Alice")
	bob := connect(m, guestID, "Bob")
	alice.waitFor(t, 0, "choose_toss")

	bobMark := bob.mark()
	alice.socket.Close()
	bob.waitFor(t, bobMark, "opponent_disconnected")

	again := connect(m, hostID, "Alice")
	again.waitFor(t, 0, "choose_toss")
	bob.waitFor(t, bobMark, "opponent_reconnected")

	// The toss carries on from the new connection.
	again.send(m, map[string]any{"header": "toss_chosen", "prediction": 1})
	bob.waitFor(t, bobMark, "toss_result")
	assert.False(t, m.IsEnding())
	assert.Empty(t, observer.results())
}

func TestAPlayerComingBackAfterTheTossIsToldItsResult(t *testing.T) {
	m, _ := newLifecycleMatch(t, lifecycleOptions(time.Minute))
	alice := connect(m, hostID, "Alice")
	bob := connect(m, guestID, "Bob")
	alice.waitFor(t, 0, "choose_toss")
	alice.send(m, map[string]any{"header": "toss_chosen", "prediction": 1})
	bob.waitFor(t, 0, "toss_result")

	bob.socket.Close()
	again := connect(m, guestID, "Bob")

	result := again.waitFor(t, 0, "toss_result")
	assert.Equal(t, "heads", result["call"])
	if result["winner"].(map[string]any)["uid"] == guestID {
		again.waitFor(t, 0, "choose_turn")
	} else {
		again.waitFor(t, 0, "turn_being_chosen")
	}
}

func TestASpectatorWatchesTheToss(t *testing.T) {
	m, _ := newLifecycleMatch(t, lifecycleOptions(time.Minute))
	alice := connect(m, hostID, "Alice")
	connect(m, guestID, "Bob")
	alice.waitFor(t, 0, "choose_toss")

	watcher := connect(m, "watcher-3", "Watcher")
	wait := watcher.waitFor(t, 0, "toss_being_chosen")
	assert.Equal(t, map[string]any{"uid": hostID, "username": "Alice"}, wait["player"])

	alice.send(m, map[string]any{"header": "toss_chosen", "prediction": 1})
	watcher.waitFor(t, 0, "toss_result")
	watcher.waitFor(t, 0, "turn_being_chosen")
	assert.NotContains(t, watcher.headers(0), "choose_turn", "a spectator never chooses")
}

func TestSurrenderingLoses(t *testing.T) {
	options := lifecycleOptions(time.Minute)
	m, observer, alice, bob, _ := startedMatch(t, options)

	bobMark := bob.mark()
	alice.send(m, map[string]any{"header": "resign"})

	finished := bob.waitFor(t, bobMark, "duel_finished")
	assert.Equal(t, "surrender", finished["ending"])
	assert.Equal(t, map[string]any{"uid": guestID, "username": "Bob"}, finished["winner"])

	require.Eventually(t, m.IsClosed, lifecycleTimeout, 5*time.Millisecond)
	results := observer.results()
	require.Len(t, results, 1)
	assert.Equal(t, guestID, results[0].WinnerID)
	assert.Equal(t, match.EndingSurrender, results[0].Ending)
	assert.Equal(t, "Friendly", results[0].Name)
	assert.Equal(t, "open", results[0].FormatID)
	assert.Equal(t, []string{m.ID}, observer.closedMatches())
}

func TestAPlayerWhoDoesNotComeBackInTimeLoses(t *testing.T) {
	m, observer, alice, bob, _ := startedMatch(t, lifecycleOptions(150*time.Millisecond))

	bobMark := bob.mark()
	alice.socket.Close()
	bob.waitFor(t, bobMark, "opponent_disconnected")

	finished := bob.waitFor(t, bobMark, "duel_finished")
	assert.Equal(t, "disconnect", finished["ending"])
	assert.Equal(t, map[string]any{"uid": guestID, "username": "Bob"}, finished["winner"])

	require.Eventually(t, m.IsClosed, lifecycleTimeout, 5*time.Millisecond)
	results := observer.results()
	require.Len(t, results, 1)
	assert.Equal(t, guestID, results[0].WinnerID)
	assert.Equal(t, match.EndingDisconnect, results[0].Ending)
}

func TestComingBackInTimeKeepsTheGameGoing(t *testing.T) {
	m, observer, alice, bob, _ := startedMatch(t, lifecycleOptions(300*time.Millisecond))

	bobMark := bob.mark()
	alice.socket.Close()
	bob.waitFor(t, bobMark, "opponent_disconnected")

	again := connect(m, hostID, "Alice")
	again.waitFor(t, 0, "state_update")
	bob.waitFor(t, bobMark, "opponent_reconnected")

	// Well past the time limit, the game goes on.
	assert.Never(t, m.IsEnding, 600*time.Millisecond, 20*time.Millisecond)
	assert.Empty(t, observer.results())
	assert.NotContains(t, bob.headers(bobMark), "duel_finished")
}

func TestBothPlayersLeavingEndsTheGameAtOnce(t *testing.T) {
	// The time limit is far away: the game must not wait for it.
	m, observer, alice, bob, _ := startedMatch(t, lifecycleOptions(time.Hour))

	bob.socket.Close()
	alice.socket.Close()

	require.Eventually(t, m.IsClosed, lifecycleTimeout, 5*time.Millisecond)
	results := observer.results()
	require.Len(t, results, 1)
	assert.Equal(t, hostID, results[0].WinnerID, "Bob left first, so Bob loses")
	assert.Equal(t, match.EndingDisconnect, results[0].Ending)
}

func TestLeavingBeforeTheOpponentJoinsClosesTheMatchWithoutAResult(t *testing.T) {
	m, observer := newLifecycleMatch(t, lifecycleOptions(100*time.Millisecond))
	alice := connect(m, hostID, "Alice")
	require.Eventually(t, func() bool { return m.Connected(hostID) }, lifecycleTimeout, 5*time.Millisecond)

	alice.socket.Close()

	require.Eventually(t, m.IsClosed, lifecycleTimeout, 5*time.Millisecond)
	assert.Empty(t, observer.results(), "a game that never started has no result")
	assert.Equal(t, []string{m.ID}, observer.closedMatches())
}

func TestAShortGameIsReportedButDoesNotCount(t *testing.T) {
	options := lifecycleOptions(time.Minute)
	options.MinimumCountedDuration = time.Hour
	m, observer, alice, bob, _ := startedMatch(t, options)

	bobMark := bob.mark()
	alice.send(m, map[string]any{"header": "resign"})

	finished := bob.waitFor(t, bobMark, "duel_finished")
	assert.Equal(t, false, finished["counted"])
	require.Eventually(t, func() bool { return len(observer.results()) == 1 }, lifecycleTimeout, 5*time.Millisecond)
	assert.False(t, observer.results()[0].Counted)
}

func TestConnectedTellsWhoIsInTheMatch(t *testing.T) {
	m, _ := newLifecycleMatch(t, lifecycleOptions(time.Minute))
	assert.False(t, m.Connected(hostID), "not joined yet")

	alice := connect(m, hostID, "Alice")
	require.Eventually(t, func() bool { return m.Connected(hostID) }, lifecycleTimeout, 5*time.Millisecond)
	assert.False(t, m.Connected(guestID))
	assert.False(t, m.Connected("watcher-3"), "only players count")

	alice.socket.Close()
	assert.False(t, m.Connected(hostID), "gone and not back")
}

func TestPlayersJoiningAtTheSameMomentStartOneToss(t *testing.T) {
	for range 20 {
		m, _ := newLifecycleMatch(t, lifecycleOptions(time.Minute))

		var wg sync.WaitGroup
		joined := make([]client, 2)
		for i, seat := range []match.Seat{{ID: hostID, Username: "Alice"}, {ID: guestID, Username: "Bob"}} {
			wg.Go(func() { joined[i] = connect(m, seat.ID, seat.Username) })
		}
		wg.Wait()

		alice := joined[0]
		alice.waitFor(t, 0, "choose_toss")
		assert.Never(t, func() bool {
			count := 0
			for _, header := range alice.headers(0) {
				if header == "choose_toss" {
					count++
				}
			}
			return count > 1
		}, 50*time.Millisecond, 5*time.Millisecond, "the toss begins once")
		m.Dispose()
	}
}

func TestComingBackWhileTheOpponentIsAwayKeepsTheMatch(t *testing.T) {
	m, observer, alice, bob, _ := startedMatch(t, lifecycleOptions(time.Hour))

	// Bob leaves; Alice drops and comes back while he is still away.
	bob.socket.Close()
	require.Eventually(t, func() bool { return !m.Connected(guestID) }, lifecycleTimeout, 5*time.Millisecond)
	again := connect(m, hostID, "Alice")
	again.waitFor(t, 0, "state_update")
	alice.socket.Close()

	assert.Never(t, m.IsEnding, 200*time.Millisecond, 10*time.Millisecond,
		"closing the replaced connection must not count as Alice leaving")
	assert.Empty(t, observer.results())
	assert.True(t, m.Connected(hostID))
}

func TestJoiningAnEndedMatchIsRefused(t *testing.T) {
	m, _ := newLifecycleMatch(t, lifecycleOptions(time.Minute))
	m.Dispose()

	late := connect(m, guestID, "Bob")

	late.waitFor(t, 0, "error")
	require.Eventually(t, late.socket.IsClosed, lifecycleTimeout, 5*time.Millisecond)
}
