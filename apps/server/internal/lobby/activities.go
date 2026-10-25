package lobby

import (
	"errors"
	"sync"
)

// Kind is what a player is doing.
type Kind string

const (
	// Hosting: waiting in an open game for an opponent.
	Hosting Kind = "hosting"
	// Queued: waiting in quick match.
	Queued Kind = "queued"
	// Playing: in a running game.
	Playing Kind = "playing"
	// Rematching: deciding on a rematch of a game just ended.
	Rematching Kind = "rematch"
)

// Activity is what a player is doing, with the game or format it concerns.
type Activity struct {
	Kind Kind
	// GameID is the open game hosted, the game played, or the game just ended
	// whose rematch is being decided.
	GameID string
	// FormatName is the format waited for in quick match.
	FormatName string
}

// BusyError is returned to a player starting something while already doing
// something else: a player hosts, waits in quick match, plays or decides on
// a rematch, one at a time.
type BusyError struct {
	Current Activity
}

func (e *BusyError) Error() string {
	switch e.Current.Kind {
	case Hosting:
		return "already hosting a game"
	case Queued:
		return "already waiting for a quick match"
	case Rematching:
		return "already deciding on a rematch"
	default:
		return "already playing a game"
	}
}

// IsBusy reports whether err is, or wraps, a *BusyError.
func IsBusy(err error) bool {
	var busy *BusyError
	return errors.As(err, &busy)
}

// Activities keeps what every player is doing, so each does one thing at a
// time. The lobby and quick match share one. It is safe for concurrent use.
type Activities struct {
	mu       sync.Mutex
	byUser   map[string]Activity
	onChange func()
}

// NewActivities returns an empty register. onChange is called after every
// change, outside any lock; nil ignores changes.
func NewActivities(onChange func()) *Activities {
	return &Activities{byUser: map[string]Activity{}, onChange: onChange}
}

// Claim records what a player starts doing, or returns a *BusyError when
// they are doing something already.
func (a *Activities) Claim(userID string, activity Activity) error {
	a.mu.Lock()
	if current, busy := a.byUser[userID]; busy {
		a.mu.Unlock()
		return &BusyError{Current: current}
	}
	a.byUser[userID] = activity
	a.mu.Unlock()

	a.changed()
	return nil
}

// Change replaces what a player is doing, if it is still from, such as
// hosting becoming playing once an opponent joins. It reports whether it did.
func (a *Activities) Change(userID string, from Kind, to Activity) bool {
	a.mu.Lock()
	current, ok := a.byUser[userID]
	changed := ok && current.Kind == from
	if changed {
		a.byUser[userID] = to
	}
	a.mu.Unlock()

	if changed {
		a.changed()
	}
	return changed
}

// Release ends what a player is doing, if it is still kind (and, for a
// non-empty gameID, about that game), and reports whether it did.
func (a *Activities) Release(userID string, kind Kind, gameID string) bool {
	a.mu.Lock()
	current, ok := a.byUser[userID]
	released := ok && current.Kind == kind && (gameID == "" || current.GameID == gameID)
	if released {
		delete(a.byUser, userID)
	}
	a.mu.Unlock()

	if released {
		a.changed()
	}
	return released
}

// Of returns what a player is doing, if anything.
func (a *Activities) Of(userID string) (Activity, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity, ok := a.byUser[userID]
	return activity, ok
}

func (a *Activities) changed() {
	if a.onChange != nil {
		a.onChange()
	}
}
