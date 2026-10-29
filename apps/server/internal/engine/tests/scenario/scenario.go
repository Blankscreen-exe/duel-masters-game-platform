package scenario

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"time"
)

const actionTimeout = 2 * time.Second

type TestScenario struct {
	Match       *match.Match
	connections map[*match.PlayerReference]*MockConnection
}

type Option func(*scenarioConfig)

type scenarioConfig struct {
	deck        []string
	chatRefusal func(userID string) string
}

type DeckEntry struct {
	UID   string
	Count int
}

func WithDeck(entries ...DeckEntry) Option {
	return func(opts *scenarioConfig) {
		deck := make([]string, 0)

		for _, entry := range entries {
			for range entry.Count {
				deck = append(deck, entry.UID)
			}
		}

		opts.deck = deck
	}
}

// WithChatRefusal asks refusal about every chat message, as the server does
// to limit how fast players chat (match.Options.ChatRefusal).
func WithChatRefusal(refusal func(userID string) string) Option {
	return func(opts *scenarioConfig) {
		opts.chatRefusal = refusal
	}
}

// scenarioDisconnectGrace is long enough that no scenario ends by a player
// not coming back.
const scenarioDisconnectGrace = time.Hour

// RegisterCards makes every card the engine implements known to matches, as
// the server does at startup. Calling it again changes nothing.
func RegisterCards() {
	for _, set := range cards.Sets {
		for uid, ctor := range *set {
			if ctor == nil {
				continue
			}
			match.AddCard(uid, ctor)
		}
	}
}

// DefaultDeck is the deck scenarios play with: 40 copies of one creature.
func DefaultDeck() []string {
	return defaultDeck()
}

func New(options ...Option) *TestScenario {
	RegisterCards()

	config := scenarioConfig{deck: defaultDeck()}
	for _, option := range options {
		option(&config)
	}

	matchSystem := match.NewSystem(match.SystemOptions{})
	m, err := matchSystem.NewMatch(match.Options{
		Name:            "test-scenario",
		Host:            match.Seat{ID: "1", Username: "Player1"},
		Guest:           match.Seat{ID: "2", Username: "Player2"},
		Visible:         true,
		DisconnectGrace: scenarioDisconnectGrace,
		ChatRefusal:     config.chatRefusal,
	})
	if err != nil {
		panic(err)
	}

	p1Conn := NewMockConnection()
	p1 := match.NewPlayer(m, 1)
	m.Player1 = match.NewPlayerReference(p1, server.NewSocket(p1Conn, m, "1", "Player1"))

	p2Conn := NewMockConnection()
	p2 := match.NewPlayer(m, 2)
	m.Player2 = match.NewPlayerReference(p2, server.NewSocket(p2Conn, m, "2", "Player2"))

	p1.CreateDeck(cloneDeck(config.deck))
	p2.CreateDeck(cloneDeck(config.deck))

	p1.Ready = true
	p2.Ready = true

	m.Start()

	return &TestScenario{
		Match: m,
		connections: map[*match.PlayerReference]*MockConnection{
			m.Player1: p1Conn,
			m.Player2: p2Conn,
		},
	}
}

func (s *TestScenario) FindCard(player *match.Player, zone string, imageID string) (*match.Card, error) {
	cards, err := player.Container(zone)
	if err != nil {
		return nil, err
	}

	for _, card := range cards {
		if card.ImageID == imageID {
			return card, nil
		}
	}

	return nil, fmt.Errorf("card %s not found in %s", imageID, zone)
}

func (s *TestScenario) ActionPlayCard(player *match.PlayerReference, cardID string) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	card, err := player.Player.GetCard(cardID, match.HAND)
	if err != nil {
		return err
	}

	action, err := s.ActionPlayCardPrompt(player, cardID)
	if err != nil {
		return err
	}

	selection, err := s.selectPlayableMana(card, action)
	if err != nil {
		return err
	}

	completionStart := conn.JSONWriteCount()
	if err := s.SubmitAction(player, selection...); err != nil {
		return err
	}

	// PlayCard broadcasts state after the complete event has resolved. A card
	// may instead open a prompt for either player while PlayCard is still
	// resolving; the acting player receives either that action or a wait message
	// so the caller can answer it. These messages are synchronized completion
	// signals, unlike polling card.Zone while effects are still moving cards.
	if err := s.WaitForMessage(player, completionStart, "state_update", "action", "wait"); err != nil {
		return err
	}

	// A spell broadcasts once it has resolved and again once PlayCard has, so
	// the first state is not the end of the play.
	return s.settleUnlessPrompted(player, completionStart)
}

