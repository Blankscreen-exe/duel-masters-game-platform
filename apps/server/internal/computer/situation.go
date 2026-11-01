package computer

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
)

// Situation is a game as a live player sees it: only what a player in the seat
// could know, written to be read and decided on.
type Situation struct {
	Status string `json:"status"`
	// Turn counts both players' turns from 1, and Step is the step of it.
	Turn int    `json:"turn,omitempty"`
	Step string `json:"step,omitempty"`
	// ManaCharged is whether this turn's mana has been charged, and CanChargeMana
	// whether it still can be.
	ManaCharged   bool `json:"mana_charged,omitempty"`
	CanChargeMana bool `json:"can_charge_mana,omitempty"`

	You      *SeatView `json:"you,omitempty"`
	Opponent *SeatView `json:"opponent,omitempty"`

	Question *QuestionView `json:"question,omitempty"`
	// Events are the lines of the game log since the last look at the game.
	Events []string `json:"events,omitempty"`
	// Refused says why moves or answers since the last look were not taken.
	Refused []string    `json:"refused,omitempty"`
	Result  *LiveResult `json:"result,omitempty"`
}

// SeatView is one player's side of the board.
type SeatView struct {
	Name      string `json:"name"`
	DeckCount int    `json:"deck_count"`
	HandCount int    `json:"hand_count"`
	// Hand is empty for the opponent's, which only its owner sees.
	Hand        []CardView `json:"hand,omitempty"`
	Battle      []CardView `json:"battle_zone"`
	Mana        []CardView `json:"mana_zone"`
	Graveyard   []string   `json:"graveyard,omitempty"`
	ShieldCount int        `json:"shield_count"`
	// FaceUpShields are the shields that can be read, by name.
	FaceUpShields []string `json:"face_up_shields,omitempty"`
}

// CardView is a card as a live player sees it. Its rules text comes the first
// time they see that card in a game, to keep the rest short.
type CardView struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Type          string   `json:"type,omitempty"`
	Civilizations []string `json:"civilizations,omitempty"`
	Cost          int      `json:"cost,omitempty"`
	// Power is its power right now, and AttackingPower its power attacking when
	// that is more.
	Power          int      `json:"power,omitempty"`
	AttackingPower int      `json:"attacking_power,omitempty"`
	Tapped         bool     `json:"tapped,omitempty"`
	CanPlay        bool     `json:"can_play,omitempty"`
	CanAttack      bool     `json:"can_attack,omitempty"`
	HasTapAbility  bool     `json:"has_tap_ability,omitempty"`
	Statuses       []string `json:"statuses,omitempty"`
	Text           string   `json:"text,omitempty"`
	// Opponents marks a card offered in a question as the opponent's.
	Opponents bool `json:"opponents,omitempty"`
}

// QuestionView is the question open for a live player.
type QuestionView struct {
	// Kind is what is asked, such as paying mana or blocking.
	Kind string `json:"kind,omitempty"`
	Text string `json:"text,omitempty"`
	// Source is the name of the card asking, when the player may know it.
	Source string `json:"source,omitempty"`
	// HowToAnswer says what answer the question takes.
	HowToAnswer string     `json:"how_to_answer"`
	Min         int        `json:"min_cards"`
	Max         int        `json:"max_cards"`
	Cancellable bool       `json:"cancellable"`
	Cards       []CardView `json:"cards,omitempty"`
	Choices     []string   `json:"choices,omitempty"`
	// Attack is the attack a block question is about.
	Attack *server.PromptAttack `json:"attack,omitempty"`
}

// situation writes the game as it stands, and takes what has happened since the
// last look. l.mu must be held.
func (l *Live) situation() Situation {
	shown := Situation{Status: l.status(), Result: l.result, Events: l.events, Refused: l.refused}
	l.events, l.refused = nil, nil
	if l.result != nil {
		l.reported = true
	}

	if state := l.view.state; state != nil {
		shown.Turn, shown.Step = state.Turn, state.Step
		shown.ManaCharged, shown.CanChargeMana = state.TurnManaCharged, state.CanChargeMana && !state.TurnManaCharged
		shown.You = l.seatView(&state.Me, state.MyTurn)
		shown.Opponent = l.seatView(&state.Opponent, false)
	}
	if q := l.view.question; q != nil {
		shown.Question = l.questionView(q)
	}
	return shown
}

