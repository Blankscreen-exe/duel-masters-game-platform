package match

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/internal"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"math/rand"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// Match struct
type Match struct {
	ID                string                                   `json:"id"`
	MatchName         string                                   `json:"name"`
	FormatID          string                                   `json:"-"`
	FormatName        string                                   `json:"-"`
	HostID            string                                   `json:"-"`
	HostDeck          []string                                 `json:"-"`
	GuestID           string                                   `json:"-"`
	GuestDeck         []string                                 `json:"-"`
	Player1           *PlayerReference                         `json:"-"`
	Player2           *PlayerReference                         `json:"-"`
	spectators        internal.ConcurrentDictionary[Spectator] `json:"-"`
	persistentEffects map[int]PersistentEffect
	// lastPersistentEffectID is the key of the persistent effect added last.
	// Like persistentEffects, it is only used on the match's event loop.
	lastPersistentEffectID int
	Turn                   byte `json:"-"`
	Visible                bool `json:"visible"`
	Step                   interface{}

	// started is set once the coin toss is settled and the game begins. It is
	// read from socket goroutines and the match system's ticker, so it must be
	// race free.
	started atomic.Bool

	// The coin toss, guarded by setupMu. PlayerSelectingToss is the id of the
	// player who calls it; TossPrediction is the call and TossOutcome how the
	// coin landed, each 1 for heads and -1 for tails, 0 until known.
	PlayerSelectingToss string
	TossPrediction      int
	TossOutcome         int
	// tossCaller is the player chosen to call the toss when the match was
	// created; empty or anyone else picks one at random.
	tossCaller string
	// tossBegun is set once both players have joined and the toss has begun;
	// turnChosen once the toss's winner has chosen who plays first.
	tossBegun  bool
	turnChosen bool

	// minimumCountedDuration is how long a game must last for its result to
	// count. disconnectGrace is how long a disconnected player has to come
	// back before losing.
	minimumCountedDuration time.Duration
	disconnectGrace        time.Duration
	// chatRefusal is Options.ChatRefusal: nil lets every chat message through.
	chatRefusal func(userID string) string

	// setupMu guards the players taking their seats and changing sockets as
	// they join, leave and come back, the coin toss, and disconnectTimers. It
	// is never held while sending to a socket or closing one, since closing a
	// socket calls back into the match.
	setupMu sync.Mutex
	// disconnectTimers hold, by player id, the time limits of the players who
	// have disconnected and not come back yet.
	disconnectTimers map[string]*time.Timer

	Matchmaking bool
	created     int64
	// ending and closed are read by the match event loop and written by socket
	// goroutines when a player disconnects, so they must be race free.
	ending      atomic.Bool
	closed      atomic.Bool
	isFirstTurn bool
	startedAt   time.Time
	turnsPlayed int
	// turnNumber counts every turn begun, extra turns included, for the duel
	// interface. turnsPlayed, kept for the duel record, leaves extra turns out.
	turnNumber int

	eventloop *EventLoop
	system    *MatchSystem

	// resolvingAttack is set while the event loop resolves an attack or a tap
	// ability.
	resolvingAttack *attackInProgress

	// board is what the event loop last worked out about the board. It is
	// replaced, never changed, so a copy read under the mutex stays valid.
	board      boardReport
	boardMutex sync.Mutex

	// recentCards are the cards that last moved or acted, most recent last,
	// for telling which card a chat line names. See noteRecent.
	recentCards      []string
	recentCardsMutex sync.Mutex

	// lastStates are the states last sent to each viewer, for someone joining
	// or coming back, guarded by statesMu.
	lastStates sentStates
	statesMu   sync.Mutex

	// brokenShields holds the ids of the shields broken since the event loop
	// last broadcast the state.
	brokenShields []string

	// activatedCards holds the ids of the cards that acted since the event loop
	// last broadcast the state.
	activatedCards []string

	hostUsername  string
	guestUsername string
}

// Name just returns "match", obligatory for a hub
func (m *Match) Name() string {
	return "match"
}

// IsClosed returns true once the match has been disposed. Effects that wait for
// a player's response use it to abandon their prompt instead of asking a player
// that can no longer answer.
func (m *Match) IsClosed() bool {
	return m != nil && m.closed.Load()
}

// IsEnding returns true once the match has started ending or has been disposed.
func (m *Match) IsEnding() bool {
	return m != nil && (m.ending.Load() || m.closed.Load())
}

// EventLoopStopped reports whether the match's event loop goroutine has exited.
// A disposed match whose loop is still running is stuck in, or spinning on, an
// event that never finished.
func (m *Match) EventLoopStopped() bool {
	return m != nil && !m.eventloop.isRunning()
}

// HasPendingEvents reports whether the match still has sequential events to
// run: scheduled, queued or running, including one waiting on a player's
// prompt. Code acting on the match from outside its event loop, such as a test
// driving it, waits for this to report false before reading what an event
// changes. Unlike sending the match an event to see when it is answered,
// checking adds nothing to the event loop's queue.
func (m *Match) HasPendingEvents() bool {
	return m != nil && m.eventloop.hasPending()
}

// Dispose closes the match, disconnects the clients and removes all references to it
func (m *Match) Dispose() {

	if m.closed.Swap(true) {
		return
	}

	// Deferred first so it runs last, after the match is gone, even if
	// disposing it panics part way.
	defer m.system.observer.MatchClosed(m.ID)

	// Stop the event loop from resolving anything further. Disposing the players
	// below ends any wait for their answers, which releases the event loop from
	// any prompt it is currently waiting on, and it must unwind without
	// resolving more of the game.
	m.ending.Store(true)

	logrus.Debugf("Disposing match %s", m.ID)

	defer func() {
		if r := recover(); r != nil {
			logrus.Warningf("Recovered from disposing a match. %v", r)
			debug.PrintStack()
		}
	}()

	m.eventloop.stop()

	m.setupMu.Lock()
	for uid, timer := range m.disconnectTimers {
		timer.Stop()
		delete(m.disconnectTimers, uid)
	}
	player1, player2 := m.Player1, m.Player2
	m.setupMu.Unlock()

	for _, spectator := range m.spectators.Iter() {
		if spectator.Socket == nil {
			continue
		}
		// Closed but kept: another goroutine may be sending to it right now,
		// and sending to a closed socket does nothing.
		spectator.Socket.Close()
	}

	if player1 != nil {
		player1.Dispose()
	}

	if player2 != nil {
		player2.Dispose()
	}

	m.system.Matches.Remove(m.ID)

	logrus.Debugf("Closed match with id %s", m.ID)

}

// IsPlayerTurn returns a boolean based on if it is the specified player's turn
func (m *Match) IsPlayerTurn(p *Player) bool {
	return m.Turn == p.Turn
}

// CurrentPlayer returns either player1 or player2 based on who's turn it currently is
func (m *Match) CurrentPlayer() *PlayerReference {

	if m.Turn == 1 {
		return m.Player1
	}

	return m.Player2

}

// PlayerForSocket returns the player ref for a given socker or an error if the socket is not p1 or p2
func (m *Match) PlayerForSocket(s *server.Socket) (*PlayerReference, error) {

	if m.Player1.Socket == s {
		return m.Player1, nil
	}

	if m.Player2.Socket == s {
		return m.Player2, nil
	}

	return nil, errors.New("Socket is not player1 or player2")

}

