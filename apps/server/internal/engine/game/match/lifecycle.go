package match

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// The coin toss's sides, as TossPrediction and TossOutcome hold them and the
// toss_chosen message sends the call.
const (
	heads = 1
	tails = -1
)

// The turn_chosen message's choices.
const (
	playFirst  = 1
	playSecond = -1
)

// hasStarted reports whether the coin toss is settled and the game has begun.
func (m *Match) hasStarted() bool {
	return m.started.Load()
}

// Connected reports whether the player has joined the match and is connected
// now, rather than not yet arrived or gone and not back.
func (m *Match) Connected(uid string) bool {
	m.setupMu.Lock()
	defer m.setupMu.Unlock()

	for _, player := range []*PlayerReference{m.Player1, m.Player2} {
		if player != nil && player.UID == uid {
			return player.Socket != nil && !player.Socket.IsClosed()
		}
	}

	return false
}

// join seats a player arriving on a socket, or gives a player coming back
// their seat on the new socket, and adds anyone else as a spectator. The coin
// toss begins once both players have joined.
func (m *Match) join(s *server.Socket) {
	if m.IsEnding() {
		s.Send(server.WarningMessage{Header: "error", Message: "This game has ended."})
		s.Close()
		return
	}

	uid := s.User.UID

	m.setupMu.Lock()
	var seat, other **PlayerReference
	var turn byte
	switch uid {
	case m.HostID:
		seat, other, turn = &m.Player1, &m.Player2, 1
	case m.GuestID:
		seat, other, turn = &m.Player2, &m.Player1, 2
	}
	if seat == nil {
		m.setupMu.Unlock()
		m.addSpectator(s)
		return
	}

	var replaced *server.Socket
	rejoining := *seat != nil
	if rejoining {
		replaced = (*seat).Socket
		(*seat).Socket = s
		m.stopDisconnectTimer(uid)
	} else {
		*seat = NewPlayerReference(NewPlayer(m, turn), s)
	}
	opponent := *other
	beginToss := !m.tossBegun && m.Player1 != nil && m.Player2 != nil
	if beginToss {
		m.tossBegun = true
	}
	m.setupMu.Unlock()

	// The old socket is no longer a player's, so closing it changes nothing.
	if replaced != nil && replaced != s {
		replaced.Close()
	}

	if rejoining {
		if opponent != nil && opponent.Socket != nil {
			opponent.Socket.Send(server.Message{Header: "opponent_reconnected"})
		}
		m.Chat("Server", s.User.Username+" reconnected")

		if m.hasStarted() {
			m.sendLatestState(s)
			// Ask again for an answer still owed.
			if player := (*seat).Player; !player.ActionState.resolved {
				s.Send(player.ActionState.data)
			}
		} else {
			m.sendTossStep(s)
		}
		return
	}

	if beginToss {
		m.beginToss()
	}
}

// addSpectator lets someone who is not playing watch the match, replacing a
// connection they already watch it on.
func (m *Match) addSpectator(s *server.Socket) {
	if spectator, ok := m.spectators.Find(s.User.UID); ok {
		spectator.Socket.Send(server.WarningMessage{
			Header:  "error",
			Message: "You started spectating from a new connection, closing this one...",
		})
		// This removes the existing spectator from the match.
		spectator.Socket.Close()
	}

	m.spectators.Add(s.User.UID, &Spectator{
		UID:      s.User.UID,
		Username: s.User.Username,
		Color:    s.User.Color,
		Socket:   s,
		LastPong: time.Now().Unix(),
	})

	m.Chat("Server", fmt.Sprintf("%s started spectating", s.User.Username))

	if m.hasStarted() {
		m.sendLatestState(s)
	} else {
		m.sendTossStep(s)
	}
}

// beginToss deals both players their decks and asks the chosen player, or a
// random one, to call the coin toss.
func (m *Match) beginToss() {
	m.Chat("Server", fmt.Sprintf("%s joined the game", m.Player1.Username))
	m.Chat("Server", fmt.Sprintf("%s joined the game", m.Player2.Username))

	m.Player1.Player.CreateDeck(m.HostDeck)
	m.Player2.Player.CreateDeck(m.GuestDeck)

	m.Player1.Player.Ready = true
	m.Player2.Player.Ready = true

	m.setupMu.Lock()
	caller := m.tossCaller
	if caller != m.HostID && caller != m.GuestID {
		caller = m.HostID
		if rand.Intn(2) == 1 {
			caller = m.GuestID
		}
	}
	m.PlayerSelectingToss = caller
	m.setupMu.Unlock()

	m.Chat("Server", fmt.Sprintf("%s calls the coin toss", m.tossPlayer(caller).Username))
	m.broadcastTossStep()
}

