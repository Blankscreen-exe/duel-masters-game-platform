// Package realtime keeps the site's live connections: one websocket per open
// tab, logged in or not, which the server uses to push lobby updates, chat and
// notices such as a game starting. It only carries messages; the features
// decide what to send, and to whom.
package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Connection limits.
const (
	// writeWait is how long a message may take to send before the connection
	// is given up.
	writeWait = 10 * time.Second
	// pongWait is how long the connection may stay silent: the browser
	// answers every ping.
	pongWait = 60 * time.Second
	// pingPeriod is how often the connection is pinged, within pongWait.
	pingPeriod = pongWait * 9 / 10
	// maxMessageSize bounds what a browser may send; the site sends nothing
	// but answers to pings over the connection.
	maxMessageSize = 512
	// sendQueue is how many messages may wait for a slow connection before it
	// is dropped, so one stuck tab never holds up everyone else.
	sendQueue = 64
)

// Viewer is who a connection belongs to. A visitor has no user id.
type Viewer struct {
	UserID   string
	Username string
}

// LoggedIn reports whether the viewer has an account.
func (v Viewer) LoggedIn() bool { return v.UserID != "" }

// Presence is told when a player may have come online or gone offline: when
// their first connection opens, and once their last one has stayed closed for
// the offline delay. Notices from different goroutines can arrive in any
// order, so it asks IsOnline for the truth rather than trusting which notice
// came last. It is called on the hub's goroutines, never while the hub is
// locked, and must not block.
type Presence interface {
	PresenceChanged(Viewer)
}

// Options configure a hub.
type Options struct {
	// OfflineAfter is how long a player with no connection still counts as
	// online, so reloading a page or a moment without a network does not
	// count as leaving.
	OfflineAfter time.Duration
	// Presence hears players come and go; nil ignores them.
	Presence Presence
	Logger   *slog.Logger
}

// Hub keeps the live connections and sends them messages. It is safe for
// concurrent use.
type Hub struct {
	offlineAfter time.Duration
	presence     Presence
	logger       *slog.Logger

	mu      sync.Mutex
	clients map[*client]struct{}
	// byUser holds the connections of each logged-in player.
	byUser map[string]map[*client]struct{}
	// leaving holds, by player, the timers that send players offline.
	leaving map[string]*time.Timer
	// viewers remembers the players online, by id.
	viewers map[string]Viewer
	closed  bool
}

// ErrClosed is returned for a connection offered to a hub that has closed.
var ErrClosed = errors.New("realtime hub closed")

// NewHub returns a hub with no connections.
func NewHub(options Options) *Hub {
	return &Hub{
		offlineAfter: options.OfflineAfter,
		presence:     options.Presence,
		logger:       options.Logger,
		clients:      map[*client]struct{}{},
		byUser:       map[string]map[*client]struct{}{},
		leaving:      map[string]*time.Timer{},
		viewers:      map[string]Viewer{},
	}
}

// client is one live connection.
type client struct {
	conn   *websocket.Conn
	viewer Viewer
	send   chan []byte
	// gone is closed once the connection is to stop, by whichever side
	// notices first.
	gone     chan struct{}
	goneOnce sync.Once
}

func (c *client) stop() {
	c.goneOnce.Do(func() { close(c.gone) })
}

// Serve keeps a websocket connection, already upgraded, until it closes or
// the hub does, and returns once it is done with it. It returns ErrClosed,
// closing the connection, when the hub has closed.
func (h *Hub) Serve(conn *websocket.Conn, viewer Viewer) error {
	c := &client{conn: conn, viewer: viewer, send: make(chan []byte, sendQueue), gone: make(chan struct{})}
	if err := h.add(c); err != nil {
		_ = conn.Close()
		return err
	}
	defer h.remove(c)

	written := make(chan struct{})
	go func() {
		defer close(written)
		h.write(c)
	}()

	h.read(c)
	c.stop()
	<-written

	// Already closed when it was told to stop; this only makes sure.
	_ = conn.Close()
	return nil
}