// ActionPlayCardPrompt plays a card from the player's hand and returns the
// question asking which mana pays for it, without answering it. The caller
// must always answer that question so the sequential event loop is never left
// waiting on it.
//
// Use ActionPlayCard instead when the card is meant to be played.
func (s *TestScenario) ActionPlayCardPrompt(player *match.PlayerReference, cardID string) (*server.ActionMessage, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	if _, err := player.Player.GetCard(cardID, match.HAND); err != nil {
		return nil, err
	}

	messageCount := conn.JSONWriteCount()

	if err := s.send(player, struct {
		Header string `json:"header"`
		ID     string `json:"virtualId"`
	}{
		Header: "add_to_playzone",
		ID:     cardID,
	}); err != nil {
		return nil, err
	}

	return s.waitForActionMessage(player, messageCount)
}

func (s *TestScenario) ActionEndTurn(player *match.PlayerReference) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	messageCount := conn.JSONWriteCount()

	// The turn being started belongs to the opponent, so a start-of-turn effect
	// such as silent skill prompts them rather than the player ending the turn.
	// That prompt blocks the transition just as an end step prompt does, and it
	// never reaches this player's connection, so both sides have to be watched.
	opponent := s.Match.PlayerRef(s.Match.Opponent(player.Player))
	opponentConn, err := s.connectionFor(opponent)
	if err != nil {
		return err
	}

	opponentMessageCount := opponentConn.JSONWriteCount()

	openedPrompt := func() bool {
		for _, raw := range opponentConn.JSONMessagesSince(opponentMessageCount) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			if header.Header == "action" {
				return true
			}
		}

		return false
	}

	if err := s.send(player, struct {
		Header string `json:"header"`
	}{
		Header: "end_turn",
	}); err != nil {
		return err
	}

	// A successful turn transition broadcasts once before untap/start/draw and
	// again after draw. Wait for the second update so tests cannot observe the
	// event loop halfway through the new turn. A prevented transition sends a
	// warning instead, and an end step effect may suspend the transition on a
	// prompt that the caller has to answer before the turn can finish.
	if err := s.waitFor(func() bool {
		stateUpdates := 0
		for _, raw := range conn.JSONMessagesSince(messageCount) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			switch header.Header {
			case "warn", "action", "wait":
				return true
			case "state_update":
				stateUpdates++
			}
		}

		// An end of turn effect can finish the game rather than start the next
		// turn, in which case the broadcasts a turn transition would have sent
		// never arrive.
		return stateUpdates >= 2 || openedPrompt() || s.Match.IsClosed()
	}); err != nil {
		return err
	}

	if s.Match.IsClosed() {
		return nil
	}

	// An end-of-turn effect may have opened a prompt, in which case the event
	// loop is deliberately blocked until the caller answers it.
	headers, err := s.MessageHeaders(player, messageCount)
	if err != nil {
		return err
	}
	for _, header := range headers {
		if header == "action" || header == "wait" {
			return nil
		}
	}

	if openedPrompt() {
		return nil
	}

	// A turn transition keeps running past its last state broadcast, so the test
	// goroutine must not touch match state until the event loop is idle again.
	return s.WaitForEventLoop()
}

// WaitForEventLoop blocks until the match event loop has finished whatever it
// was processing: every sequential event scheduled so far has run. Events write their messages synchronously, so they
// have reached the players' connections too.
//
// Call this before reading or mutating match state directly after a scenario
// action, because an action can return while the engine is still executing the
// steps that follow its last state broadcast.
func (s *TestScenario) WaitForEventLoop() error {
	return s.waitForIdleLoop(func() bool { return false })
}