// callToss settles the coin toss with the caller's call, heads or tails, and
// asks its winner to choose who plays first. Anything else is ignored.
func (m *Match) callToss(s *server.Socket, call int) {
	if call != heads && call != tails {
		return
	}

	m.setupMu.Lock()
	allowed := m.tossBegun && !m.turnChosen && m.TossOutcome == 0 && m.PlayerSelectingToss == s.User.UID
	if allowed {
		m.TossPrediction = call
		m.TossOutcome = heads
		if rand.Intn(2) == 1 {
			m.TossOutcome = tails
		}
	}
	outcome := m.TossOutcome
	winner := m.tossWinner()
	m.setupMu.Unlock()

	if !allowed {
		return
	}

	m.Chat("Server", fmt.Sprintf("%s called %s", s.User.Username, sideName(call)))
	m.Chat("Server", fmt.Sprintf("The coin landed on %s: %s won the toss", sideName(outcome), m.tossPlayer(winner).Username))

	m.Broadcast(server.TossResultMessage{
		Header:  "toss_result",
		Caller:  m.tossPlayer(s.User.UID),
		Call:    sideName(call),
		Outcome: sideName(outcome),
		Winner:  m.tossPlayer(winner),
	})
	m.broadcastTossStep()
}

// chooseTurn lets the toss's winner play first or second, and starts the
// game. Anything else is ignored.
func (m *Match) chooseTurn(s *server.Socket, choice int) {
	if choice != playFirst && choice != playSecond {
		return
	}

	m.setupMu.Lock()
	allowed := m.TossOutcome != 0 && !m.turnChosen && m.tossWinner() == s.User.UID
	if allowed {
		m.turnChosen = true
	}
	m.setupMu.Unlock()

	if !allowed {
		return
	}

	chooserIsHost := s.User.UID == m.HostID
	hostFirst := chooserIsHost == (choice == playFirst)
	// BeginNewTurn passes the turn on before the first turn, so the turn is
	// set to the player who does not play first.
	m.Turn = 1
	if hostFirst {
		m.Turn = 2
	}

	if choice == playFirst {
		m.Chat("Server", fmt.Sprintf("%s decided to play first", s.User.Username))
	} else {
		m.Chat("Server", fmt.Sprintf("%s decided to play second", s.User.Username))
	}

	m.Start()
}

// tossWinner is the id of the coin toss's winner, or empty before the toss is
// called. The caller holds setupMu.
func (m *Match) tossWinner() string {
	if m.TossOutcome == 0 {
		return ""
	}
	if m.TossPrediction == m.TossOutcome {
		return m.PlayerSelectingToss
	}
	if m.PlayerSelectingToss == m.HostID {
		return m.GuestID
	}
	return m.HostID
}

// tossPlayer is a player of the match, by id, as toss messages name them.
func (m *Match) tossPlayer(uid string) server.TossPlayer {
	if uid == m.HostID {
		return server.TossPlayer{UID: uid, Username: m.hostUsername}
	}
	return server.TossPlayer{UID: uid, Username: m.guestUsername}
}

// tossStep is the coin toss's current step for the player or spectator uid:
// what they are asked to do or wait for, after the toss's result once it is
// known. It is empty once the game has started.
func (m *Match) tossStep(uid string) []any {
	m.setupMu.Lock()
	caller, call, outcome, winner := m.PlayerSelectingToss, m.TossPrediction, m.TossOutcome, m.tossWinner()
	begun, chosen := m.tossBegun, m.turnChosen
	m.setupMu.Unlock()

	switch {
	case !begun || chosen:
		return nil
	case outcome == 0:
		header := "toss_being_chosen"
		if uid == caller {
			header = "choose_toss"
		}
		return []any{server.TossMessage{Header: header, Player: m.tossPlayer(caller)}}
	default:
		header := "turn_being_chosen"
		if uid == winner {
			header = "choose_turn"
		}
		return []any{
			server.TossResultMessage{
				Header:  "toss_result",
				Caller:  m.tossPlayer(caller),
				Call:    sideName(call),
				Outcome: sideName(outcome),
				Winner:  m.tossPlayer(winner),
			},
			server.TossMessage{Header: header, Player: m.tossPlayer(winner)},
		}
	}
}

