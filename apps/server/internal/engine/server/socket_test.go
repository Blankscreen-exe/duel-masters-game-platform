package server

import (
	"errors"
	"testing"
	"time"
)

// limitConnection records the read limit a socket sets, then ends at once.
type limitConnection struct {
	limit int64
}

func (c *limitConnection) SetReadLimit(limit int64)          { c.limit = limit }
func (c *limitConnection) SetReadDeadline(time.Time) error   { return nil }
func (c *limitConnection) SetPongHandler(func(string) error) {}
func (c *limitConnection) ReadMessage() (int, []byte, error) {
	return 0, nil, errors.New("closed")
}
func (c *limitConnection) SetWriteDeadline(time.Time) error { return nil }
func (c *limitConnection) WriteMessage(int, []byte) error   { return nil }
func (c *limitConnection) WriteJSON(any) error              { return nil }
func (c *limitConnection) Close() error                     { return nil }

type silentHub struct{}

func (silentHub) Parse(*Socket, []byte) {}
func (silentHub) Name() string          { return "test" }
func (silentHub) OnSocketClose(*Socket) {}

func TestASocketAcceptsAnActionSelectingEveryCard(t *testing.T) {
	conn := &limitConnection{}

	NewSocket(conn, silentHub{}, "player", "Player").Listen()

	// 80 cards with the longest ids, as {"header":"action","cards":[...]}.
	largestAction := len(`{"header":"action","cards":[],"cancel":false,"count":0}`) + 80*(36+3)
	if conn.limit < int64(largestAction) {
		t.Fatalf("the read limit %d is below the largest action, %d bytes", conn.limit, largestAction)
	}
}
