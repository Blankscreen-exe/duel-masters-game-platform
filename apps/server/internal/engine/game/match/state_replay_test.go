package match

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// recordingConnection keeps the JSON messages sent over it.
type recordingConnection struct {
	mu   sync.Mutex
	sent []server.MatchStateMessage
}

func (*recordingConnection) SetReadLimit(int64)                {}
func (*recordingConnection) SetReadDeadline(time.Time) error   { return nil }
func (*recordingConnection) SetPongHandler(func(string) error) {}
func (*recordingConnection) ReadMessage() (int, []byte, error) { return 0, nil, nil }
func (*recordingConnection) SetWriteDeadline(time.Time) error  { return nil }
func (*recordingConnection) WriteMessage(int, []byte) error    { return nil }
func (*recordingConnection) Close() error                      { return nil }

func (c *recordingConnection) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var message server.MatchStateMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, message)
	return nil
}

func TestSomeoneArrivingGetsTheLatestStateWithoutReplayingWhatIsOver(t *testing.T) {
	m := &Match{HostID: "host-1", GuestID: "guest-2"}
	state := func(spectator bool) *server.MatchStateMessage {
		return &server.MatchStateMessage{Header: "state_update", State: server.MatchState{
			Spectator:      spectator,
			BrokenShields:  []string{"shield-1"},
			ActivatedCards: []string{"card-1"},
		}}
	}
	host, guest, spectator := state(false), state(false), state(true)
	host.State.MyTurn = true
	m.keepStates(host, guest, spectator)

	for uid, want := range map[string]struct{ myTurn, spectator bool }{
		"host-1":  {myTurn: true},
		"guest-2": {},
		"watcher": {spectator: true},
	} {
		conn := &recordingConnection{}
		m.sendLatestState(server.NewSocket(conn, m, uid, uid))

		if len(conn.sent) != 1 {
			t.Fatalf("%s: expected one state, got %d", uid, len(conn.sent))
		}
		got := conn.sent[0].State
		if got.MyTurn != want.myTurn || got.Spectator != want.spectator {
			t.Fatalf("%s: got the state for another viewer: %+v", uid, got)
		}
		if len(got.BrokenShields) != 0 || len(got.ActivatedCards) != 0 {
			t.Fatalf("%s: broken shields and acting cards must not replay, got %v and %v", uid, got.BrokenShields, got.ActivatedCards)
		}
	}

	if len(host.State.BrokenShields) != 1 {
		t.Fatal("the state sent at the time keeps its broken shields")
	}
}

func TestNothingIsSentBeforeTheFirstState(t *testing.T) {
	m := &Match{HostID: "host-1", GuestID: "guest-2"}
	conn := &recordingConnection{}

	m.sendLatestState(server.NewSocket(conn, m, "host-1", "Host"))

	if len(conn.sent) != 0 {
		t.Fatalf("expected nothing, got %d messages", len(conn.sent))
	}
}
