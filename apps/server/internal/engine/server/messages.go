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

type DuelFinishedMessage struct {
	Header               string              `json:"header"`
	DuelID               string              `json:"duelId"`
	Winner               *DuelFinishedPlayer `json:"winner,omitempty"`
	MatchResultGenerated bool                `json:"matchResultGenerated"`
	WonByDisconnect      bool                `json:"wonByDisconnect"`
	Turns                int                 `json:"turns"`
	DurationSeconds      int64               `json:"durationSeconds"`
}

// ActionMessage is used to prompt the user to make a selection of the specified cards
type ActionMessage struct {
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

// ShowCardsMessage is used to show the user n cards without an action to perform
type ShowCardsMessage struct {
	Header  string   `json:"header"`
	Message string   `json:"message"`
	Cards   []string `json:"cards"`
}

type PlaySoundMessage struct {
	Header string `json:"header"`
	Sound  string `json:"sound"`
}
