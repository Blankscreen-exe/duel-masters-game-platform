package server

// Message is the default message struct
type Message struct {
	Header string `json:"header"`
}

// DecksMessage lists the users decks
type DeckMessage struct {
	UID      string   `json:"uid"`
	Owner    string   `json:"owner"`
	Name     string   `json:"name"`
	Public   bool     `json:"public"`
	Standard bool     `json:"standard"`
	Cards    []string `json:"cards"`
}

type DecksMessage struct {
	Header string        `json:"header"`
	Decks  []DeckMessage `json:"decks"`
}

// ChatMessage stores information about a chat message
type ChatMessage struct {
	Header  string `json:"header"`
	Message string `json:"message"`
	Sender  string `json:"sender"`
	Color   string `json:"color"`
	// Cards are the cards a game log line names, in the order it names them.
	Cards []ChatCard `json:"cards,omitempty"`
}

// ChatCard is a card a game log line names.
type ChatCard struct {
	// Name is the card's name as the line writes it.
	Name string `json:"name"`
	// ImageID is the card's uid, which shows it.
	ImageID string `json:"uid"`
	// Owner is the username of the player whose card it is, or empty when it
	// cannot be told which player's copy the line means.
	Owner string `json:"owner,omitempty"`
	// CardID is the card's virtualId while it is in a zone both players can
	// see, so it can be found on the board. Empty otherwise.
	CardID string `json:"virtualId,omitempty"`
}

// CardState stores information about the state of a card
type CardState struct {
	CardID  string `json:"virtualId"`
	ImageID string `json:"uid"`
	Name    string `json:"name"`
	Civ     string `json:"civilization"`
	Flags   uint8  `json:"flags"`
	// Type is "creature" or "spell", so the interface can name the action
	// that plays the card. Empty when the card is face down.
	Type string `json:"type"`
}

// ShieldState stores information about the state of a shield
type ShieldState struct {
	CardID  string `json:"virtualId"`
	ImageID string `json:"uid"`
	Flags   uint8  `json:"flags"`
}

// PlayerState stores information about the state of the current player
type PlayerState struct {
	Username   string         `json:"username"`
	Color      string         `json:"color"`
	Deck       int            `json:"deck"`
	HandCount  int            `json:"handCount"`
	Hand       []CardState    `json:"hand"`
	Shieldzone []ShieldState  `json:"shieldzone"`
	ShieldMap  map[string]int `json:"shieldMap"`
	Manazone   []CardState    `json:"manazone"`
	Graveyard  []CardState    `json:"graveyard"`
	Battlezone []CardState    `json:"playzone"`
	// Powers holds the power of each creature in the battle zone, by virtualId.
	Powers map[string]CreaturePower `json:"powers"`
	// Statuses holds the abilities and limits of each creature in the battle
	// zone, by virtualId.
	Statuses map[string][]CreatureStatus `json:"statuses"`
}

// CreatureStatus is an ability or limit a creature in the battle zone has
// right now.
type CreatureStatus struct {
	// ID names the status after the condition behind it, such as "blocker" or
	// "power_attacker".
	ID string `json:"id"`
	// Value is the amount a status carries, such as the power a power attacker
	// gains. Zero when it carries none.
	Value int `json:"value,omitempty"`
	// Granted is set when another card's effect gave the status, rather than
	// the creature itself or the rules.
	Granted bool `json:"granted,omitempty"`
}

// CreaturePower is a creature's power as the rules see it right now, which
// effects can raise or lower from the power printed on the card.
type CreaturePower struct {
	Printed int `json:"printed"`
	Current int `json:"current"`
	// Attacking is the power while the creature attacks, which effects such
	// as power attacker raise further.
	Attacking int `json:"attacking"`
}

// MatchState stores information about the current state of the match in the eyes of a given player
type MatchState struct {
	MyTurn        bool        `json:"myTurn"`
	HasAddedMana  bool        `json:"hasAddedManaThisRound"`
	HasAttacked   bool        `json:"hasAttackedThisRound"`
	CanChargeMana bool        `json:"canChargeManaThisRound"`
	Me            PlayerState `json:"me"`
	Opponent      PlayerState `json:"opponent"`
	Spectator     bool        `json:"spectator"`
	// Step is the current step of the turn, named by match.StepName.
	Step string `json:"step"`
	// BrokenShields lists, by virtualId, the shields broken since the last
	// state was sent, so the interface can show them shattering. Shields moved
	// into a hand by other effects are not broken and are not listed.
	BrokenShields []string `json:"brokenShields"`
	// ActivatedCards lists, by virtualId, the cards that acted since the last
	// state was sent: summoned, cast, attacked, blocked, used a tap ability, or
	// moved or destroyed another card with their effect. Only cards in zones
	// both players can see are listed.
	ActivatedCards []string `json:"activatedCards"`
	// TurnManaCharged is whether the player whose turn it is has put a card into
	// their mana zone this turn, which they may only do once.
	TurnManaCharged bool `json:"turnManaCharged"`
	// Attacker is the virtualId of the creature whose attack is under way, from
	// when the attack goes ahead until it has resolved. Empty otherwise.
	Attacker string `json:"attacker,omitempty"`
	// Turn is the number of the turn under way, counting both players' turns
	// and extra turns from 1. It is 0 before the first turn has begun.
	Turn int `json:"turn"`
}

// MatchStateMessage is the message that should be sent to the client for state updates
type MatchStateMessage struct {
	Header string     `json:"header"`
	State  MatchState `json:"state"`
}