// PlayerRef returns the player ref for a given player
func (m *Match) PlayerRef(p *Player) *PlayerReference {

	if m.Player1.Player == p {
		return m.Player1
	}

	return m.Player2

}

// Opponent returns the opponent of the given player
func (m *Match) Opponent(p *Player) *Player {
	if m.Player1.Player == p {
		return m.Player2.Player
	}

	return m.Player1.Player
}

// GetPower returns the power of a given card after applying conditions
func (m *Match) GetPower(card *Card, isAttacking bool) int {

	power := card.Power

	for _, condition := range card.Conditions() {

		switch condition.ID {

		case cnd.PowerAmplifier:
			{
				if val, ok := condition.Val.(int); ok {
					power += val
				}
			}

		case cnd.PowerAttacker:
			{
				if !isAttacking {
					continue
				}

				if val, ok := condition.Val.(int); ok {
					power += val
				}
			}

		}

	}

	power += card.PowerModifier(m, isAttacking)

	e := &GetPowerEvent{
		Card:      card,
		Attacking: isAttacking,
		Power:     power,
	}
	m.HandleFx(NewContext(m, e))

	return e.Power

}

// CastSpell Fires a SpellCast event
func (m *Match) CastSpell(card *Card, fromShield bool) {

	m.noteActivated(card.ID)

	m.HandleFx(NewContext(m, &SpellCast{
		CardID:        card.ID,
		FromShield:    fromShield,
		MatchPlayerID: m.getPlayerMatchId(card.Player),
	}))

	m.BroadcastState()

}

func (m *Match) getPlayerMatchId(player *Player) byte {
	if player == m.Player1.Player {
		return 1
	}
	return 2
}

// Battle handles a battle between two creatures
func (m *Match) Battle(attacker *Card, defender *Card, blocked bool, fromAttackPlayer bool) {

	if blocked {
		m.noteActivated(defender.ID)
	}

	// The battle is reported with both creatures' names, whoever wins.
	m.noteRecent(attacker.ID)
	m.noteRecent(defender.ID)

	attackerPower := m.GetPower(attacker, true)
	defenderPower := m.GetPower(defender, false)

	m.HandleFx(NewContext(m, &Battle{Attacker: attacker, AttackerPower: attackerPower, Defender: defender, DefenderPower: defenderPower, Blocked: blocked, FromAttackPlayer: fromAttackPlayer}))

	m.BroadcastState()

}

// Destroy sends the given card to its players graveyard
func (m *Match) Destroy(card *Card, source *Card, context CreatureDestroyedContext) {

	if card.Zone != BATTLEZONE {
		return
	}
	m.ReportActionInChat(card.Player, fmt.Sprintf("%s (%v) was destroyed by %s", card.Name, m.GetPower(card, false), source.Name))
	m.HandleFx(NewContext(m, &CreatureDestroyed{Card: card, Source: source, Context: context}))

}

// MoveCard moves a card and sends a chat message about what source moved it
func (m *Match) MoveCard(card *Card, destination string, source *Card) {

	_, err := card.Player.MoveCard(card.ID, card.Zone, destination, source.ID)

	if err != nil {
		return
	}

	m.ReportActionInChat(card.Player, fmt.Sprintf("%s was moved to %s %s by %s", card.Name, card.Player.Username(), destination, source.Name))

}

// MoveCardToFront moves a card and sends a chat message about what source moved it
func (m *Match) MoveCardToFront(card *Card, destination string, source *Card) {

	_, err := card.Player.MoveCardToFront(card.ID, card.Zone, destination, source.ID)

	if err != nil {
		return
	}

	m.ReportActionInChat(card.Player, fmt.Sprintf("%s was moved to %s's %s by %s", card.Name, card.Player.Username(), destination, source.Name))

}

// BreakShields breaks the given shields and handles shieldtriggers
func (m *Match) BreakShields(attemptedShields []*Card, source *Card) {

	if len(attemptedShields) < 1 {
		return
	}

	event := &BreakShieldEvent{
		Cards:  attemptedShields,
		Source: source,
	}
	ctx := NewContext(m, event)
	m.HandleFx(ctx)
	if ctx.cancel {
		return
	}
	shields := event.Cards
	// An effect may take every shield out of the break without cancelling it.
	if len(shields) == 0 {
		return
	}

	m.ReportActionInChat(shields[0].Player, fmt.Sprintf("%v of %v's shields were broken", len(shields), m.PlayerRef(shields[0].Player).Socket.User.Username))

	var shieldTriggers []*Card

	for _, shield := range shields {

		card, err := shield.Player.MoveCard(shield.ID, SHIELDZONE, HAND, source.ID)

		if err != nil {
			continue
		}

		// A move that an effect prevented returns the card without an error.
		if card.Zone == HAND {
			m.brokenShields = append(m.brokenShields, card.ID)
		}

		m.HandleFx(NewContext(m, &BrokenShieldEvent{CardID: card.ID, Source: source.ID}))

		// Handle shield triggers
		if card.HasCondition(cnd.ShieldTrigger) {
			shieldTriggers = append(shieldTriggers, card)
		}

	}

	m.ResolveShieldTriggers(shieldTriggers, source)

}

// ResolveShieldTriggers offers the given cards, which must already be in their
// owner's hand, to be played for free one at a time.
//
// Breaking shields goes through BreakShields, which calls this once the shields
// have moved. Effects that put shields into a hand without breaking them and
// still allow their shield triggers call this directly, so that no shield break
// event is dispatched on their behalf.
func (m *Match) ResolveShieldTriggers(shieldTriggers []*Card, source *Card) {

	for len(shieldTriggers) > 0 {

		// we broadcast state here as it's needed to move the cards to the hand and
		// after each used shieldtrigger
		m.BroadcastState()

		event := &ShieldTriggerEvent{
			Cards:  shieldTriggers,
			Source: source.ID,
		}
		ctx := NewContext(m, event)
		m.HandleFx(ctx)

		playableShieldTriggers := event.Cards

		if ctx.Cancelled() || len(playableShieldTriggers) == 0 {
			return
		}

		player := shieldTriggers[0].Player
		opponent := m.Opponent(player)

		m.Wait(opponent, "Waiting for your opponent to make an action")

		m.NewActionFullList(
			player,
			playableShieldTriggers,
			1,
			1,
			"Shield trigger! If those cards are playable you can do so for free one at a time.",
			true,
			event.UnplayableCards,
		)

		for {

			action := player.NextAction()

			if action.Cancel {
				m.CloseAction(player)
				m.EndWait(opponent)
				return
			}

			if len(action.Cards) != 1 {
				m.DefaultActionWarning(player)
				continue
			}

			var card *Card
			for _, shieldTrigger := range playableShieldTriggers {
				if shieldTrigger.ID == action.Cards[0] {
					card = shieldTrigger
				}
			}

			if card == nil {
				m.DefaultActionWarning(player)
				continue
			}

			if card.HasCondition(cnd.Spell) {
				m.CastSpell(card, true)
			} else {
				if moved, err := card.Player.MoveCard(card.ID, HAND, BATTLEZONE, "shield_trigger"); err == nil && moved.Zone == BATTLEZONE {
					m.noteActivated(moved.ID)
				}
			}

			// Eliminate all shield triggers from that list that are not in hand anymore for any reason.
			// We need to also make sure to specifically eliminate the played card, for the cases where the card
			// can return itself to the hand. e.g. a card with bounce that can bounce itself back to hand.
			var stillValidShieldtriggers []*Card
			for _, shieldTrigger := range shieldTriggers {
				if shieldTrigger.Zone == HAND && shieldTrigger != card {
					stillValidShieldtriggers = append(stillValidShieldtriggers, shieldTrigger)
				}
			}
			shieldTriggers = stillValidShieldtriggers

			m.HandleFx(NewContext(m, &ShieldTriggerPlayedEvent{
				Card:   card,
				Source: source.ID,
			}))

			m.CloseAction(card.Player)

			break

		}

		m.EndWait(opponent)

	}

}