// settleUnlessPrompted waits for the event loop to finish what an action
// started, as WaitForEventLoop does, unless the action has opened a prompt for
// the player, or made them wait on one for their opponent. A prompt blocks the
// loop until it is answered, so the caller is left to answer it instead.
//
// An action's last state broadcast does not mean it has finished: the loop can
// still be working through what follows it, and state read meanwhile is
// mid-change.
func (s *TestScenario) settleUnlessPrompted(player *match.PlayerReference, since int) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	return s.waitForIdleLoop(func() bool {
		for _, raw := range conn.JSONMessagesSince(since) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			if header.Header == "action" || header.Header == "wait" {
				return true
			}
		}

		return false
	})
}

// waitForIdleLoop waits until the match has no sequential events pending, or
// until stop reports that there is no point waiting any longer. It only reads
// the match's pending count, so waiting never adds an event of its own.
func (s *TestScenario) waitForIdleLoop(stop func() bool) error {
	deadline := time.Now().Add(actionTimeout)
	for time.Now().Before(deadline) {
		if stop() || !s.Match.HasPendingEvents() {
			return nil
		}

		time.Sleep(time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for the match event loop to become idle after %s", actionTimeout)
}

// ActionAttackCreature attacks a specific opposing creature and waits until
// the confirmed attack and its resulting battle have finished resolving.
func (s *TestScenario) ActionAttackCreature(player *match.PlayerReference, attackerID string, defenderID string) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	if _, err := player.Player.GetCard(attackerID, match.BATTLEZONE); err != nil {
		return err
	}

	opponent := s.Match.Opponent(player.Player)
	if _, err := opponent.GetCard(defenderID, match.BATTLEZONE); err != nil {
		return err
	}

	messageCount := conn.JSONWriteCount()
	if err := s.send(player, struct {
		Header string `json:"header"`
		ID     string `json:"virtualId"`
	}{
		Header: "attack_creature",
		ID:     attackerID,
	}); err != nil {
		return err
	}

	action, err := s.waitForAttackPrompt(player, messageCount)
	if err != nil {
		return err
	}

	defenderOffered := false
	for _, candidate := range action.Cards {
		if candidate.CardID == defenderID {
			defenderOffered = true
			break
		}
	}
	if !defenderOffered {
		// Attack target selection is cancellable. Answer the outstanding prompt
		// and let the cancelled attack unwind before returning, so a caller that
		// expects this error finds the match settled.
		_ = s.CancelAction(player)
		_ = s.WaitForEventLoop()
		return fmt.Errorf("creature %s was not offered as an attack target", defenderID)
	}

	completionStart := conn.JSONWriteCount()
	if err := s.SubmitAction(player, defenderID); err != nil {
		return err
	}

	// Confirming an attack broadcasts once after tapping the attacker, the
	// battle broadcasts, and the outer AttackCreature action broadcasts again
	// after all nested effects have returned. A nested effect may instead open a
	// prompt for either player, in which case the loop is blocked until the
	// caller answers it and the later broadcasts never arrive; a prompt for the
	// defender reaches this player as a wait.
	if err := s.waitFor(func() bool {
		stateUpdates := 0
		for _, raw := range conn.JSONMessagesSince(completionStart) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			switch header.Header {
			case "action", "wait":
				return true
			case "state_update":
				stateUpdates++
			}
		}

		return stateUpdates >= 2
	}); err != nil {
		return err
	}

	return s.settleUnlessPrompted(player, completionStart)
}

// ActionUseTapAbility taps a creature to use its tap ability and waits until it
// has resolved. It returns early when the ability opens a prompt for its
// controller, which the caller then answers.
func (s *TestScenario) ActionUseTapAbility(player *match.PlayerReference, cardID string) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	if _, err := player.Player.GetCard(cardID, match.BATTLEZONE); err != nil {
		return err
	}

	messageCount := conn.JSONWriteCount()
	if err := s.send(player, struct {
		Header string `json:"header"`
		ID     string `json:"virtualId"`
	}{
		Header: "tap_ability",
		ID:     cardID,
	}); err != nil {
		return err
	}

	if err := s.waitFor(func() bool {
		for _, raw := range conn.JSONMessagesSince(messageCount) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			switch header.Header {
			case "state_update", "action", "wait", "warn":
				return true
			}
		}

		return false
	}); err != nil {
		return err
	}

	// A rejected tap ability answers with a warning and nothing else.
	headers, err := s.MessageHeaders(player, messageCount)
	if err != nil {
		return err
	}
	for _, header := range headers {
		if header == "action" || header == "wait" {
			return nil
		}
		if header == "warn" {
			// The match still broadcasts after the warning, so let the rejected
			// ability finish unwinding before the caller acts again.
			_ = s.WaitForEventLoop()
			return fmt.Errorf("the tap ability was rejected")
		}
	}

	return s.WaitForEventLoop()
}

