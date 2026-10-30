package computer

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// errClosed is what reading a closed connection returns.
var errClosed = errors.New("the computer's connection is closed")

// connection is the computer's end of a game, inside the server: the engine
// sends to it as to a browser, and reads the moves the computer makes from
// it. It is a server.Connection.
//
// What the engine sends is queued, so sending never waits for the computer:
// the engine sends from its event loop, which must not be held up. The
// queue cannot grow without end, as a game only sends so much between two
// of the computer's turns to read it.
type connection struct {
	mu    sync.Mutex
	inbox [][]byte
	// arrived wakes the computer when the inbox has something; it holds one
	// wake-up at most.
	arrived chan struct{}

	// moves carries the computer's moves to the engine.
	moves chan []byte

	closed    chan struct{}
	closeOnce sync.Once
}

var _ server.Connection = (*connection)(nil)

func newConnection() *connection {
	return &connection{
		arrived: make(chan struct{}, 1),
		moves:   make(chan []byte),
		closed:  make(chan struct{}),
	}
}

func (c *connection) SetReadLimit(int64)                {}
func (c *connection) SetReadDeadline(time.Time) error   { return nil }
func (c *connection) SetPongHandler(func(string) error) {}
func (c *connection) SetWriteDeadline(time.Time) error  { return nil }

// WriteMessage takes the engine's pings, which need no answer here.
func (c *connection) WriteMessage(int, []byte) error { return nil }

// WriteJSON queues a message from the engine for the computer.
func (c *connection) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.inbox = append(c.inbox, data)
	c.mu.Unlock()

	select {
	case c.arrived <- struct{}{}:
	default:
	}
	return nil
}

// ReadMessage returns the computer's next move, as the engine reads a
// browser's, until the connection closes.
func (c *connection) ReadMessage() (int, []byte, error) {
	select {
	case move := <-c.moves:
		return websocket.TextMessage, move, nil
	case <-c.closed:
		return 0, nil, errClosed
	}
}

// Close closes the connection, ending the engine's reading of it and the
// computer's play. Closing it again changes nothing.
func (c *connection) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

// take returns what the engine has sent since it was last called.
func (c *connection) take() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := c.inbox
	c.inbox = nil
	return messages
}

// send hands a move to the engine, or gives up once the connection closes.
func (c *connection) send(move any) {
	data, err := json.Marshal(move)
	if err != nil {
		return
	}

	select {
	case c.moves <- data:
	case <-c.closed:
	}
}