// End ends the match: winner won by the game's rules, and message says how.
// Card effects and the game's own rules call it.
func (m *Match) End(winner *Player, message string) {
	m.end(winner, message, EndingPlay)
}

// Abort ends the match without a winner, such as when the server stops. A
// game that started is reported as aborted, and never counts.
func (m *Match) Abort(message string) {
	m.end(nil, message, EndingAborted)
}

// end ends the match, reporting the result of a game that started, telling
// everyone who won and closing the match. Only the first ending counts.
func (m *Match) end(winner *Player, message string, ending Ending) {

	logrus.Debugf("Attempting to end match %s", m.ID)

	if m.ending.Swap(true) {
		logrus.Debugf("Cannot end match, %s is already ending", m.ID)
		return
	}

	if m.hasStarted() {
		result := m.result(winner, ending, time.Now())
		m.system.observer.MatchFinished(result)

		m.Broadcast(server.WarningMessage{
			Header:  "error",
			Message: message,
		})

		m.Broadcast(m.duelFinishedMessage(result))
	}

	m.Dispose()

}

// result is the finished game, ended at endedAt.
func (m *Match) result(winner *Player, ending Ending, endedAt time.Time) Result {
	m.setupMu.Lock()
	player1, player2 := m.Player1, m.Player2
	m.setupMu.Unlock()

	result := Result{
		MatchID:    m.ID,
		Name:       m.MatchName,
		FormatID:   m.FormatID,
		FormatName: m.FormatName,
		Host:       ResultPlayer{ID: m.HostID, Username: m.hostUsername, Deck: m.HostDeck},
		Guest:      ResultPlayer{ID: m.GuestID, Username: m.guestUsername, Deck: m.GuestDeck},
		Ending:     ending,
		StartedAt:  m.startedAt,
		EndedAt:    endedAt,
		Turns:      m.turnsPlayed,
	}
	result.Counted = ending != EndingAborted && result.Duration() >= m.minimumCountedDuration

	switch {
	case winner == nil:
	case player1 != nil && player1.Player == winner:
		result.WinnerID = m.HostID
	case player2 != nil && player2.Player == winner:
		result.WinnerID = m.GuestID
	}

	return result
}

// duelFinishedMessage tells everyone in the match how the game ended.
func (m *Match) duelFinishedMessage(result Result) server.DuelFinishedMessage {
	message := server.DuelFinishedMessage{
		Header:          "duel_finished",
		DuelID:          result.MatchID,
		Ending:          string(result.Ending),
		Counted:         result.Counted,
		Turns:           result.Turns,
		DurationSeconds: int64(result.Duration().Seconds()),
	}

	switch result.WinnerID {
	case "":
	case result.Host.ID:
		message.Winner = &server.DuelFinishedPlayer{UID: result.Host.ID, Username: result.Host.Username}
	case result.Guest.ID:
		message.Winner = &server.DuelFinishedPlayer{UID: result.Guest.ID, Username: result.Guest.Username}
	}

	return message
}

// ColorChat sends a chat message with color
func (m *Match) ColorChat(sender string, message string, color string) {
	msg := &server.ChatMessage{
		Header:  "chat",
		Message: message,
		Sender:  sender,
		Color:   color,
	}

	m.Broadcast(msg)
}

func (m *Match) ReportActionInChat(player *Player, message string) {
	sender := fmt.Sprintf("Server_%d", m.getPlayerMatchId(player))
	m.Broadcast(&server.ChatMessage{
		Header:  "chat",
		Message: message,
		Sender:  sender,
		Color:   "#ccc",
		Cards:   m.mentionedCards(message),
	})
}

// Chat sends a chat message with the default color
func (m *Match) Chat(sender string, message string) {
	m.ColorChat(sender, message, "#ccc")
}

func (m *Match) Broadcast(msg interface{}) {

	defer func() {
		if r := recover(); r != nil {
			logrus.Warnf("Recovered during Broadcast(). %v", r)
		}
	}()

	// The seats are read under setupMu, since before the game starts players
	// take them from other goroutines. A seat is empty until its player joins.
	m.setupMu.Lock()
	player1, player2 := m.Player1, m.Player2
	m.setupMu.Unlock()

	for _, player := range []*PlayerReference{player1, player2} {
		if player != nil && player.Socket != nil {
			player.Socket.Send(msg)
		}
	}

	// could fail due to concurrent updates to spectators map
	// but don't want to lock it here as broadcast will be used everywhere
	// so recover will deal with those special occasians where it fails
	for _, spectator := range m.spectators.Iter() {
		if spectator.Socket == nil {
			continue
		}
		spectator.Socket.Send(msg)
	}

}

// BroadcastState sends the current game's state to both players, hiding the opponent's hand.
// Only the event loop may call it, because working out creatures' power runs card effects.
func (m *Match) BroadcastState() {
	// An attacker taps once its attack is beyond cancelling, before blocks,
	// battles and shield breaks follow, which is when it is seen to act.
	if attack := m.resolvingAttack; attack != nil && !attack.announced {
		if attacker, err := m.CurrentPlayer().Player.GetCard(attack.cardID, BATTLEZONE); err == nil && attacker.Tapped {
			attack.announced = true
			m.noteActivated(attack.cardID)
		}
	}

	brokenShields := m.brokenShields
	m.brokenShields = nil
	activatedCards := m.activatedCards
	m.activatedCards = nil

	report := m.knownBoard()

	// At the end of a turn every card's conditions are cleared, and they are
	// only rebuilt in the next turn's untap step. Power and statuses worked out
	// in between would dip and recover for no reason, so the ones from before
	// are sent.
	switch m.Step.(type) {
	case *EndOfTurnStep, *BeginTurnStep:
	default:
		m.workOutCreatures(&report)
	}

	report.attack = m.workOutAttack()
	report.turn = m.turnNumber
	m.keepBoard(report)

	m.broadcastState(report, brokenShields, activatedCards)
}

// sentStates are the states last sent to each viewer.
type sentStates struct {
	player1, player2, spectator *server.MatchStateMessage
}