// ActionAttackPlayer attacks the opponent directly and returns the shield
// selection prompt the attacker receives, without answering it. The caller must
// always answer that prompt through ResolveAttack or CancelAction so the
// sequential event loop is never left waiting on a response.
//
// The defender must hold at least one shield; an attack against an empty shield
// zone never opens a shield selection and ends the match instead.
func (s *TestScenario) ActionAttackPlayer(player *match.PlayerReference, attackerID string) (*server.ActionMessage, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	if _, err := player.Player.GetCard(attackerID, match.BATTLEZONE); err != nil {
		return nil, err
	}

	messageCount := conn.JSONWriteCount()
	if err := s.send(player, struct {
		Header string `json:"header"`
		ID     string `json:"virtualId"`
	}{
		Header: "attack_player",
		ID:     attackerID,
	}); err != nil {
		return nil, err
	}

	return s.waitForAttackPrompt(player, messageCount)
}

// ActionAttackCreaturePrompt attacks with a creature and returns the target
// selection prompt without answering it, for tests that need to cancel it. The
// caller must always answer that prompt through SubmitAction or CancelAction so
// the sequential event loop is never left waiting on a response.
//
// Use ActionAttackCreature instead when the attack is meant to go through.
func (s *TestScenario) ActionAttackCreaturePrompt(player *match.PlayerReference, attackerID string) (*server.ActionMessage, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	if _, err := player.Player.GetCard(attackerID, match.BATTLEZONE); err != nil {
		return nil, err
	}

	messageCount := conn.JSONWriteCount()
	if err := s.send(player, struct {
		Header string `json:"header"`
		ID     string `json:"virtualId"`
	}{
		Header: "attack_creature",
		ID:     attackerID,
	}); err != nil {
		return nil, err
	}

	return s.waitForAttackPrompt(player, messageCount)
}

// ActionChargeMana puts a card from the player's hand into their manazone and
// waits for the engine to answer, either with the resulting state broadcast or
// with the warning that explains why the charge was refused. Inspect the hand
// and manazone afterwards to tell the two apart.
func (s *TestScenario) ActionChargeMana(player *match.PlayerReference, cardID string) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	if _, err := player.Player.GetCard(cardID, match.HAND); err != nil {
		return err
	}

	messageCount := conn.JSONWriteCount()
	if err := s.send(player, struct {
		Header string `json:"header"`
		ID     string `json:"virtualId"`
	}{
		Header: "add_to_manazone",
		ID:     cardID,
	}); err != nil {
		return err
	}

	if err := s.waitFor(func() bool {
		for _, raw := range conn.JSONMessagesSince(messageCount) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			switch header.Header {
			case "state_update", "warn":
				return true
			}
		}

		return false
	}); err != nil {
		return err
	}

	return s.WaitForEventLoop()
}

// ResolveAttack answers the attacker's pending shield selection with the given
// shields and waits until the attack has finished resolving. It returns early
// when the attack opens another prompt for the attacker, or puts them into a
// wait state because the defender was offered a block; the caller then answers
// whichever prompt is outstanding.
func (s *TestScenario) ResolveAttack(player *match.PlayerReference, shieldIDs ...string) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	completionStart := conn.JSONWriteCount()
	if err := s.SubmitAction(player, shieldIDs...); err != nil {
		return err
	}

	// Confirming the attack broadcasts once after tapping the attacker and the
	// outer AttackPlayer action broadcasts again once blocking, battles, shield
	// breaking and their nested effects have returned. A battle with a blocker
	// broadcasts in between.
	if err := s.waitFor(func() bool {
		stateUpdates := 0
		for _, raw := range conn.JSONMessagesSince(completionStart) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			switch header.Header {
			case "action", "wait":
				return true
			case "state_update":
				stateUpdates++
			}
		}

		return stateUpdates >= 2
	}); err != nil {
		return err
	}

	return s.settleUnlessPrompted(player, completionStart)
}