// read reads until the connection fails, answering pings and keeping it
// alive. Browsers send nothing else that matters.
func (h *Hub) read(c *client) {
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Closing the connection when it is told to stop ends the read below.
	go func() {
		<-c.gone
		_ = c.conn.Close()
	}()

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// write sends the connection its messages and pings, until it stops.
func (h *Hub) write(c *client) {
	ping := time.NewTicker(pingPeriod)
	defer ping.Stop()

	for {
		select {
		case <-c.gone:
			_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""), time.Now().Add(writeWait))
			return
		case message := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.stop()
				return
			}
		case <-ping.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				c.stop()
				return
			}
		}
	}
}

func (h *Hub) add(c *client) error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return ErrClosed
	}

	h.clients[c] = struct{}{}
	cameOnline := false
	if c.viewer.LoggedIn() {
		uid := c.viewer.UserID
		if timer, ok := h.leaving[uid]; ok {
			timer.Stop()
			delete(h.leaving, uid)
		}
		if h.byUser[uid] == nil {
			h.byUser[uid] = map[*client]struct{}{}
		}
		h.byUser[uid][c] = struct{}{}
		_, known := h.viewers[uid]
		cameOnline = !known
		h.viewers[uid] = c.viewer
	}
	h.mu.Unlock()

	if cameOnline && h.presence != nil {
		h.presence.PresenceChanged(c.viewer)
	}

	return nil
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, c)
	if !c.viewer.LoggedIn() {
		return
	}

	uid := c.viewer.UserID
	delete(h.byUser[uid], c)
	if len(h.byUser[uid]) > 0 || h.closed {
		return
	}
	delete(h.byUser, uid)

	h.leaving[uid] = time.AfterFunc(h.offlineAfter, func() { h.goOffline(uid) })
}

// goOffline sends a player offline once their last connection has stayed
// closed for the offline delay.
func (h *Hub) goOffline(uid string) {
	h.mu.Lock()
	if _, waiting := h.leaving[uid]; !waiting || len(h.byUser[uid]) > 0 {
		h.mu.Unlock()
		return
	}
	delete(h.leaving, uid)
	viewer := h.viewers[uid]
	delete(h.viewers, uid)
	h.mu.Unlock()

	if h.presence != nil {
		h.presence.PresenceChanged(viewer)
	}
}

// Broadcast sends a message to every connection, visitors included.
func (h *Hub) Broadcast(message any) {
	h.deliver(message, func(*client) bool { return true })
}

// SendTo sends a message to every connection of one player.
func (h *Hub) SendTo(userID string, message any) {
	h.deliver(message, func(c *client) bool { return c.viewer.UserID == userID })
}

func (h *Hub) deliver(message any, to func(*client) bool) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("encoding a live message failed", "error", err)
		return
	}

	h.mu.Lock()
	recipients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		if to(c) {
			recipients = append(recipients, c)
		}
	}
	h.mu.Unlock()

	for _, c := range recipients {
		select {
		case c.send <- data:
		case <-c.gone:
		default:
			// Too far behind to catch up: drop it, and the browser reconnects.
			h.logger.Warn("dropping a live connection that is not keeping up", "user", c.viewer.UserID)
			c.stop()
		}
	}
}

// Online lists the players online, by name.
func (h *Hub) Online() []Viewer {
	h.mu.Lock()
	online := make([]Viewer, 0, len(h.viewers))
	for _, viewer := range h.viewers {
		online = append(online, viewer)
	}
	h.mu.Unlock()

	slices.SortFunc(online, func(a, b Viewer) int {
		switch {
		case a.Username < b.Username:
			return -1
		case a.Username > b.Username:
			return 1
		default:
			return 0
		}
	})

	return online
}

// IsOnline reports whether a player is online.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, online := h.viewers[userID]
	return online
}

// Close ends every connection and refuses new ones, waiting until the
// connections have stopped or ctx ends. Players are not reported offline:
// the server is stopping.
func (h *Hub) Close(ctx context.Context) error {
	h.mu.Lock()
	h.closed = true
	for uid, timer := range h.leaving {
		timer.Stop()
		delete(h.leaving, uid)
	}
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		c.stop()
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		h.mu.Lock()
		left := len(h.clients)
		h.mu.Unlock()
		if left == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("closing live connections: %d still open: %w", left, ctx.Err())
		case <-ticker.C:
		}
	}
}
