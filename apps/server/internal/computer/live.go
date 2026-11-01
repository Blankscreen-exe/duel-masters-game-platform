package computer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

const (
	// liveSettle is how long the game must be quiet before a live player is
	// told it is their move, so they decide on the game as the engine has left
	// it rather than part way through a move.
	liveSettle = 150 * time.Millisecond
	// liveMoveWindow is how long the engine has to answer a move before it
	// counts as refused, as the engine says nothing back to a move it ignores.
	liveMoveWindow = 2 * time.Second
	// maxLiveEvents is the most lines of the game log kept between two looks at
	// the game.
	maxLiveEvents = 80
)

// What a live player is to do next, or why they have nothing to do.
const (
	StatusNoGame       = "no_game"
	StatusCallCoin     = "call_coin"
	StatusChooseOrder  = "choose_order"
	StatusQuestion     = "question"
	StatusYourTurn     = "your_turn"
	StatusOpponentMove = "opponent_to_move"
	StatusFinished     = "finished"
)

// Moves a live player can make on their turn.
const (
	MoveChargeMana     = "charge_mana"
	MovePlay           = "play"
	MoveAttackPlayer   = "attack_player"
	MoveAttackCreature = "attack_creature"
	MoveTapAbility     = "tap_ability"
	MoveEndTurn        = "end_turn"
)

// ErrNotNow is the error for a move or answer the game has no place for at the
// moment: it says what is wrong.
var ErrNotNow = errors.New("not possible right now")

// Live is a seat in one game played from outside the server, by whoever holds
// the Desk: it does not choose moves itself but keeps what the seat has been
// told and passes on the moves it is given. It sees only what a player in the
// seat could. It is safe for concurrent use.
type Live struct {
	conn    *connection
	desk    *Desk
	library *Library
	logger  *slog.Logger

	mu   sync.Mutex
	view view
	// changed is closed and replaced whenever anything arrives from the engine.
	changed chan struct{}
	// lastMessage is when the engine last said anything.
	lastMessage time.Time
	// moveSentAt is when a move was sent the engine has yet to react to.
	moveSentAt time.Time
	// events and refused are what happened, and what the engine refused, since
	// the last look at the game.
	events  []string
	refused []string
	// described are the cards whose rules text has been given already, by name.
	described map[string]bool
	result    *LiveResult
	reported  bool
}

// LiveResult is how a live player's game ended.
type LiveResult struct {
	Won bool `json:"won"`
	// Winner is empty for a game with none, such as one stopped by the server.
	Winner string `json:"winner,omitempty"`
	Ending string `json:"ending"`
	Turns  int    `json:"turns,omitempty"`
}

func newLive(desk *Desk, logger *slog.Logger) *Live {
	return &Live{
		conn:      newConnection(),
		desk:      desk,
		library:   desk.library,
		logger:    logger,
		changed:   make(chan struct{}),
		described: map[string]bool{},
	}
}

// Connection is the connection the seat plays over.
func (l *Live) Connection() server.Connection { return l.conn }

// Play follows the game until its connection closes, which the game does once
// it is over. It is to be run on a goroutine of its own, which it ends by
// returning.
func (l *Live) Play() {
	l.conn.send(headerOnly{Header: "join_match"})

	abandon := time.NewTicker(l.desk.abandonCheck)
	defer abandon.Stop()

	for {
		select {
		case <-l.conn.closed:
			// The game's last words may have come just before it closed.
			l.takeMessages()
			l.closed()
			return
		case <-l.conn.arrived:
			l.takeMessages()
		case <-abandon.C:
			if l.desk.abandoned() {
				l.logger.Warn("the live player has gone, so their game is given up")
				l.conn.send(headerOnly{Header: "resign"})
			}
		}
	}
}

// takeMessages reads what the engine has sent.
func (l *Live) takeMessages() {
	messages := l.conn.take()

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, message := range messages {
		l.view.take(message)
		l.note(message)
	}
	if len(messages) > 0 {
		l.lastMessage = l.desk.now()
		l.moveSentAt = time.Time{}
		close(l.changed)
		l.changed = make(chan struct{})
	}
}

