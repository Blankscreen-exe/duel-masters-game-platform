package match

import (
	"context"
	"errors"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/internal"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/ventu-io/go-shortid"
)

// unstartedLifetime is how long a match may wait for its players to join and
// settle the coin toss before it is closed.
const unstartedLifetime = 10 * time.Minute

// tickInterval is how often the match system looks for matches to close.
const tickInterval = 10 * time.Second

// SystemOptions configure a match system.
type SystemOptions struct {
	// Observer hears about every match finishing and closing; nil ignores
	// them.
	Observer Observer
	// DevCommands lets players change the game from the chat, such as adding
	// cards to their hand. Only for local testing: never on a public server.
	DevCommands bool
}

type MatchSystem struct {
	Matches     internal.ConcurrentDictionary[Match]
	observer    Observer
	devCommands bool
}

// MatchSummary is a point-in-time view of a live match. It deliberately
// excludes decks, card state, sockets, and other private state.
type MatchSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	FormatID      string `json:"formatId"`
	FormatName    string `json:"formatName"`
	HostID        string `json:"hostId"`
	HostUsername  string `json:"hostUsername"`
	GuestID       string `json:"guestId"`
	GuestUsername string `json:"guestUsername"`
	Visible       bool   `json:"visible"`
	Matchmaking   bool   `json:"matchmaking"`
	CreatedAt     int64  `json:"createdAt"`
}

func NewSystem(options SystemOptions) *MatchSystem {
	observer := options.Observer
	if observer == nil {
		observer = NoObserver{}
	}

	return &MatchSystem{
		Matches:     internal.NewConcurrentDictionary[Match](),
		observer:    observer,
		devCommands: options.DevCommands,
	}
}

// RunTicker closes matches that never started, until ctx ends.
func (s *MatchSystem) RunTicker(ctx context.Context) {
	defer internal.Recover()

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, m := range s.Matches.Iter() {
				ProcessMatch(m)
			}
		}
	}
}

// ProcessMatch closes a match that has waited too long for its players to
// join and settle the coin toss.
func ProcessMatch(m *Match) {

	defer internal.Recover()

	if !m.hasStarted() && m.created < time.Now().Add(-unstartedLifetime).Unix() {
		logrus.Debugf("Closing match %s", m.ID)
		m.Dispose()
	}
}

// NewMatch creates a match for two known players and starts its event loop.
// The players then join it over their sockets; once both have, the coin toss
// decides who plays first.
func (s *MatchSystem) NewMatch(options Options) (*Match, error) {
	switch {
	case options.Host.ID == "" || options.Guest.ID == "":
		return nil, errors.New("a match needs a host and a guest")
	case options.Host.ID == options.Guest.ID:
		return nil, errors.New("a match needs two different players")
	case options.DisconnectGrace <= 0:
		return nil, errors.New("a match needs a positive disconnect grace")
	}

	id, err := shortid.Generate()

	if err != nil {
		id = uuid.New().String()
	}

	m := &Match{
		ID:                id,
		MatchName:         options.Name,
		FormatID:          options.Format.ID,
		FormatName:        options.Format.Name,
		HostID:            options.Host.ID,
		HostDeck:          options.Host.Deck,
		GuestID:           options.Guest.ID,
		GuestDeck:         options.Guest.Deck,
		spectators:        internal.NewConcurrentDictionary[Spectator](),
		persistentEffects: make(map[int]PersistentEffect),
		Turn:              1,

		Visible: options.Visible,

		Matchmaking: options.Matchmaking,
		created:     time.Now().Unix(),
		isFirstTurn: true,
		turnsPlayed: 0,

		eventloop: NewEventLoop(),

		system: s,

		hostUsername:  options.Host.Username,
		guestUsername: options.Guest.Username,

		tossCaller:             options.TossCaller,
		minimumCountedDuration: options.MinimumCountedDuration,
		disconnectGrace:        options.DisconnectGrace,
		disconnectTimers:       map[string]*time.Timer{},
	}

	go m.eventloop.start()

	s.Matches.Add(id, m)

	logrus.Debugf("Created match %s", id)

	return m, nil

}

// MatchSummaries returns point-in-time copies of every match currently owned
// by the match system.
func (s *MatchSystem) MatchSummaries() []MatchSummary {
	matches := s.Matches.Iter()
	summaries := make([]MatchSummary, 0, len(matches))

	for _, m := range matches {
		summaries = append(summaries, m.Summary())
	}

	return summaries
}

// Summary returns a point-in-time view of the match, from what does not change
// once it is created.
func (m *Match) Summary() MatchSummary {
	return MatchSummary{
		ID:            m.ID,
		Name:          m.MatchName,
		FormatID:      m.FormatID,
		FormatName:    m.FormatName,
		HostID:        m.HostID,
		HostUsername:  m.hostUsername,
		GuestID:       m.GuestID,
		GuestUsername: m.guestUsername,
		Visible:       m.Visible,
		Matchmaking:   m.Matchmaking,
		CreatedAt:     m.created,
	}
}