// MessageMark records how many messages each player had been sent at one
// moment, so a later wait only looks at what came after it.
type MessageMark map[*match.PlayerReference]int

// Mark records how many messages each player has been sent so far.
func (s *TestScenario) Mark() MessageMark {
	mark := make(MessageMark, len(s.connections))
	for player, conn := range s.connections {
		mark[player] = conn.JSONWriteCount()
	}

	return mark
}

// WaitForAnswerToSettle waits, after player has answered a prompt, until what
// the answer set in motion has finished. Take the mark before sending the
// answer. It returns once:
//   - the event loop has no events pending; or
//   - any player has been sent a new prompt, or told to wait on one, since the
//     mark: a prompt blocks the event loop until it is answered, so the caller
//     answers it next. Prompts are "action" messages and the card previews
//     that must be closed ("show_cards_non_dismissible"), which are all the
//     engine waits on a player's answer for; or
//   - player's answer was refused (action_error or warn): the prompt stays
//     open.
//
// A state update is not a sign that the answer has been dealt with: the event
// waiting on the prompt goes on running after it.
func (s *TestScenario) WaitForAnswerToSettle(player *match.PlayerReference, mark MessageMark) error {
	return s.waitForIdleLoop(func() bool {
		for other, since := range mark {
			conn, err := s.connectionFor(other)
			if err != nil {
				continue
			}

			for _, raw := range conn.JSONMessagesSince(since) {
				var header server.Message
				if err := json.Unmarshal([]byte(raw), &header); err != nil {
					continue
				}

				switch header.Header {
				case "action", "show_cards_non_dismissible", "wait":
					return true
				case "action_error", "warn":
					if other == player {
						return true
					}
				}
			}
		}

		return false
	})
}

// playerAnswer is the message a client answers a question with.
type playerAnswer struct {
	Header string   `json:"header"`
	Prompt int      `json:"prompt"`
	Cards  []string `json:"cards"`
	Count  int      `json:"count"`
	Cancel bool     `json:"cancel"`
}

// answerOpenPrompt answers the question open for player, as a client answers
// the question it shows. It waits for a question to be open.
func (s *TestScenario) answerOpenPrompt(player *match.PlayerReference, answer playerAnswer) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	var prompt int
	if err := s.waitFor(func() bool {
		prompt = conn.OpenPrompt()
		return prompt != 0
	}); err != nil {
		return fmt.Errorf("no question is open for %s to answer: %w", player.Username, err)
	}

	return s.SendAnswer(player, prompt, answer.Cancel, answer.Count, answer.Cards...)
}

// SendAnswer sends player's answer to question prompt, whether or not it is
// open, as a client does that answers a question it showed earlier.
func (s *TestScenario) SendAnswer(player *match.PlayerReference, prompt int, cancel bool, count int, cardIDs ...string) error {
	return s.send(player, playerAnswer{
		Header: "action",
		Prompt: prompt,
		Cards:  cardIDs,
		Count:  count,
		Cancel: cancel,
	})
}

// SubmitAction answers the open question by choosing cards.
func (s *TestScenario) SubmitAction(player *match.PlayerReference, cardIDs ...string) error {
	return s.answerOpenPrompt(player, playerAnswer{Cards: cardIDs, Count: len(cardIDs)})
}

// SubmitCount answers a prompt that asks for a number rather than for cards,
// such as fx.SelectCount.
func (s *TestScenario) SubmitCount(player *match.PlayerReference, count int) error {
	return s.answerOpenPrompt(player, playerAnswer{Cards: []string{}, Count: count})
}

// SubmitChoice answers a multiple-choice action using its zero-based option index.
func (s *TestScenario) SubmitChoice(player *match.PlayerReference, choice int) error {
	return s.answerOpenPrompt(player, playerAnswer{Count: choice})
}

// CancelAction answers "no" or cancels a pending action prompt (e.g. a binary yes/no question).
func (s *TestScenario) CancelAction(player *match.PlayerReference) error {
	return s.answerOpenPrompt(player, playerAnswer{Cards: []string{}, Cancel: true})
}