func (l *Live) seatView(side *server.PlayerState, canAct bool) *SeatView {
	seat := &SeatView{
		Name:        side.Username,
		DeckCount:   side.Deck,
		HandCount:   side.HandCount,
		Battle:      make([]CardView, 0, len(side.Battlezone)),
		Mana:        make([]CardView, 0, len(side.Manazone)),
		ShieldCount: len(side.Shieldzone),
	}
	for _, card := range side.Hand {
		view := l.cardView(card)
		view.CanPlay = canAct && card.Flags&uint8(match.PlayableFlag) != 0
		seat.Hand = append(seat.Hand, view)
	}
	for _, card := range side.Battlezone {
		seat.Battle = append(seat.Battle, l.creatureView(side, card, canAct))
	}
	for _, card := range side.Manazone {
		view := l.cardView(card)
		seat.Mana = append(seat.Mana, CardView{Name: view.Name, Civilizations: view.Civilizations, Tapped: view.Tapped})
	}
	for _, card := range side.Graveyard {
		seat.Graveyard = append(seat.Graveyard, card.Name)
	}
	for _, shield := range side.Shieldzone {
		if card, ok := l.library.Card(shield.ImageID); ok {
			seat.FaceUpShields = append(seat.FaceUpShields, card.Name)
		}
	}
	return seat
}

func (l *Live) creatureView(side *server.PlayerState, card server.CardState, canAct bool) CardView {
	view := l.cardView(card)
	if power, ok := side.Powers[card.CardID]; ok {
		view.Power = power.Current
		if power.Attacking > power.Current {
			view.AttackingPower = power.Attacking
		}
	}
	for _, status := range side.Statuses[card.CardID] {
		view.Statuses = append(view.Statuses, status.ID)
	}
	view.CanAttack = canAct && !view.Tapped && card.Flags&uint8(match.SummonSickFlag) == 0
	view.HasTapAbility = card.Flags&uint8(match.TapAbilityFlag) != 0
	return view
}

// cardView describes a card from what is printed on it, with its rules text the
// first time it is seen. l.mu must be held.
func (l *Live) cardView(card server.CardState) CardView {
	view := CardView{ID: card.CardID, Name: card.Name, Tapped: card.Flags&uint8(match.TappedFlag) != 0}
	known, ok := l.library.Card(card.ImageID)
	if !ok {
		if view.Name == "" {
			view.Name = "face-down card"
		}
		return view
	}

	view.Name, view.Type, view.Civilizations = known.Name, known.Type, known.Civilizations
	view.Cost, view.Power = known.Cost, known.Power
	if !l.described[known.Name] {
		l.described[known.Name] = true
		view.Text = known.Text
	}
	return view
}

func (l *Live) questionView(q *question) *QuestionView {
	shown := &QuestionView{
		Kind:        q.about.Kind,
		Text:        q.text,
		HowToAnswer: howToAnswer(q),
		Min:         q.min,
		Max:         q.max,
		Cancellable: q.cancellable,
		Choices:     q.choices,
		Attack:      q.about.Attack,
	}
	if q.about.Source != nil {
		shown.Source = q.about.Source.Name
	}
	for _, card := range q.selectable() {
		view := l.cardView(card)
		view.Opponents = q.opponents(card.CardID)
		shown.Cards = append(shown.Cards, view)
	}
	return shown
}

// howToAnswer says what answer a question takes.
func howToAnswer(q *question) string {
	switch {
	case q.show:
		return "Read the cards in events, then answer with cancel to close them."
	case q.actionType == "question" && len(q.choices) == 0:
		return "Answer with no cards for yes, or with cancel for no."
	case q.actionType == "question" || q.actionType == "searchable":
		return "Answer with count, the position in choices counting from 0."
	case q.actionType == "count":
		return "Answer with count, a number from min_cards to max_cards."
	case q.actionType == "order":
		return "Answer with every card's id in cards, in the order you want them."
	}
	return "Answer with the ids of the cards you choose in cards, from min_cards to max_cards of them, or with cancel if cancellable."
}