// keepStates keeps the states just sent, for sendLatestState. Their broken
// shields and acting cards are left out: those were news once, and showing
// them again to someone arriving would replay what is over.
func (m *Match) keepStates(player1, player2, spectator *server.MatchStateMessage) {
	replay := func(state *server.MatchStateMessage) *server.MatchStateMessage {
		kept := *state
		kept.State.BrokenShields = []string{}
		kept.State.ActivatedCards = []string{}
		return &kept
	}

	m.statesMu.Lock()
	defer m.statesMu.Unlock()
	m.lastStates = sentStates{player1: replay(player1), player2: replay(player2), spectator: replay(spectator)}
}

// sendLatestState sends someone joining or coming back the state last sent
// to viewers like them: the host's, the guest's or a spectator's. It reads
// nothing of the live game, so it is safe beside the event loop; before the
// first state there is nothing to send.
func (m *Match) sendLatestState(s *server.Socket) {
	m.statesMu.Lock()
	state := m.lastStates.spectator
	switch s.User.UID {
	case m.HostID:
		state = m.lastStates.player1
	case m.GuestID:
		state = m.lastStates.player2
	}
	m.statesMu.Unlock()

	if state != nil {
		s.Send(state)
	}
}

// broadcastKnownState is BroadcastState for code running beside the event loop,
// such as a player joining. It sends the board as the event loop last worked it
// out, rather than running card effects or reading the attack from another
// goroutine, and
// leaves broken shields and acting cards for the event loop's next broadcast
// to report.
func (m *Match) broadcastKnownState() {
	m.broadcastState(m.knownBoard(), nil, nil)
}

// noteActivated records that a card acted, for the next state to report.
func (m *Match) noteActivated(cardID string) {
	m.noteRecent(cardID)

	for _, noted := range m.activatedCards {
		if noted == cardID {
			return
		}
	}

	m.activatedCards = append(m.activatedCards, cardID)
}

// noteEffectSource records the card whose effect moved another card, when the
// source is a card in a zone both players can see. A source can also name the
// moved card itself, as summoning does, or a cause that is not a card at all,
// such as a draw, and neither is an effect acting.
func (m *Match) noteEffectSource(source string, movedCardID string) {
	if source == "" || source == movedCardID {
		return
	}

	for _, p := range []*PlayerReference{m.Player1, m.Player2} {
		for _, zone := range []string{BATTLEZONE, SPELLZONE, MANAZONE, GRAVEYARD} {
			if _, err := p.Player.GetCard(source, zone); err == nil {
				m.noteActivated(source)
				return
			}
		}
	}
}

func (m *Match) broadcastState(board boardReport, brokenShields []string, activatedCards []string) {

	if brokenShields == nil {
		brokenShields = make([]string, 0)
	}

	if activatedCards == nil {
		activatedCards = make([]string, 0)
	}

	player1 := *m.Player1.Player.Denormalized()
	player1.Username = m.Player1.Username
	player1.Color = m.Player1.Color
	player1.Powers = ownedBy(player1.Battlezone, board.powers)
	player1.Statuses = ownedBy(player1.Battlezone, board.statuses)

	player2 := *m.Player2.Player.Denormalized()
	player2.Username = m.Player2.Username
	player2.Color = m.Player2.Color
	player2.Powers = ownedBy(player2.Battlezone, board.powers)
	player2.Statuses = ownedBy(player2.Battlezone, board.statuses)

	_, isAttackStep := m.Step.(*AttackStep)
	step := StepName(m.Step)

	// Step only becomes the attack step once an attack has resolved, but blocks,
	// shield triggers and their prompts come first, and players need to see that
	// the turn has moved on. Should the attack be prevented after all, the state
	// broadcast once it has resolved reports the step the turn stayed in.
	if board.attack.resolving {
		step = StepName(&AttackStep{})
	}

	p1state := &server.MatchStateMessage{
		Header: "state_update",
		State: server.MatchState{
			MyTurn:        m.Turn == 1,
			HasAddedMana:  m.Player1.Player.HasChargedMana,
			HasAttacked:   m.Turn == 1 && isAttackStep,
			CanChargeMana: m.Player1.Player.CanChargeMana,
			Step:          step,
			BrokenShields: brokenShields,
			Me:            player1,
			Opponent:      player2,
		},
	}

	p2state := &server.MatchStateMessage{
		Header: "state_update",
		State: server.MatchState{
			MyTurn:        m.Turn == 2,
			HasAddedMana:  m.Player2.Player.HasChargedMana,
			HasAttacked:   m.Turn == 2 && isAttackStep,
			CanChargeMana: m.Player2.Player.CanChargeMana,
			Step:          step,
			BrokenShields: brokenShields,
			Me:            player2,
			Opponent:      player1,
		},
	}

	spectatorState := &server.MatchStateMessage{
		Header: "state_update",
		State: server.MatchState{
			MyTurn:        false,
			HasAddedMana:  false,
			HasAttacked:   false,
			CanChargeMana: false,
			Step:          step,
			BrokenShields: brokenShields,
			Me:            player1,
			Opponent:      player2,
			Spectator:     true,
		},
	}

	p1state.State.Opponent.Hand = make([]server.CardState, 0)
	p2state.State.Opponent.Hand = make([]server.CardState, 0)
	spectatorState.State.Me.Hand = make([]server.CardState, 0)
	spectatorState.State.Opponent.Hand = make([]server.CardState, 0)

	turnManaCharged := m.CurrentPlayer().Player.HasChargedMana

	for _, state := range []*server.MatchStateMessage{p1state, p2state, spectatorState} {
		state.State.ActivatedCards = activatedCards
		state.State.TurnManaCharged = turnManaCharged
		state.State.Attacker = board.attack.attacker
		state.State.Turn = board.turn
	}

	m.keepStates(p1state, p2state, spectatorState)

	m.Player1.Socket.Send(p1state)
	m.Player2.Socket.Send(p2state)

	// Iter returns a copy, so the spectators are not kept locked while sending.
	// Spectators joining or leaving would otherwise wait on every slow socket.
	for _, spectator := range m.spectators.Iter() {
		if spectator.Socket == nil {
			continue
		}
		spectator.Socket.Send(spectatorState)
	}

}

// Warn sends a warning to the specified player ref
func Warn(p *PlayerReference, message string) {

	p.Socket.Send(server.WarningMessage{
		Header:  "warn",
		Message: message,
	})

}

// WarnError sends an error message to the specified player ref
func WarnError(p *PlayerReference, message string) {

	p.Socket.Send(server.WarningMessage{
		Header:  "error",
		Message: message,
	})

}

// WarnPlayer sends a warning to the specified player
func (m *Match) WarnPlayer(p *Player, message string) {

	m.PlayerRef(p).Socket.Send(server.WarningMessage{
		Header:  "warn",
		Message: message,
	})

}

// ActionWarning adds an error message to the players current action popup
func (m *Match) ActionWarning(p *Player, message string) {
	m.PlayerRef(p).Socket.Send(server.ActionWarningMessage{
		Header:  "action_error",
		Message: message,
	})
}

// DefaultActionWarning sends an action warning with a predefined message
func (m *Match) DefaultActionWarning(p *Player) {
	m.PlayerRef(p).Socket.Send(server.ActionWarningMessage{
		Header:  "action_error",
		Message: "Your selection of cards does not fulfill the requirements",
	})
}

