package computer

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"
)

const (
	// DeskOnlineWindow is how recently someone must have been in touch with the
	// desk to count as there. Waiting for a game keeps them in touch.
	DeskOnlineWindow = 3 * time.Minute
	// DeskAbandonAfter is how long a live player may stay silent before their
	// game is given up, rather than leave their opponent waiting for ever. It
	// is well beyond the time a move takes to think over.
	DeskAbandonAfter = 15 * time.Minute
	// deskAbandonCheck is how often a live game looks for a player who has gone.
	deskAbandonCheck = 30 * time.Second
)

// Desk is where a player from outside the server takes up games: one player,
// playing one game at a time. It is safe for concurrent use.
type Desk struct {
	library *Library
	now     func() time.Time
	// abandonCheck is deskAbandonCheck, outside tests.
	abandonCheck time.Duration

	mu          sync.Mutex
	lastContact time.Time
	// games are the games seated and not yet over and told of, oldest first; a
	// second one seated in the moment before the first was seen waits its turn.
	games []*Live
	// seated is closed and replaced whenever a game is seated.
	seated chan struct{}
}

// NewDesk returns a desk. now is the clock, time.Now outside tests.
func NewDesk(library *Library, now func() time.Time) *Desk {
	return &Desk{library: library, now: now, abandonCheck: deskAbandonCheck, seated: make(chan struct{})}
}

// Touch records that the player is there.
func (d *Desk) Touch() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastContact = d.now()
}

// Online reports whether the player has been in touch lately.
func (d *Desk) Online() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.lastContact.IsZero() && d.now().Sub(d.lastContact) < DeskOnlineWindow
}

// abandoned reports whether the player has been silent too long.
func (d *Desk) abandoned() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.now().Sub(d.lastContact) >= DeskAbandonAfter
}

// Free reports whether no game is under way at the desk.
func (d *Desk) Free() bool {
	d.mu.Lock()
	games := slices.Clone(d.games)
	d.mu.Unlock()

	for _, game := range games {
		if !game.isOver() {
			return false
		}
	}
	return true
}

// Seat seats the desk's player in a new game. It is an Options.Outside.
func (d *Desk) Seat(logger *slog.Logger) Seat {
	game := newLive(d, logger)

	d.mu.Lock()
	defer d.mu.Unlock()

	// A game starting gives its player the whole time to turn up.
	d.lastContact = d.now()
	d.games = append(d.games, game)
	close(d.seated)
	d.seated = make(chan struct{})
	return game
}

// Current returns the game the player is to deal with: the oldest not yet over,
// or else one that is over that they have yet to be told of.
func (d *Desk) Current() (*Live, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.games = slices.DeleteFunc(d.games, (*Live).isSpent)
	for _, game := range d.games {
		if !game.isOver() {
			return game, true
		}
	}
	if len(d.games) > 0 {
		return d.games[0], true
	}
	return nil, false
}

// WaitForGame returns the current game, waiting up to longest for one to be
// seated, or false when none was or ctx ended.
func (d *Desk) WaitForGame(ctx context.Context, longest time.Duration) (*Live, bool) {
	deadline := time.NewTimer(longest)
	defer deadline.Stop()

	for {
		d.mu.Lock()
		seated := d.seated
		d.mu.Unlock()

		if game, ok := d.Current(); ok {
			return game, true
		}

		select {
		case <-seated:
		case <-deadline.C:
			return nil, false
		case <-ctx.Done():
			return nil, false
		}
	}
}