// MessageHeaders returns the header field of every JSON message sent to the player so far,
// starting from position since. Useful for debugging test failures.
func (s *TestScenario) MessageHeaders(player *match.PlayerReference, since int) ([]string, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}
	var headers []string
	for _, raw := range conn.JSONMessagesSince(since) {
		var h server.Message
		if json.Unmarshal([]byte(raw), &h) == nil {
			headers = append(headers, h.Header)
		}
	}
	return headers, nil
}

// Warnings returns the text of every "warn" message sent to the player since
// position since, so tests can assert how often an effect warned and why.
func (s *TestScenario) Warnings(player *match.PlayerReference, since int) ([]string, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	warnings := make([]string, 0)
	for _, raw := range conn.JSONMessagesSince(since) {
		var header server.Message
		if err := json.Unmarshal([]byte(raw), &header); err != nil || header.Header != "warn" {
			continue
		}

		message := &server.WarningMessage{}
		if err := json.Unmarshal([]byte(raw), message); err == nil {
			warnings = append(warnings, message.Message)
		}
	}

	return warnings, nil
}

// SendChat sends a chat message from the player, as the duel board does.
// Chat runs beside the event loop; wait for what it sends with WaitForMessage.
func (s *TestScenario) SendChat(player *match.PlayerReference, text string) error {
	return s.send(player, struct {
		Header  string `json:"header"`
		Message string `json:"message"`
	}{Header: "chat", Message: text})
}

// ChatMessages returns the text of every chat message the player received since
// position since. Effects report what they did through the chat, so this is how
// a test asserts that something was announced and not only performed.
func (s *TestScenario) ChatMessages(player *match.PlayerReference, since int) ([]string, error) {
	log, err := s.ChatLog(player, since)
	if err != nil {
		return nil, err
	}

	messages := make([]string, 0, len(log))
	for _, message := range log {
		messages = append(messages, message.Message)
	}

	return messages, nil
}

// ChatLog returns every chat message the player received since position
// since, whole, including the cards a game log line names.
func (s *TestScenario) ChatLog(player *match.PlayerReference, since int) ([]server.ChatMessage, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	messages := make([]server.ChatMessage, 0)
	for _, raw := range conn.JSONMessagesSince(since) {
		var header server.Message
		if err := json.Unmarshal([]byte(raw), &header); err != nil || header.Header != "chat" {
			continue
		}

		message := server.ChatMessage{}
		if err := json.Unmarshal([]byte(raw), &message); err == nil {
			messages = append(messages, message)
		}
	}

	return messages, nil
}

// LatestState returns the most recent match state sent to the player at or
// after position since, as the duel interface would show it.
func (s *TestScenario) LatestState(player *match.PlayerReference, since int) (*server.MatchState, error) {
	states, err := s.StatesSince(player, since)
	if err != nil {
		return nil, err
	}

	if len(states) == 0 {
		return nil, fmt.Errorf("no state_update sent to %s since message %d", player.Socket.User.Username, since)
	}

	return &states[len(states)-1], nil
}

// StatesSince returns every match state sent to the player at or after
// position since, oldest first.
func (s *TestScenario) StatesSince(player *match.PlayerReference, since int) ([]server.MatchState, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	return StatesOn(conn, since)
}

// Spectate joins the started match as a spectator with the given user id and
// returns the spectator's connection once the match has sent it a state. Joining
// runs beside the event loop, so a spectator can join while it waits on a
// prompt. Read what the spectator was sent with StatesOn.
func (s *TestScenario) Spectate(userID string, username string) (*MockConnection, error) {
	conn := NewMockConnection()
	socket := server.NewSocket(conn, s.Match, userID, username)

	data, err := json.Marshal(server.Message{Header: "join_match"})
	if err != nil {
		return nil, err
	}

	s.Match.Parse(socket, data)

	if err := s.waitFor(func() bool {
		states, err := StatesOn(conn, 0)
		return err == nil && len(states) > 0
	}); err != nil {
		return nil, fmt.Errorf("spectator %s was never sent a state: %w", username, err)
	}

	return conn, nil
}