// HandleFx ...
func (m *Match) HandleFx(ctx *Context) {
	if m == nil || m.IsEnding() {
		return
	}

	if m.Player1 == nil || m.Player1.Player == nil || m.Player2 == nil || m.Player2.Player == nil {
		return
	}

	players := make([]*PlayerReference, 0)

	// The player in which turn it is is to be handled first
	if m.Turn == m.Player1.Player.Turn {
		players = append(players, m.Player1, m.Player2)
	} else {
		players = append(players, m.Player2, m.Player1)
	}

	cards := make([]*Card, 0)

	for _, p := range players {

		cards = append(cards, p.Player.battlezone...)
		cards = append(cards, p.Player.spellzone...)
		cards = append(cards, p.Player.hand...)
		cards = append(cards, p.Player.shieldzone...)
		cards = append(cards, p.Player.hiddenzone...)
		cards = append(cards, p.Player.manazone...)
		cards = append(cards, p.Player.graveyard...)
		cards = append(cards, p.Player.deck...)

	}

	// Handle persistent effects
	for _, fx := range m.persistentEffects {
		if m.IsEnding() {
			return
		}
		fx.effect(ctx, fx.exit)
	}

	// Handle regular card effects c.Use(...
	for _, card := range cards {

		for _, h := range card.handlers {

			if ctx.cancel || m.IsEnding() {
				return
			}

			h(card, ctx)

		}

	}

	// Handle ctx.ScheduleAfter effects
	for _, h := range ctx.postFxs {

		if ctx.cancel || m.IsEnding() {
			return
		}

		h()

	}

	// End the game  if any of the players are out of cards in their deck
	for _, p := range players {
		if len(p.Player.deck) < 1 {
			m.End(m.Opponent(p.Player), fmt.Sprintf("%s won by deck out!", m.Opponent(p.Player).Username()))
		}
	}

}

// NewAction prompts the user to make a selection of the specified []Cards
func (m *Match) NewAction(player *Player, cards []*Card, minSelections int, maxSelections int, text string, cancellable bool) {
	m.NewActionFullList(player, cards, minSelections, maxSelections, text, cancellable, nil)
}

func (m *Match) NewActionFullList(player *Player, cards []*Card, minSelections int, maxSelections int, text string, cancellable bool, unselectableCards []*Card) {

	msg := &server.ActionMessage{
		Header:            "action",
		Cards:             denormalizeCards(cards, false),
		Text:              text,
		MinSelections:     minSelections,
		MaxSelections:     maxSelections,
		Cancellable:       cancellable,
		UnselectableCards: denormalizeCards(unselectableCards, false),
	}

	m.ask(player, msg)

}

// NewBacksideAction prompts the user to make a selection of the specified cards without their names or images
func (m *Match) NewBacksideAction(player *Player, cards []*Card, minSelections int, maxSelections int, text string, cancellable bool) {

	msg := &server.ActionMessage{
		Header:        "action",
		Cards:         denormalizeCards(cards, true),
		Text:          text,
		MinSelections: minSelections,
		MaxSelections: maxSelections,
		Cancellable:   cancellable,
	}

	m.ask(player, msg)

}

// NewMultipartAction prompts the user to make a selection of the specified {string: []Cards}
func (m *Match) NewMultipartAction(player *Player, cards map[string][]*Card, minSelections int, maxSelections int, text string, cancellable bool) {
	m.newMultipartActionBase(player, cards, minSelections, maxSelections, text, cancellable, false)
}

// NewMultipartAction prompts the user to make a selection of the specified {string: []Cards}
func (m *Match) NewMultipartActionBackside(player *Player, cards map[string][]*Card, minSelections int, maxSelections int, text string, cancellable bool) {
	m.newMultipartActionBase(player, cards, minSelections, maxSelections, text, cancellable, true)
}

func (m *Match) newMultipartActionBase(player *Player, cards map[string][]*Card, minSelections int, maxSelections int, text string, cancellable bool, backsideOnly bool) {

	cardMap := make(map[string][]server.CardState)

	for key, cards := range cards {
		cardMap[key] = denormalizeCards(cards, backsideOnly)
	}

	msg := &server.MultipartActionMessage{
		Header:        "action",
		Cards:         cardMap,
		Text:          text,
		MinSelections: minSelections,
		MaxSelections: maxSelections,
		Cancellable:   cancellable,
	}

	m.ask(player, msg)

}

// NewCountAction prompts the user to select a number in an interval
func (m *Match) NewCountAction(player *Player, text string, minSelections int, maxSelections int) {

	msg := &server.ActionMessage{
		Header:        "action",
		ActionType:    "count",
		Text:          text,
		MinSelections: minSelections,
		MaxSelections: maxSelections,
	}

	m.ask(player, msg)

}

// NewQuestionAction prompts the user to answer a yes/no question
func (m *Match) NewQuestionAction(player *Player, text string) {

	msg := &server.ActionMessage{
		Header:     "action",
		ActionType: "question",
		Text:       text,
	}

	m.ask(player, msg)

}

func (m *Match) NewOrderAction(player *Player, cards []*Card, text string) {

	msg := &server.ActionMessage{
		Header:     "action",
		ActionType: "order",
		Cards:      denormalizeCards(cards, false),
		Text:       text,
	}

	m.ask(player, msg)

}

// NewMultipleChoiceQuestionAction prompts the user to answer a yes/no question
func (m *Match) NewMultipleChoiceQuestionAction(player *Player, text string, choices []string) {

	msg := &server.ActionMessage{
		Header:     "action",
		ActionType: "question",
		Text:       text,
		Choices:    choices,
	}

	m.ask(player, msg)

}

// NewMultipleChoiceSearchableAction prompts the user to select an option
// from a multiple choice selection with a search and scrollable bar
func (m *Match) NewMultipleChoiceSearchableAction(player *Player, text string, choices []string) {

	msg := &server.ActionMessage{
		Header:     "action",
		ActionType: "searchable",
		Text:       text,
		Choices:    choices,
	}

	m.ask(player, msg)

}

// ask puts a question to a player: it opens their prompt, numbering msg, and
// sends it to them.
func (m *Match) ask(player *Player, msg server.Prompting) {
	player.prompt.ask(msg)
	m.PlayerRef(player).Socket.Send(msg)
}

// CloseAction closes the card selection popup for the given player
func (m *Match) CloseAction(p *Player) {
	p.prompt.close()
	m.PlayerRef(p).Socket.Send(server.Message{
		Header: "close_action",
	})
}

// Wait sends a waiting popup with a message to the specified player
func (m *Match) Wait(p *Player, message string) {
	m.PlayerRef(p).Socket.Send(server.WaitMessage{
		Header:  "wait",
		Message: message,
	})
}

// EndWait closes the waiting popup for the specified player
func (m *Match) EndWait(p *Player) {
	m.PlayerRef(p).Socket.Send(server.Message{
		Header: "end_wait",
	})
}

// ShowCards shows the specified cards to the player with a message of why it is being shown
func (m *Match) ShowCards(p *Player, message string, cards []string) {
	m.PlayerRef(p).Socket.Send(server.ShowCardsMessage{
		Header:  "show_cards",
		Message: message,
		Cards:   cards,
	})
}