// WarningMessage is used to send a warning to a player
type WarningMessage struct {
	Header  string `json:"header"`
	Message string `json:"message"`
}

type DuelFinishedPlayer struct {
	UID      string `json:"uid,omitempty"`
	Username string `json:"username,omitempty"`
}

// DuelFinishedMessage tells everyone in a match how the game ended.
type DuelFinishedMessage struct {
	Header string `json:"header"`
	DuelID string `json:"duelId"`
	// Winner is absent when there is none.
	Winner *DuelFinishedPlayer `json:"winner,omitempty"`
	// Ending is "play" (won by the game's rules), "surrender" or
	// "disconnect".
	Ending string `json:"ending"`
	// Counted is false for a game too short to count, or a practice game.
	Counted bool `json:"counted"`
	// Practice is set for a game that never counts, such as one against the
	// computer.
	Practice        bool  `json:"practice"`
	Turns           int   `json:"turns"`
	DurationSeconds int64 `json:"durationSeconds"`
}

// TossPlayer is a player taking part in the coin toss.
type TossPlayer struct {
	UID      string `json:"uid"`
	Username string `json:"username"`
}

// TossMessage is a step of the coin toss that decides who plays first, sent
// with the player whose turn it is: "choose_toss" asks them to call heads or
// tails and "toss_being_chosen" tells everyone else they are calling;
// "choose_turn" asks the toss's winner to play first or second and
// "turn_being_chosen" tells everyone else they are choosing.
type TossMessage struct {
	Header string     `json:"header"`
	Player TossPlayer `json:"player"`
}

// TossResultMessage tells everyone the call and how the coin landed, decided
// before any animation starts so every screen lands on the same side.
type TossResultMessage struct {
	Header string     `json:"header"`
	Caller TossPlayer `json:"caller"`
	// Call and Outcome are "heads" or "tails".
	Call    string     `json:"call"`
	Outcome string     `json:"outcome"`
	Winner  TossPlayer `json:"winner"`
}

// Prompt is the number of a question put to a player. They name it in their
// answer, so that an answer only ever settles the question it was given to.
type Prompt struct {
	ID int `json:"prompt,omitempty"`
}

// Number gives the question its number.
func (p *Prompt) Number(id int) {
	p.ID = id
}

// Prompting is a message that puts a question to a player.
type Prompting interface {
	Number(id int)
}

// PromptSource is the card whose effect asks a question.
type PromptSource struct {
	CardID  string `json:"virtualId"`
	ImageID string `json:"uid"`
	Name    string `json:"name"`
}

// PromptFighter is a creature in an attack, with its power in that attack.
type PromptFighter struct {
	Name  string `json:"name"`
	Power int    `json:"power"`
}

// PromptAttack is the attack a block question is about.
type PromptAttack struct {
	Attacker PromptFighter `json:"attacker"`
	// Target is the creature attacked; nil when the player is attacked.
	Target *PromptFighter `json:"target,omitempty"`
}

// PromptAbout says what a question is about, for programs that answer
// questions as a player does, such as the computer opponent, which cannot go
// by the words meant for people.
type PromptAbout struct {
	// Kind is what is asked, such as paying mana or blocking; "effect" for
	// a question a card's effect asks. See match.PromptKind.
	Kind string `json:"kind,omitempty"`
	// Source is the card whose effect asks, when the player asked may know
	// it: their own card, or one in a zone both players can see.
	Source *PromptSource `json:"source,omitempty"`
	// OpponentCards lists, by virtualId, the cards offered that belong to the
	// opponent of the player asked; the others are their own.
	OpponentCards []string `json:"opponentCards,omitempty"`
	// Attack is the attack a block question is about.
	Attack *PromptAttack `json:"attack,omitempty"`
}

// ActionMessage is used to prompt the user to make a selection of the specified cards
type ActionMessage struct {
	Prompt
	PromptAbout
	Header            string      `json:"header"`
	ActionType        string      `json:"actionType"`
	Cards             []CardState `json:"cards"`
	Text              string      `json:"text"`
	MinSelections     int         `json:"minSelections"`
	MaxSelections     int         `json:"maxSelections"`
	Cancellable       bool        `json:"cancellable"`
	UnselectableCards []CardState `json:"unselectableCards"`
	Choices           []string    `json:"choices"`
}

// MultipartActionMessage is used to prompt the user to make a selection of the specified cards
type MultipartActionMessage struct {
	Prompt
	PromptAbout
	Header        string                 `json:"header"`
	Cards         map[string][]CardState `json:"cards"`
	Text          string                 `json:"text"`
	MinSelections int                    `json:"minSelections"`
	MaxSelections int                    `json:"maxSelections"`
	Cancellable   bool                   `json:"cancellable"`
}

// ActionWarningMessage is used to apply an error
type ActionWarningMessage struct {
	Header  string `json:"header"`
	Message string `json:"message"`
}

// WaitMessage is used to send a waiting popup with a message
type WaitMessage struct {
	Header  string `json:"header"`
	Message string `json:"message"`
}

// ShowCardsMessage is used to show the user n cards without an action to
// perform. A preview the player must close is a question, and is numbered.
type ShowCardsMessage struct {
	Prompt
	Header  string   `json:"header"`
	Message string   `json:"message"`
	Cards   []string `json:"cards"`
}

type PlaySoundMessage struct {
	Header string `json:"header"`
	Sound  string `json:"sound"`
}