// StatesOn returns every match state written to a connection at or after
// position since, oldest first.
func StatesOn(conn *MockConnection, since int) ([]server.MatchState, error) {
	states := make([]server.MatchState, 0)
	for _, raw := range conn.JSONMessagesSince(since) {
		var header server.Message
		if err := json.Unmarshal([]byte(raw), &header); err != nil || header.Header != "state_update" {
			continue
		}

		message := &server.MatchStateMessage{}
		if err := json.Unmarshal([]byte(raw), message); err != nil {
			return nil, err
		}
		states = append(states, message.State)
	}

	return states, nil
}

// MessageCount returns the number of JSON messages the server has sent to the player so far.
// Capture this value BEFORE an action so that WaitForMessage can find the response.
func (s *TestScenario) MessageCount(player *match.PlayerReference) (int, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return 0, err
	}
	return conn.JSONWriteCount(), nil
}

// WaitForMessage blocks until a server message whose header matches one of the supplied values
// appears at a position >= since in the player's received message log.
// Always capture since = MessageCount(player) BEFORE the action that triggers the message.
func (s *TestScenario) WaitForMessage(player *match.PlayerReference, since int, headers ...string) error {
	conn, err := s.connectionFor(player)
	if err != nil {
		return err
	}

	return s.waitForHeaders(conn, since, headers...)
}

// JoinSpectator connects a new spectator to the started match by sending the
// same join message a real client sends. It returns without waiting for the
// join to complete; use WaitForSpectatorMessage to wait for its effects.
func (s *TestScenario) JoinSpectator(uid string, username string) (*server.Socket, *MockConnection, error) {
	conn := NewMockConnection()
	socket := server.NewSocket(conn, s.Match, uid, username)

	data, err := json.Marshal(server.Message{Header: "join_match"})
	if err != nil {
		return nil, nil, err
	}

	s.Match.Parse(socket, data)

	return socket, conn, nil
}

// WaitForSpectatorMessage is WaitForMessage for a spectator connection
// returned by JoinSpectator.
func (s *TestScenario) WaitForSpectatorMessage(conn *MockConnection, since int, headers ...string) error {
	return s.waitForHeaders(conn, since, headers...)
}

func (s *TestScenario) waitForHeaders(conn *MockConnection, since int, headers ...string) error {
	return s.waitFor(func() bool {
		for _, raw := range conn.JSONMessagesSince(since) {
			var h server.Message
			if err := json.Unmarshal([]byte(raw), &h); err != nil {
				continue
			}
			for _, header := range headers {
				if h.Header == header {
					return true
				}
			}
		}
		return false
	})
}

// WaitForAction returns the first standard card or question action sent at or
// after since.
func (s *TestScenario) WaitForAction(player *match.PlayerReference, since int) (*server.ActionMessage, error) {
	return s.waitForActionMessage(player, since)
}

// LatestAction returns the most recent standard card or question action sent
// at or after since. This is useful when a higher-level scenario method has
// already answered an earlier prompt, such as mana payment.
func (s *TestScenario) LatestAction(player *match.PlayerReference, since int) (*server.ActionMessage, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	var action *server.ActionMessage
	err = s.waitFor(func() bool {
		for _, raw := range conn.JSONMessagesSince(since) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil || header.Header != "action" {
				continue
			}

			candidate := &server.ActionMessage{}
			if err := json.Unmarshal([]byte(raw), candidate); err != nil {
				continue
			}

			action = candidate
		}

		return action != nil
	})
	if err != nil {
		return nil, err
	}

	return action, nil
}

// WaitForMultipartAction returns the first grouped-card action sent at or after since.
func (s *TestScenario) WaitForMultipartAction(player *match.PlayerReference, since int) (*server.MultipartActionMessage, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	var action *server.MultipartActionMessage
	err = s.waitFor(func() bool {
		for _, raw := range conn.JSONMessagesSince(since) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil || header.Header != "action" {
				continue
			}

			candidate := &server.MultipartActionMessage{}
			if err := json.Unmarshal([]byte(raw), candidate); err != nil || candidate.Cards == nil {
				continue
			}

			action = candidate
			return true
		}

		return false
	})
	if err != nil {
		return nil, err
	}

	return action, nil
}

func (s *TestScenario) connectionFor(player *match.PlayerReference) (*MockConnection, error) {
	conn, ok := s.connections[player]
	if !ok {
		return nil, errors.New("mock connection not found for player")
	}

	return conn, nil
}