// ShowCardsNonDismissible shows the specified cards to the player with a message of why it is being shown
// The difference is that the cards preview pop-up is not dismissible by clicking outside of it,
// Because is part of an Attacking Player sequence of events and it must be only cancelled by clicking on the 'Close' button
func (m *Match) ShowCardsNonDismissible(p *Player, message string, cards []string) {
	msg := &server.ShowCardsMessage{
		Header:  "show_cards_non_dismissible",
		Message: message,
		Cards:   cards,
	}

	m.ask(p, msg)

	defer m.CloseAction(p)

	for {
		action := p.NextAction()

		if action.Cancel {
			break
		}
	}
}

// Coin toss to decide who starts the game
func (m *Match) CoinToss() {
	var PlayerToChooseToss *PlayerReference
	var PlayerToAwaitToss *PlayerReference

	if m.PlayerSelectingToss == "" {
		if rand.Intn(100) >= 50 {
			m.PlayerSelectingToss = m.Player2.UID

			PlayerToChooseToss = m.Player2
			PlayerToAwaitToss = m.Player1
		} else {
			m.PlayerSelectingToss = m.Player1.UID

			PlayerToChooseToss = m.Player1
			PlayerToAwaitToss = m.Player2
		}
	}

	m.Chat("Server", fmt.Sprintf("Coin toss to be selected by %s", PlayerToChooseToss.Username))

	PlayerToAwaitToss.Socket.Send(server.Message{
		Header: "toss_being_chosen",
	})

	PlayerToChooseToss.Socket.Send(server.Message{
		Header: "choose_toss",
	})
}

// Start starts the match
func (m *Match) Start() {

	m.startedAt = time.Now()
	m.started.Store(true)

	m.Player1.Player.ShuffleDeck()
	m.Player2.Player.ShuffleDeck()

	m.Player1.Player.InitShieldzone()
	m.Player2.Player.InitShieldzone()

	m.Player1.Player.DrawCards(5)
	m.Player2.Player.DrawCards(5)

	m.Chat("Server", "The duel has begun!")

	m.BeginNewTurn()

}

// BeginNewTurn starts a new turn
func (m *Match) BeginNewTurn(repeatTurn ...bool) {

	m.Step = &BeginTurnStep{}
	m.turnNumber++

	if len(repeatTurn) == 0 || !repeatTurn[0] {
		m.turnsPlayed++

		if m.Turn == 1 {
			m.Turn = 2
		} else {
			m.Turn = 1
		}
	}

	ctx := NewContext(m, m.Step)

	m.HandleFx(ctx)

	m.CurrentPlayer().Player.HasChargedMana = false
	m.CurrentPlayer().Player.CanChargeMana = true

	m.BroadcastState()

	m.UntapStep()

}

// UntapStep ...
func (m *Match) UntapStep() {

	m.Step = &UntapStep{}

	currentPlayer := m.CurrentPlayer().Player
	manaUntapContext := NewContext(m, &UntapManaEvent{Player: currentPlayer})
	m.HandleFx(manaUntapContext)

	if !manaUntapContext.Cancelled() {
		if mana, err := currentPlayer.Container(MANAZONE); err == nil {
			for _, c := range mana {
				c.Tapped = false
			}
		}
	}

	ctx := NewContext(m, m.Step)

	m.HandleFx(ctx)

	m.StartOfTurnStep()

}

// StartOfTurnStep ...
func (m *Match) StartOfTurnStep() {

	m.Step = &StartOfTurnStep{}

	ctx := NewContext(m, m.Step)

	m.HandleFx(ctx)

	m.ReportActionInChat(m.CurrentPlayer().Player, fmt.Sprintf("Your turn, %s", m.CurrentPlayer().Socket.User.Username))

	m.DrawStep()

}

// DrawStep ...
func (m *Match) DrawStep() {

	m.Step = &DrawStep{}

	ctx := NewContext(m, m.Step)

	m.HandleFx(ctx)

	if m.isFirstTurn == false {
		m.CurrentPlayer().Player.DrawCards(1)
	}

	m.BroadcastState()

	m.ChargeStep()

}

// ChargeStep ...
func (m *Match) ChargeStep() {

	m.Step = &ChargeStep{}

	ctx := NewContext(m, m.Step)

	m.HandleFx(ctx)

	// The turn waits here for the player, so the duel interface has to learn
	// that it reached this step. The draw step's broadcast came before it.
	m.BroadcastState()

}

// EndStep ...
func (m *Match) EndStep() {

	m.Step = &EndStep{}

	ctx := NewContext(m, m.Step)

	m.HandleFx(ctx)

	m.ReportActionInChat(m.CurrentPlayer().Player, fmt.Sprintf("%s ended their turn", m.CurrentPlayer().Socket.User.Username))

	m.EndOfTurnTriggers()

}

// EndOfTurnTriggers ...
func (m *Match) EndOfTurnTriggers() {
	m.Step = &EndOfTurnStep{}
	m.isFirstTurn = false
	ctx := NewContext(m, m.Step)
	m.HandleFx(ctx)

	if !ctx.Cancelled() {
		m.BeginNewTurn()
	}
}

// EndTurn is called when the player attempts to end their turn
// If the context is not cancelled by a card, the EndStep is called
func (m *Match) EndTurn() {

	ctx := NewContext(m, &EndTurnEvent{})

	m.HandleFx(ctx)

	if !ctx.cancel {
		m.EndStep()
	}

}

// ChargeMana is called when the player attempts to charge mana
func (m *Match) ChargeMana(p *PlayerReference, cardID string) {

	if p.Player.HasChargedMana {
		Warn(p, "You have already charged mana this round")
		return
	}

	if !p.Player.CanChargeMana {
		Warn(p, "You can't charge mana after playing or attacking with creatures/spells or using tap ability.")
		return
	}

	if card, err := p.Player.MoveCard(cardID, HAND, MANAZONE, cardID); err == nil {
		p.Player.HasChargedMana = true
		m.BroadcastState()
		m.ReportActionInChat(p.Player, fmt.Sprintf("%s was added to %s's manazone", card.Name, p.Socket.User.Username))
	}

}

// PlayCard is called when the player attempts to play a card
func (m *Match) PlayCard(p *PlayerReference, cardID string) {

	ctx := NewContext(m, &PlayCardEvent{
		CardID: cardID,
	})

	m.HandleFx(ctx)

	if !ctx.Cancelled() {
		if _, ok := m.Step.(*MainStep); !ok {
			m.Step = &MainStep{}
		}

		p.Player.CanChargeMana = false

		if !p.Player.HasCard(HAND, cardID) {
			m.noteActivated(cardID)
		}
	}

	m.BroadcastState()

}

// AttackPlayer is called when the player attempts to attack the opposing player
func (m *Match) AttackPlayer(p *PlayerReference, cardID string) {

	_, err := p.Player.GetCard(cardID, BATTLEZONE)

	if err != nil {
		Warn(p, "The creature you tried to attack with is not in the battlezone")
		return
	}

	event := &AttackPlayer{
		CardID: cardID,
	}

	m.resolveAttack(cardID, event)

	if event.Confirmed {
		m.Step = &AttackStep{}
		p.Player.CanChargeMana = false
	}

	m.BroadcastState()

}

