package match

import "time"

// Seat is a player's place in a match, decided before the match is created.
type Seat struct {
	ID       string
	Username string
	// Deck is the engine card ids of the player's 40 cards.
	Deck []string
}

// Options describe a match to create.
type Options struct {
	Name   string
	Host   Seat
	Guest  Seat
	Format FormatDescriptor
	// Visible lists the match publicly; Matchmaking marks it as made by quick
	// match. The engine only reports them back.
	Visible     bool
	Matchmaking bool
	// TossCaller is the id of the player who calls the coin toss that decides
	// who plays first. Anyone else, or empty, picks the caller at random.
	TossCaller string
	// MinimumCountedDuration is how long a game must last for its result to
	// count (Result.Counted). Shorter games are still reported.
	MinimumCountedDuration time.Duration
	// DisconnectGrace is how long a disconnected player has to come back
	// before losing. It must be positive.
	DisconnectGrace time.Duration
}

// Ending is how a game ended. The values are part of the messages sent to the
// players.
type Ending string

const (
	// EndingPlay: a player won by the game's rules, such as an attack with no
	// shields left or the opponent running out of cards.
	EndingPlay Ending = "play"
	// EndingSurrender: a player gave up.
	EndingSurrender Ending = "surrender"
	// EndingDisconnect: a player left and did not come back in time.
	EndingDisconnect Ending = "disconnect"
)

// ResultPlayer is a player as a result reports them.
type ResultPlayer struct {
	ID       string
	Username string
	Deck     []string
}

// Result is a finished game, as the engine reports it. Only games that
// started (got past the coin toss) have a result.
type Result struct {
	MatchID    string
	Name       string
	FormatID   string
	FormatName string
	Host       ResultPlayer
	Guest      ResultPlayer
	// WinnerID is the winning player's id, or empty when there is none.
	WinnerID  string
	Ending    Ending
	StartedAt time.Time
	EndedAt   time.Time
	// Turns counts the turns played, extra turns left out.
	Turns int
	// Counted is false for a game shorter than MinimumCountedDuration.
	Counted bool
}

// Duration is how long the game lasted.
func (r Result) Duration() time.Duration {
	return max(r.EndedAt.Sub(r.StartedAt), 0)
}

// Observer is told when matches finish and close. Its methods are called from
// the match's own goroutines, sometimes while the match is ending, so they
// must return quickly and must not call back into the match; work that takes
// time, such as writing to a database, belongs on the observer's own
// goroutine.
type Observer interface {
	// MatchFinished reports a game that started and then ended.
	MatchFinished(Result)
	// MatchClosed reports that a match is gone, finished or not. It is called
	// once per match, after any MatchFinished.
	MatchClosed(matchID string)
}

// NoObserver ignores every match, for tests and tools that do not need to
// hear about them.
type NoObserver struct{}

func (NoObserver) MatchFinished(Result)       {}
func (NoObserver) MatchClosed(matchID string) {}