// note keeps what a message says that the view does not. l.mu must be held.
func (l *Live) note(data []byte) {
	var message struct {
		server.ChatMessage
		Cards json.RawMessage `json:"cards"`
	}
	if json.Unmarshal(data, &message) != nil {
		return
	}

	switch message.Header {
	case "chat":
		l.event(l.chatLine(message.Sender, message.Message))
	case "warn", "action_error", "error":
		l.refused = append(l.refused, message.Message)
	case "show_cards", "show_cards_non_dismissible":
		var shown server.ShowCardsMessage
		if json.Unmarshal(data, &shown) == nil {
			l.event(fmt.Sprintf("Shown to you: %s %v", shown.Message, l.cardNames(shown.Cards)))
		}
	case "duel_finished":
		var finished server.DuelFinishedMessage
		if json.Unmarshal(data, &finished) == nil {
			l.finish(finished)
		}
	}
}

// chatLine writes a chat message as a line of the game log. The game's own
// lines have a sender starting "Server"; anything else is a player talking.
func (l *Live) chatLine(sender, message string) string {
	if strings.HasPrefix(sender, "Server") {
		return message
	}
	return sender + " says: " + message
}

func (l *Live) event(line string) {
	l.events = append(l.events, line)
	if extra := len(l.events) - maxLiveEvents; extra > 0 {
		l.events = slices.Delete(l.events, 0, extra)
	}
}

// cardNames names cards the engine shows by their engine ids.
func (l *Live) cardNames(ids []string) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if card, ok := l.library.Card(id); ok {
			names = append(names, card.Name)
		} else {
			names = append(names, id)
		}
	}
	return names
}

func (l *Live) finish(finished server.DuelFinishedMessage) {
	me := ""
	if l.view.state != nil {
		me = l.view.state.Me.Username
	}
	result := &LiveResult{Ending: finished.Ending, Turns: finished.Turns}
	if finished.Winner != nil {
		result.Winner = finished.Winner.Username
		result.Won = result.Winner == me
	}
	l.result = result
}

// closed settles a game that closed, told how it ended or not.
func (l *Live) closed() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.result == nil {
		l.result = &LiveResult{Ending: "stopped"}
	}
	l.view.finished = true
	l.view.question = nil
	close(l.changed)
	l.changed = make(chan struct{})
}

// isOver reports whether the game has ended.
func (l *Live) isOver() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.result != nil
}

// isSpent reports whether the game has ended and the player has been told.
func (l *Live) isSpent() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.result != nil && l.reported
}

// Wait returns the game as it stands once it is the player's move or the game
// is over, or as it stands after longest or when ctx ends.
func (l *Live) Wait(ctx context.Context, longest time.Duration) Situation {
	deadline := time.NewTimer(longest)
	defer deadline.Stop()

	for {
		l.mu.Lock()
		ready, again := l.settled()
		changed := l.changed
		if ready {
			defer l.mu.Unlock()
			return l.situation()
		}
		l.mu.Unlock()

		recheck := time.NewTimer(again)
		select {
		case <-changed:
		case <-recheck.C:
		case <-deadline.C:
			recheck.Stop()
			return l.Situation()
		case <-ctx.Done():
			recheck.Stop()
			return l.Situation()
		}
		recheck.Stop()
	}
}

// settled reports whether the player can decide now, and otherwise how long
// until that is worth asking again. l.mu must be held.
func (l *Live) settled() (bool, time.Duration) {
	if l.result != nil {
		return true, 0
	}

	now := l.desk.now()
	if !l.moveSentAt.IsZero() {
		if waited := now.Sub(l.moveSentAt); waited < liveMoveWindow {
			return false, liveMoveWindow - waited
		}
		l.moveSentAt = time.Time{}
		l.refused = append(l.refused, "The game did not take that move, and nothing changed.")
	}
	if quiet := now.Sub(l.lastMessage); quiet < liveSettle {
		return false, liveSettle - quiet
	}
	if l.status() == StatusOpponentMove {
		return false, time.Minute
	}
	return true, 0
}

// status is what the player is to do. l.mu must be held.
func (l *Live) status() string {
	v := &l.view
	switch {
	case l.result != nil:
		return StatusFinished
	case v.callToss:
		return StatusCallCoin
	case v.chooseTurn:
		return StatusChooseOrder
	case v.question != nil:
		return StatusQuestion
	case v.state != nil && v.state.MyTurn && !v.waiting:
		return StatusYourTurn
	}
	return StatusOpponentMove
}

// Situation returns the game as it stands, with what happened since the last
// look at it.
func (l *Live) Situation() Situation {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.situation()
}