// AttackCreature is called when the player attempts to attack an opposing creature
func (m *Match) AttackCreature(p *PlayerReference, cardID string) {

	_, err := p.Player.GetCard(cardID, BATTLEZONE)

	if err != nil {
		Warn(p, "The creature you tried to attack with is not in the battlezone")
		return
	}

	event := &AttackCreature{
		CardID:              cardID,
		AttackableCreatures: make([]*Card, 0),
	}

	m.resolveAttack(cardID, event)

	if event.Confirmed {
		m.Step = &AttackStep{}
		p.Player.CanChargeMana = false
	}

	m.BroadcastState()

}

func (m *Match) TapAbility(p *PlayerReference, cardID string) {
	_, err := p.Player.GetCard(cardID, BATTLEZONE)

	if err != nil {
		Warn(p, "The creature you tried to use a tap ability for is not in the battlezone")
		return
	}

	event := &TapAbility{
		CardID: cardID,
	}

	m.resolveAttack(cardID, event)

	if event.Confirmed {
		// Tap abilities can only be used during the attack step
		// https://duelmasters.fandom.com/wiki/Step#Step_7_(Attack_step)
		m.Step = &AttackStep{}
		p.Player.CanChargeMana = false
	}

	m.BroadcastState()
}

// attackInProgress is an attack or tap ability the event loop is resolving.
type attackInProgress struct {
	cardID string
	event  interface{}
	// attacks is set for an attack, and not for a tap ability.
	attacks bool
	// announced is set once the creature has been reported as acting.
	announced bool
}

// resolveAttack resolves an attack or tap ability by the given creature,
// letting BroadcastState know until it has finished. Only the event loop
// calls this.
func (m *Match) resolveAttack(cardID string, event interface{}) {
	_, tapAbility := event.(*TapAbility)
	attack := &attackInProgress{cardID: cardID, event: event, attacks: !tapAbility}
	m.resolvingAttack = attack
	defer func() { m.resolvingAttack = nil }()

	m.HandleFx(NewContext(m, event))

	// A creature that went ahead without tapping is reported now instead.
	if !attack.announced && attackWentAhead(attack.event) {
		m.noteActivated(cardID)
	}
}

func attackWentAhead(event interface{}) bool {
	switch e := event.(type) {
	case *AttackPlayer:
		return e.Confirmed
	case *AttackCreature:
		return e.Confirmed
	case *TapAbility:
		return e.Confirmed
	default:
		return false
	}
}

// Parse handles websocket messages in this Hub
func (m *Match) Parse(s *server.Socket, data []byte) {

	defer internal.Recover()

	var message server.Message
	if err := json.Unmarshal(data, &message); err != nil {
		return
	}

	switch message.Header {

	case "mpong":
		m.eventloop.schedule(func() {
			p, err := m.PlayerForSocket(s)

			if err != nil {
				return
			}

			p.LastPong = time.Now().Unix()
		}, ParallelEvent)

	case "join_match":
		// Parallel, so a player coming back is seated even while the event loop
		// waits on a prompt; join guards the seats itself.
		m.eventloop.schedule(func() {
			m.join(s)
		}, ParallelEvent)

	case "chat":
		m.eventloop.schedule(func() {

			// Allow other sockets than player1 and player2 to chat?

			var msg struct {
				Message string `json:"message"`
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}

			if m.chatRefusal != nil {
				if refusal := m.chatRefusal(s.User.UID); refusal != "" {
					s.Send(&server.ChatMessage{
						Header:  "chat",
						Message: refusal,
						Sender:  "Server",
						Color:   "#ccc",
					})
					return
				}
			}

			m.handleAdminMessages(msg.Message, s.User)

			if s.User.Chatblocked {
				s.Send(&server.ChatMessage{
					Header:  "chat",
					Message: msg.Message,
					Sender:  s.User.Username,
					Color:   "#ccc",
				})
			} else {
				m.ColorChat(s.User.Username, msg.Message, s.User.Color)
			}
		}, ParallelEvent)

	case "toss_chosen":
		m.eventloop.schedule(func() {
			var msg struct {
				Prediction int `json:"prediction"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}
			m.callToss(s, msg.Prediction)
		}, SequentialEvent)

	case "turn_chosen":
		m.eventloop.schedule(func() {
			var msg struct {
				Player int `json:"player"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}
			m.chooseTurn(s, msg.Player)
		}, SequentialEvent)

	case "add_to_manazone":
		m.eventloop.schedule(func() {

			p, err := m.PlayerForSocket(s)

			if err != nil {
				return
			}

			if m.Turn != p.Player.Turn {
				return
			}

			var msg struct {
				ID string `json:"virtualId"`
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}

			m.ChargeMana(p, msg.ID)

		}, SequentialEvent)

	case "end_turn":
		m.eventloop.schedule(func() {

			p, err := m.PlayerForSocket(s)

			if err != nil {
				return
			}

			if m.Turn != p.Player.Turn {
				return
			}

			m.EndTurn()

		}, SequentialEvent)

	case "add_to_playzone":
		m.eventloop.schedule(func() {

			p, err := m.PlayerForSocket(s)

			if err != nil {
				return
			}

			if m.Turn != p.Player.Turn {
				return
			}

			var msg struct {
				ID string `json:"virtualId"`
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}

			m.PlayCard(p, msg.ID)

		}, SequentialEvent)

	case "action":
		// Answered at once rather than scheduled: the event loop is waiting
		// for the answer, and giving it never blocks.
		m.answer(s, data)

	case "attack_player":
		m.eventloop.schedule(func() {

			p, err := m.PlayerForSocket(s)

			if err != nil {
				return
			}

			if m.Turn != p.Player.Turn {
				return
			}

			var msg struct {
				ID string `json:"virtualId"`
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}

			m.AttackPlayer(p, msg.ID)

		}, SequentialEvent)

	case "attack_creature":
		m.eventloop.schedule(func() {

			p, err := m.PlayerForSocket(s)

			if err != nil {
				return
			}

			if m.Turn != p.Player.Turn {
				return
			}

			var msg struct {
				ID string `json:"virtualId"`
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}

			m.AttackCreature(p, msg.ID)

		}, SequentialEvent)

	case "tap_ability":
		m.eventloop.schedule(func() {

			p, err := m.PlayerForSocket(s)

			if err != nil {
				return
			}

			if m.Turn != p.Player.Turn {
				return
			}

			var msg struct {
				ID string `json:"virtualId"`
			}

			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}

			m.TapAbility(p, msg.ID)

		}, SequentialEvent)

	case "resign":
		m.eventloop.schedule(func() {
			p, err := m.PlayerForSocket(s)

			if err != nil {
				return
			}

			m.end(m.Opponent(p.Player), fmt.Sprintf("%s won: %s surrendered.", m.Opponent(p.Player).Username(), p.Username), EndingSurrender)

		}, ParallelEvent)

	default:
		logrus.Debugf("Received message in incorrect format: %v", string(data))

	}

}