func (s *TestScenario) send(player *match.PlayerReference, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	s.Match.Parse(player.Socket, data)
	return nil
}

// waitForAttackPrompt waits for the prompt an attack declaration opens, but
// gives up immediately when the engine rejects the attack with a warning
// instead of making the caller wait out the full action timeout.
func (s *TestScenario) waitForAttackPrompt(player *match.PlayerReference, start int) (*server.ActionMessage, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	var action *server.ActionMessage
	var warning string

	err = s.waitFor(func() bool {
		for _, raw := range conn.JSONMessagesSince(start) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			switch header.Header {
			case "warn":
				message := &server.WarningMessage{}
				if err := json.Unmarshal([]byte(raw), message); err == nil {
					warning = message.Message
				}
				return true
			case "action":
				candidate := &server.ActionMessage{}
				if err := json.Unmarshal([]byte(raw), candidate); err != nil {
					continue
				}

				action = candidate
				return true
			}
		}

		return false
	})
	if err != nil {
		return nil, err
	}

	if action == nil {
		// Let the rejected attack finish unwinding so the caller can act again.
		_ = s.WaitForEventLoop()
		return nil, fmt.Errorf("the attack was rejected: %s", warning)
	}

	return action, nil
}

func (s *TestScenario) waitForActionMessage(player *match.PlayerReference, start int) (*server.ActionMessage, error) {
	conn, err := s.connectionFor(player)
	if err != nil {
		return nil, err
	}

	var action *server.ActionMessage

	err = s.waitFor(func() bool {
		for _, raw := range conn.JSONMessagesSince(start) {
			var header server.Message
			if err := json.Unmarshal([]byte(raw), &header); err != nil {
				continue
			}

			if header.Header != "action" {
				continue
			}

			candidate := &server.ActionMessage{}
			if err := json.Unmarshal([]byte(raw), candidate); err != nil {
				continue
			}

			action = candidate
			return true
		}

		return false
	})
	if err != nil {
		return nil, err
	}

	return action, nil
}

func (s *TestScenario) selectPlayableMana(card *match.Card, action *server.ActionMessage) ([]string, error) {
	blocked := make(map[string]struct{}, len(action.UnselectableCards))
	for _, unselectable := range action.UnselectableCards {
		blocked[unselectable.CardID] = struct{}{}
	}

	available := make([]*match.Card, 0, len(action.Cards))
	for _, candidate := range action.Cards {
		if _, ok := blocked[candidate.CardID]; ok {
			continue
		}

		mana, err := card.Player.GetCard(candidate.CardID, match.MANAZONE)
		if err != nil {
			continue
		}

		available = append(available, mana)
	}

	for selections := action.MinSelections; selections <= action.MaxSelections; selections++ {
		if picked, ok := pickPlayableMana(card, available, selections, 0, nil); ok {
			return picked, nil
		}
	}

	return nil, fmt.Errorf("could not find playable mana selection for %s", card.Name)
}

func (s *TestScenario) waitFor(predicate func() bool) error {
	deadline := time.Now().Add(actionTimeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return nil
		}

		time.Sleep(time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for scenario action after %s", actionTimeout)
}

func pickPlayableMana(card *match.Card, available []*match.Card, target int, start int, current []*match.Card) ([]string, bool) {
	if len(current) == target {
		if !card.Player.CanPlayCard(card, current) {
			return nil, false
		}

		selection := make([]string, len(current))
		for i, mana := range current {
			selection[i] = mana.ID
		}

		return selection, true
	}

	remaining := target - len(current)
	for i := start; i <= len(available)-remaining; i++ {
		if selection, ok := pickPlayableMana(card, available, target, i+1, append(current, available[i])); ok {
			return selection, true
		}
	}

	return nil, false
}

func defaultDeck() []string {
	deck := make([]string, 0, 40)
	for range 40 {
		deck = append(deck, "af3bc221-1cc2-4f58-83ea-2673ac2c66c5") // Immortal Baron, Vorg
	}

	return deck
}

func cloneDeck(deck []string) []string {
	cloned := make([]string, len(deck))
	copy(cloned, deck)
	return cloned
}