// sendTossStep tells one socket the coin toss's current step, for a player
// coming back or a spectator arriving.
func (m *Match) sendTossStep(s *server.Socket) {
	for _, message := range m.tossStep(s.User.UID) {
		s.Send(message)
	}
}

// broadcastTossStep tells everyone in the match the coin toss's next step.
// The result itself has just been broadcast, so only the step is sent.
func (m *Match) broadcastTossStep() {
	m.setupMu.Lock()
	player1, player2 := m.Player1, m.Player2
	m.setupMu.Unlock()

	sendStep := func(s *server.Socket) {
		if s == nil {
			return
		}
		step := m.tossStep(s.User.UID)
		if len(step) > 0 {
			s.Send(step[len(step)-1])
		}
	}

	for _, player := range []*PlayerReference{player1, player2} {
		if player != nil {
			sendStep(player.Socket)
		}
	}
	for _, spectator := range m.spectators.Iter() {
		sendStep(spectator.Socket)
	}
}

func sideName(side int) string {
	if side == heads {
		return "heads"
	}
	return "tails"
}

// leave handles a player's socket closing: the opponent is told, and the
// player has the disconnect grace to come back before losing. When both
// players have gone, the game ends at once and the first to leave loses.
func (m *Match) leave(s *server.Socket) {
	m.setupMu.Lock()
	var player, opponent *PlayerReference
	switch {
	case m.Player1 != nil && m.Player1.Socket == s:
		player, opponent = m.Player1, m.Player2
	case m.Player2 != nil && m.Player2.Socket == s:
		player, opponent = m.Player2, m.Player1
	}
	if player == nil {
		// Not a player's current socket, such as one replaced by a newer one.
		m.setupMu.Unlock()
		return
	}

	opponentGone := opponent != nil && (opponent.Socket == nil || opponent.Socket.IsClosed())
	uid := player.UID
	if _, counting := m.disconnectTimers[uid]; !opponentGone && !counting {
		m.disconnectTimers[uid] = time.AfterFunc(m.disconnectGrace, func() {
			m.disconnectExpired(uid)
		})
	}
	m.setupMu.Unlock()

	if opponentGone {
		m.end(player.Player, fmt.Sprintf("%s won: %s left the game and did not come back.", player.Username, opponent.Username), EndingDisconnect)
		return
	}

	if opponent != nil {
		opponent.Socket.Send(server.Message{Header: "opponent_disconnected"})
	}
	m.Chat("Server", fmt.Sprintf("%s disconnected and has %s to come back", player.Username, graceText(m.disconnectGrace)))
}

// disconnectExpired ends the game when a disconnected player has not come
// back in time: their opponent wins, or the match closes if it never started.
func (m *Match) disconnectExpired(uid string) {
	if m.IsEnding() {
		return
	}

	m.setupMu.Lock()
	_, waiting := m.disconnectTimers[uid]
	delete(m.disconnectTimers, uid)
	player, opponent := m.Player1, m.Player2
	if uid == m.GuestID {
		player, opponent = m.Player2, m.Player1
	}
	m.setupMu.Unlock()

	// A player who came back stopped their timer; one already running may
	// still arrive here, and is ignored.
	if !waiting || player == nil {
		return
	}

	if opponent == nil {
		m.Dispose()
		return
	}

	m.end(opponent.Player, fmt.Sprintf("%s won: %s left the game and did not come back.", opponent.Username, player.Username), EndingDisconnect)
}

// stopDisconnectTimer cancels a returning player's time limit. The caller
// holds setupMu.
func (m *Match) stopDisconnectTimer(uid string) {
	if timer, ok := m.disconnectTimers[uid]; ok {
		timer.Stop()
		delete(m.disconnectTimers, uid)
	}
}

// graceText writes a time limit for the chat, such as "2 minutes".
func graceText(grace time.Duration) string {
	if grace >= time.Minute && grace%time.Minute == 0 {
		minutes := int(grace / time.Minute)
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}

	seconds := int(grace.Round(time.Second) / time.Second)
	if seconds == 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", seconds)
}