// OnSocketClose is called when a socket disconnects
func (m *Match) OnSocketClose(s *server.Socket) {

	if m.IsEnding() {
		return
	}

	// is this a spectator leaving?
	if spectator, ok := m.spectators.Find(s.User.UID); ok && spectator.Socket == s {
		m.Chat("Server", fmt.Sprintf("%s stopped spectating", spectator.Username))
		m.spectators.Remove(spectator.UID)
		return
	}

	m.leave(s)

}

func (p *PlayerReference) Dispose() {
	defer internal.Recover()

	if p.Player != nil {
		p.Player.Dispose()
	}

	if p.Socket != nil {
		p.Socket.Close()
	}
}

func (m *Match) handleAdminMessages(message string, user server.User) {

	if !hasAdminRightsAndValidMsgFormat(message, user, m.system.devCommands) {
		return
	}

	handleAdminCommandCases(m, message)
}

// hasAdminRightsAndValidMsgFormat reports whether a chat message is a dev
// command to run: only when the match system allows dev commands.
func hasAdminRightsAndValidMsgFormat(message string, user server.User, devCommands bool) bool {

	/* hasRights := false

	for _, permission := range user.Permissions {
		if permission == "admin" {
			hasRights = true
			break
		}
	}

	if !hasRights {
		return false
	} */

	if !devCommands {
		return false
	}

	msgParts := strings.Split(message, " ")

	return len(msgParts) >= 2

}

func handleAdminCommandCases(m *Match, message string) {

	var cardsToAdd []string
	currentPlayer := m.CurrentPlayer().Player
	msgParts := strings.Split(message, " ")

	switch msgParts[0] {

	// /init all 5 - initialize all zones (hand, shield, mana, deck, graveyard)
	//					with 5 cards from each civ (25 in total) for each zone
	case "/init":

		cardsToAdd = getCardsToAddFromGivenCommand(currentPlayer, msgParts[1])

		spawnCardsToGivenZones(m, currentPlayer, cardsToAdd, []string{HAND, DECK, SHIELDZONE, MANAZONE, GRAVEYARD}, msgParts)

	// /add red 4 - add 4 fire cards to your hand
	// /add n - add 1 nature card to your hand
	// /add imageID 3 - add 3 copies of that specific card to your hand
	// /add all 5 - add 5 cards from each civ to your hand (25 in total)
	case "/add":

		cardsToAdd = getCardsToAddFromGivenCommand(currentPlayer, msgParts[1])

		spawnCardsToGivenZones(m, currentPlayer, cardsToAdd, []string{HAND}, msgParts)

	// /mana hand - will add all the cards in your hand to the manazone
	// /mana red 4 - add 4 fire mana
	// /mana n - add 1 nature mana
	// /mana imageID 3 - add 3 mana of that specific card
	// /mana all 5 - add 5 mana from each civ to your hand (25 in total)
	case "/mana":

		cardsToAdd = getCardsToAddFromGivenCommand(currentPlayer, msgParts[1])

		spawnCardsToGivenZones(m, currentPlayer, cardsToAdd, []string{MANAZONE}, msgParts)

		return

	// /shield hand - will add all the cards in your hand to the shieldzone
	// /shield red 4 - add 4 fire shields
	// /shield n - add 1 nature shield
	// /shield imageID 3 - add 3 shields of that specific card
	// /shield all 5 - add 5 shields from each civ (25 in total)
	case "/shield":

		cardsToAdd = getCardsToAddFromGivenCommand(currentPlayer, msgParts[1])

		spawnCardsToGivenZones(m, currentPlayer, cardsToAdd, []string{SHIELDZONE}, msgParts)

		return

	// /deck hand - will add all the cards in your hand to the deck
	// /deck red 4 - add 4 fire cards to the deck
	// /deck n - add 1 nature card to the deck
	// /deck imageID 3 - add 3 copies of that specific card to the deck
	// /deck all 5 - add 5 cards from each civ to the deck (25 in total)
	case "/deck":

		cardsToAdd = getCardsToAddFromGivenCommand(currentPlayer, msgParts[1])

		spawnCardsToGivenZones(m, currentPlayer, cardsToAdd, []string{DECK}, msgParts)

		return

	// /grave hand - will add all the cards in your hand to the graveyard
	// /grave red 4 - add 4 fire cards to your graveyard
	// /grave n - add 1 nature card to your graveyard
	// /grave imageID 3 - add 3 copies of that specific card to your graveyard
	// /grave all 5 - add 5 cards from each civ to your graveyard (25 in total)
	case "/grave":

		cardsToAdd = getCardsToAddFromGivenCommand(currentPlayer, msgParts[1])

		spawnCardsToGivenZones(m, currentPlayer, cardsToAdd, []string{GRAVEYARD}, msgParts)

		return

	}

}

func getCardsToAddFromGivenCommand(crtPlayer *Player, command string) []string {

	var cardsToAdd []string

	switch command {

	case "hand":
		for _, c := range crtPlayer.hand {
			cardsToAdd = append(cardsToAdd, c.ID)
		}
	case "fire", "f", "red":
		cardsToAdd = []string{"af3bc221-1cc2-4f58-83ea-2673ac2c66c5"}
	case "water", "w", "blue":
		cardsToAdd = []string{"9781089f-1aa9-4a75-b106-35e9d431e31d"}
	case "light", "l", "yellow":
		cardsToAdd = []string{"7b58e8c2-0b1e-4ef5-812f-e667c2092c73"}
	case "darkness", "d", "black":
		cardsToAdd = []string{"e2b992ee-91a3-49d3-8228-7be60a0b9ec5"}
	case "nature", "n", "green":
		cardsToAdd = []string{"1d72eb3e-5185-449a-a16f-391bd2338343"}
	case "all", "fromAll", "fromall":
		cardsToAdd =
			[]string{
				"af3bc221-1cc2-4f58-83ea-2673ac2c66c5",
				"9781089f-1aa9-4a75-b106-35e9d431e31d",
				"7b58e8c2-0b1e-4ef5-812f-e667c2092c73",
				"e2b992ee-91a3-49d3-8228-7be60a0b9ec5",
				"1d72eb3e-5185-449a-a16f-391bd2338343",
			}
	default:
		cardsToAdd = []string{command}
	}

	return cardsToAdd

}

func spawnCardsToGivenZones(m *Match,
	player *Player,
	cardsToSpawn []string,
	zones []string,
	commands []string) {

	if len(cardsToSpawn) > 0 {
		count := 1
		if len(commands) > 2 && commands[1] != "hand" {
			number, err := strconv.Atoi(commands[2])
			if err == nil && number >= 1 {
				count = number
			}
		}

		ctx := NewContext(m, &UntapStep{})

		for _, z := range zones {
			for i := range cardsToSpawn {
				for range count {
					card, err := player.SpawnCard(cardsToSpawn[i], z)
					if err != nil {
						logrus.Warnf("Failed to spawn card %s: %v", cardsToSpawn[i], err)
						continue
					}
					// Run this card's own handlers with UntapStep so conditions like cnd.Creature and cnd.Spell are initialised, matching real game behaviour
					for _, h := range card.handlers {
						h(card, ctx)
					}
				}
			}
		}
	}

	// Admin commands arrive as chat, which runs beside the event loop.
	m.broadcastKnownState()

}
