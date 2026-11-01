package computer

import (
	"bytes"
	"encoding/json"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// question is a question the engine has put to the computer.
type question struct {
	id int
	// show is a preview of cards it must close, which asks nothing else.
	show       bool
	actionType string
	text       string
	about      server.PromptAbout
	// cards are the cards offered, those of every part of a question in
	// parts together.
	cards        []server.CardState
	unselectable map[string]bool
	min, max     int
	cancellable  bool
	choices      []string
	// refusals counts the answers to it the engine refused.
	refusals int
}

// opponents reports whether a card offered is the opponent's.
func (q *question) opponents(cardID string) bool {
	for _, id := range q.about.OpponentCards {
		if id == cardID {
			return true
		}
	}
	return false
}

// selectable are the cards offered that may be chosen.
func (q *question) selectable() []server.CardState {
	selectable := make([]server.CardState, 0, len(q.cards))
	for _, card := range q.cards {
		if !q.unselectable[card.CardID] {
			selectable = append(selectable, card)
		}
	}
	return selectable
}

// view is what the computer has been told about its game.
type view struct {
	state *server.MatchState
	// question is the question open for the computer, if any.
	question *question
	// waiting is set while the opponent answers a question, which nothing
	// else happens during.
	waiting bool
	// callToss and chooseTurn are set while the computer is to call the coin
	// toss, or choose whether to play first as its winner.
	callToss, chooseTurn bool
	// finished is set once the game is over.
	finished bool
	// warnings counts the warnings the engine sent, each refusing a move.
	warnings int
}

// actionMessage is a question as it arrives: the cards offered are a list,
// or for a question in parts a list for each part.
type actionMessage struct {
	server.Prompt
	server.PromptAbout
	ActionType        string             `json:"actionType"`
	Text              string             `json:"text"`
	Cards             json.RawMessage    `json:"cards"`
	MinSelections     int                `json:"minSelections"`
	MaxSelections     int                `json:"maxSelections"`
	Cancellable       bool               `json:"cancellable"`
	UnselectableCards []server.CardState `json:"unselectableCards"`
	Choices           []string           `json:"choices"`
}

// take updates the view with a message from the engine. A message it does not
// need is ignored, as is one it cannot read.
func (v *view) take(data []byte) {
	var message server.Message
	if json.Unmarshal(data, &message) != nil {
		return
	}

	switch message.Header {
	case "state_update":
		var update server.MatchStateMessage
		if json.Unmarshal(data, &update) == nil {
			v.state = &update.State
		}
	case "action":
		if q, ok := readQuestion(data); ok {
			v.question = q
		}
	case "show_cards_non_dismissible":
		var show server.ShowCardsMessage
		if json.Unmarshal(data, &show) == nil {
			v.question = &question{id: show.ID, show: true}
		}
	case "close_action":
		v.question = nil
	case "action_error":
		if v.question != nil {
			v.question.refusals++
		}
	case "warn":
		v.warnings++
	case "wait":
		v.waiting = true
	case "end_wait":
		v.waiting = false
	case "choose_toss":
		v.callToss = true
	case "choose_turn":
		v.chooseTurn = true
	case "duel_finished":
		v.finished = true
		v.question = nil
	}
}

func readQuestion(data []byte) (*question, bool) {
	var message actionMessage
	if json.Unmarshal(data, &message) != nil {
		return nil, false
	}

	q := &question{
		id:           message.ID,
		actionType:   message.ActionType,
		text:         message.Text,
		about:        message.PromptAbout,
		unselectable: map[string]bool{},
		min:          message.MinSelections,
		max:          message.MaxSelections,
		cancellable:  message.Cancellable,
		choices:      message.Choices,
	}

	raw := bytes.TrimSpace(message.Cards)
	switch {
	case len(raw) == 0 || bytes.Equal(raw, []byte("null")):
	case raw[0] == '{':
		var parts map[string][]server.CardState
		if json.Unmarshal(raw, &parts) != nil {
			return nil, false
		}
		for _, part := range parts {
			q.cards = append(q.cards, part...)
		}
	default:
		if json.Unmarshal(raw, &q.cards) != nil {
			return nil, false
		}
	}

	// Cards shown but not offered come with the question, as unselectable.
	for _, card := range message.UnselectableCards {
		q.unselectable[card.CardID] = true
	}

	return q, true
}