// CallCoin calls the coin toss.
func (l *Live) CallCoin(heads bool) error {
	l.mu.Lock()
	if !l.view.callToss {
		l.mu.Unlock()
		return fmt.Errorf("%w: you are not calling the coin toss", ErrNotNow)
	}
	l.view.callToss = false
	l.mu.Unlock()

	prediction := callHeads
	if !heads {
		prediction = -callHeads
	}
	l.send(tossCall{Header: "toss_chosen", Prediction: prediction})
	return nil
}

// ChooseOrder chooses to play first or second, for the toss's winner.
func (l *Live) ChooseOrder(first bool) error {
	l.mu.Lock()
	if !l.view.chooseTurn {
		l.mu.Unlock()
		return fmt.Errorf("%w: you are not choosing who plays first", ErrNotNow)
	}
	l.view.chooseTurn = false
	l.mu.Unlock()

	choice := playFirst
	if !first {
		choice = -playFirst
	}
	l.send(turnChoice{Header: "turn_chosen", Player: choice})
	return nil
}

// Move makes a move of the player's own turn, with a card of theirs unless it
// is ending the turn. It returns an error wrapping ErrNotNow for a move the
// game has no place for; the engine has the last word on the rest.
func (l *Live) Move(kind, cardID string) error {
	move, err := l.moveFor(kind, cardID)
	if err != nil {
		return err
	}
	l.send(move)
	return nil
}

// moveFor is the move to send for what the player asks.
func (l *Live) moveFor(kind, cardID string) (any, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if status := l.status(); status != StatusYourTurn {
		return nil, fmt.Errorf("%w: it is not time for a move, the status is %s", ErrNotNow, status)
	}

	var zone []server.CardState
	switch kind {
	case MoveEndTurn:
		return headerOnly{Header: "end_turn"}, nil
	case MoveChargeMana, MovePlay:
		zone = l.view.state.Me.Hand
	case MoveAttackPlayer, MoveAttackCreature, MoveTapAbility:
		zone = l.view.state.Me.Battlezone
	default:
		return nil, fmt.Errorf("%w: there is no move called %q", ErrNotNow, kind)
	}
	if !slices.ContainsFunc(zone, func(card server.CardState) bool { return card.CardID == cardID }) {
		return nil, fmt.Errorf("%w: card %q is not where that move needs it", ErrNotNow, cardID)
	}

	header := kind
	switch kind {
	case MoveChargeMana:
		header = "add_to_manazone"
	case MovePlay:
		header = "add_to_playzone"
	}
	return cardMove{Header: header, ID: cardID}, nil
}

// Answer answers the open question: with the cards chosen, the number or
// choice given, or by cancelling, which is also how to answer no. It returns
// an error wrapping ErrNotNow when no question is open or a card was not
// offered; the engine has the last word on the rest.
func (l *Live) Answer(cardIDs []string, count int, cancel bool) error {
	reply, err := l.answerFor(cardIDs, count, cancel)
	if err != nil {
		return err
	}
	l.send(reply)
	return nil
}

// answerFor is the answer to send for what the player gives.
func (l *Live) answerFor(cardIDs []string, count int, cancel bool) (answer, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	q := l.view.question
	if q == nil {
		return answer{}, fmt.Errorf("%w: no question is open", ErrNotNow)
	}
	for _, id := range cardIDs {
		offered := slices.ContainsFunc(q.selectable(), func(card server.CardState) bool { return card.CardID == id })
		if !offered {
			return answer{}, fmt.Errorf("%w: card %q is not offered", ErrNotNow, id)
		}
	}
	if cardIDs == nil {
		cardIDs = []string{}
	}

	return answer{Header: "action", Prompt: q.id, Cards: cardIDs, Count: count, Cancel: cancel || q.show}, nil
}

// Say says something in the game's chat.
func (l *Live) Say(message string) {
	// Nothing comes back to wait for but the line itself, so this is no move.
	l.conn.send(chatMove{Header: "chat", Message: message})
}

// Resign gives up the game.
func (l *Live) Resign() {
	l.conn.send(headerOnly{Header: "resign"})
}

// chatMove is a line of chat, as the duel board sends it.
type chatMove struct {
	Header  string `json:"header"`
	Message string `json:"message"`
}

// send sends a move, marking it first as one the engine is yet to react to, as
// its reaction may come before this returns.
func (l *Live) send(move any) {
	l.mu.Lock()
	l.moveSentAt = l.desk.now()
	l.mu.Unlock()

	l.conn.send(move)
}
