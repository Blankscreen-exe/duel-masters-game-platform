package games

import (
	"context"

	"golang.org/x/time/rate"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
)

// MessagesPerSecond and MessageBurst are the admin settings for how fast a
// connection to a game may send moves and messages: a burst of MessageBurst at
// once, then MessagesPerSecond a second. Faster ones wait their turn.
var (
	MessagesPerSecond = settings.Number{
		Key:         "games.messages_per_second",
		Group:       "Games",
		Description: "How many moves and messages a second a connection to a game may send, once the burst below is used up. Faster ones wait their turn, so a flood slows down instead of swamping the game; nothing is lost.",
		Default:     10,
		Min:         1,
		Max:         100,
	}
	MessageBurst = settings.Number{
		Key:         "games.message_burst",
		Group:       "Games",
		Description: "How many moves and messages a connection to a game may send at once before the pace above applies.",
		Default:     20,
		Min:         1,
		Max:         200,
	}
)

// pacedConnection passes a connection's messages on to the engine no faster
// than the settings allow: a message past the pace waits its turn, and the
// connection reads nothing more meanwhile. Closing it ends a wait, so a
// player put out of the site leaves at once.
type pacedConnection struct {
	server.Connection
	settings *settings.Service
	pace     *rate.Limiter
	closed   context.Context
	close    context.CancelFunc
}

func newPacedConnection(conn server.Connection, siteSettings *settings.Service) *pacedConnection {
	closed, close := context.WithCancel(context.Background())
	perSecond, burst := paceOf(siteSettings)
	return &pacedConnection{
		Connection: conn,
		settings:   siteSettings,
		pace:       rate.NewLimiter(perSecond, burst),
		closed:     closed,
		close:      close,
	}
}

// paceOf is the pace the settings allow at the moment.
func paceOf(siteSettings *settings.Service) (perSecond rate.Limit, burst int) {
	return rate.Limit(siteSettings.Number(MessagesPerSecond)), int(siteSettings.Number(MessageBurst))
}

// ReadMessage reads the next message, and returns it once the pace allows,
// following any change an admin has made to the settings.
func (c *pacedConnection) ReadMessage() (int, []byte, error) {
	kind, data, err := c.Connection.ReadMessage()
	if err != nil {
		return kind, data, err
	}

	perSecond, burst := paceOf(c.settings)
	if c.pace.Limit() != perSecond {
		c.pace.SetLimit(perSecond)
	}
	if c.pace.Burst() != burst {
		c.pace.SetBurst(burst)
	}

	if err := c.pace.Wait(c.closed); err != nil {
		return 0, nil, err
	}
	return kind, data, nil
}

func (c *pacedConnection) Close() error {
	c.close()
	return c.Connection.Close()
}
