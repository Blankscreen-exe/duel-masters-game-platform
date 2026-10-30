// Package computer plays games as a player does, for the computer opponent.
//
// A computer player takes a seat over a connection inside the server, the
// engine sending it what it would send a browser in that seat and reading
// the moves it makes, so it never sees what a player could not: the other
// player's hand, the shields or the order of a deck. What a card does is no
// secret, so it reads the cards it is shown in its Library.
//
// It plays with rules of thumb, valuing each move by what it would leave on
// the board, as its Style weighs things.
package computer

import (
	"cmp"
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// Coin toss answers, as the engine takes them.
const (
	callHeads = 1
	playFirst = 1
)

const (
	// settle is how long the game must be quiet before the computer acts, so
	// it acts on the game as the engine has left it rather than part way
	// through a move.
	settle = 25 * time.Millisecond
	// retry is how long it waits for the engine to answer one of its moves
	// before it looks at the game again, as the engine says nothing back to
	// a move it ignores.
	retry = time.Second
)

// Options configure a computer player.
type Options struct {
	Library *Library
	Style   Style
	// Strength is how well it plays; Normal unless set.
	Strength Strength
	// Pause is how long it waits before each move, so a person playing it
	// can follow what it does.
	Pause time.Duration
	// Seed makes its random choices, such as which face-down shield to
	// break, the same every time; zero picks them anew.
	Seed uint64
	// Logger hears what it does, the moves it makes at debug level; nil
	// hears nothing.
	Logger *slog.Logger
}

// Player is a computer player in one game.
type Player struct {
	conn     *connection
	library  *Library
	style    Style
	strength Strength
	pause    time.Duration
	rand     *rand.Rand
	logger   *slog.Logger

	// view and plan are only used by Play.
	view view
	plan turnPlan
}

// New returns a computer player. Give its Connection to the game, then Play.
func New(options Options) *Player {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	seed := options.Seed
	if seed == 0 {
		seed = rand.Uint64()
	}
	return &Player{
		conn:     newConnection(),
		library:  options.Library,
		style:    options.Style,
		strength: cmp.Or(options.Strength, Normal),
		pause:    options.Pause,
		rand:     rand.New(rand.NewPCG(seed, seed>>1|1)),
		logger:   logger,
	}
}

// Connection is the connection the player plays over, to seat it in a game
// as a browser's would be.
func (pl *Player) Connection() server.Connection { return pl.conn }

// Play plays the game until its connection closes, which the game does once
// it is over. It is to be run on a goroutine of its own, which it ends by
// returning.
func (pl *Player) Play() {
	// It joins the game as the duel board does, with its first message.
	pl.conn.send(headerOnly{Header: "join_match"})

	quiet := time.NewTimer(settle)
	defer quiet.Stop()

	for {
		select {
		case <-pl.conn.closed:
			return
		case <-pl.conn.arrived:
			for _, message := range pl.conn.take() {
				pl.view.take(message)
				if pl.logger.Enabled(context.Background(), slog.LevelDebug) {
					pl.logger.Debug("the computer is told", "message", string(message))
				}
			}
			quiet.Reset(settle)
		case <-quiet.C:
			if move, ok := pl.decide(); ok {
				if !pl.wait(pl.pause) {
					return
				}
				pl.logger.Debug("the computer moves", "move", move)
				pl.conn.send(move)
			}
			quiet.Reset(retry)
		}
	}
}

// wait waits for d, and reports false if the connection closed meanwhile.
func (pl *Player) wait(d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-pl.conn.closed:
		return false
	}
}

// tossCall and turnChoice are the coin toss's answers.
type tossCall struct {
	Header     string `json:"header"`
	Prediction int    `json:"prediction"`
}

type turnChoice struct {
	Header string `json:"header"`
	Player int    `json:"player"`
}

// decide is the computer's next move, if it has one to make now.
func (pl *Player) decide() (any, bool) {
	v := &pl.view
	switch {
	case v.finished:
		return nil, false
	case v.callToss:
		v.callToss = false
		call := callHeads
		if pl.rand.IntN(2) == 0 {
			call = -callHeads
		}
		return tossCall{Header: "toss_chosen", Prediction: call}, true
	case v.chooseTurn:
		v.chooseTurn = false
		return turnChoice{Header: "turn_chosen", Player: playFirst}, true
	case v.question != nil:
		reply, ok := pl.answerTo(v.question, v.state)
		if !ok {
			return pl.giveUp(), true
		}
		return reply, true
	case v.waiting || v.state == nil || !v.state.MyTurn:
		return nil, false
	}
	return pl.nextMove(v.state), true
}

// giveUp ends the game for a question the computer found no answer the
// engine takes, rather than leave its opponent waiting for ever.
func (pl *Player) giveUp() any {
	q := pl.view.question
	pl.logger.Warn("the computer found no answer the engine takes, and gives up the game",
		"kind", q.about.Kind, "type", q.actionType, "min", q.min, "max", q.max, "offered", len(q.cards))
	pl.view.finished = true
	return headerOnly{Header: "resign"}
}
