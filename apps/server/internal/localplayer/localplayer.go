// Package localplayer lets Claude Code, on the computer the server runs on, play
// the site's games: it offers the tools of the Model Context Protocol, over HTTP
// on a port only that computer can reach.
//
// The tools build the live opponent's deck and play its seat at the
// computer.Desk, which is told only what a player in that seat could know. Only
// the game has the last word on any move.
package localplayer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
)

// Path is where the tools are served.
const Path = "/mcp"

const (
	// longestWait is the most one tool call waits on the game. Waiting longer
	// is calling again, which tells the desk the player is still there.
	longestWait = 40 * time.Second
	// cardsPerPage is how many cards one search returns.
	cardsPerPage = 30
	// maxSaid is the longest line of chat, in bytes, as chat has its own limits
	// for what people type.
	maxSaid = 300
)

const instructions = `You are playing Duel Masters against the person you are talking to, on their own site. Play to win, and enjoy it.

Before a game: read list_formats and ask which format they want if they have not said, then build your own deck: find_cards to look through the cards the game can play in that format, then save_deck with exactly 40 cards. Build something with a real plan rather than a pile of good cards, and tell them the idea without giving away the list. Then tell them to pick Claude in the Vs computer tab of the Play page, and call wait_for_turn until the game starts.

During a game: call wait_for_turn whenever the status is opponent_to_move or no_game; it comes back after about 40 seconds if nothing has happened, so call it again. make_move, answer_question and coin_toss each return the game once it is next your move. Read refused in any result: it says why the game did not take something. A turn goes: charge mana once with a card from your hand, summon and cast with can_play cards, attack with can_attack creatures, then end_turn. Paying mana, choosing targets and blocking all arrive as questions; each says how to answer it. A card's rules text appears only the first time you see it in a game, so use find_cards if you need it again.

Play fair: know only what the tools tell you. Do not read the site's database, logs or the player's deck files, and do not look for the order of any deck. You may talk to the person here or with say in the game's chat, as you like, but keep the game moving.`

// Dependencies are what the tools need.
type Dependencies struct {
	Desk      *computer.Desk
	Opponents *opponents.Service
	Catalogue *cards.Catalogue
	Formats   *formats.Service
	Settings  *settings.Service
}

// tools are the tools' handlers.
type tools struct {
	Dependencies
}

// NewHandler returns the HTTP handler serving the tools at Path. Each request
// stands alone, so Claude Code carries on after the server restarts. The
// protocol's own protection refuses requests made through other sites' pages,
// and those naming any host but this computer.
func NewHandler(deps Dependencies) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "duel-masters", Version: "1.0.0"}, &mcp.ServerOptions{Instructions: instructions})
	t := &tools{Dependencies: deps}
	t.register(server)

	mux := http.NewServeMux()
	mux.Handle(Path, mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Stateless:             true,
		JSONResponse:          true,
		CrossOriginProtection: http.NewCrossOriginProtection(),
	}))
	return mux
}

// tool adds a tool whose result is written as JSON once. Nothing works while
// the setting is off, except dealing with a game already under way.
func tool[In any](server *mcp.Server, t *tools, name, description string, handler func(context.Context, In) (any, error)) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Description: description}, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, any, error) {
		if err := t.ensureOn(); err != nil {
			return nil, nil, err
		}
		t.Desk.Touch()

		result, err := handler(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		text, err := json.Marshal(result)
		if err != nil {
			return nil, nil, fmt.Errorf("writing the result: %w", err)
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(text)}}}, nil, nil
	})
}

// ensureOn returns what to tell the player while the setting is off.
func (t *tools) ensureOn() error {
	if t.Settings.Bool(opponents.LiveOpponent) {
		return nil
	}
	if _, inGame := t.Desk.Current(); inGame {
		return nil
	}
	return errors.New("Claude is switched off on this site: an admin can turn on " + opponents.LiveOpponent.Key + " in the admin panel's settings")
}

func (t *tools) register(server *mcp.Server) {
	tool(server, t, "status", "Whether a game is under way, and the deck and format you are offering. Use it first.", t.status)
	tool(server, t, "list_formats", "The formats games can be played in, with their rules.", t.listFormats)
	tool(server, t, "find_cards", "Search the cards the game can play, by words in their name or rules and by what they are. Also how to read a card's rules text again.", t.findCards)
	tool(server, t, "save_deck", "Make this your deck for the next game, in a format. It must be exactly 40 cards at most 4 of each, legal in the format; if it is not, it says what is wrong and nothing changes.", t.saveDeck)
	tool(server, t, "wait_for_turn", "Wait until it is your move or your game ends. Comes back after about 40 seconds if nothing has happened: call it again.", t.waitForTurn)
	tool(server, t, "coin_toss", "Answer the coin toss: heads or tails when the status is call_coin, first or second when it is choose_order.", t.coinToss)
	tool(server, t, "make_move", "Make a move of your own turn, when the status is your_turn. Returns the game once it is next your move.", t.makeMove)
	tool(server, t, "answer_question", "Answer the open question, when the status is question: the question says how. Returns the game once it is next your move.", t.answerQuestion)
	tool(server, t, "say", "Say something in the game's chat, which your opponent sees on the board.", t.say)
	tool(server, t, "resign", "Give up the game under way.", t.resign)
}

// statusResult is what status answers.
type statusResult struct {
	Game   computer.Situation `json:"game"`
	Format string             `json:"format,omitempty"`
	// Deck is your deck for the next game, by card name, and DeckProblems what
	// keeps it from being played.
	Deck         map[string]int `json:"deck,omitempty"`
	DeckProblems []string       `json:"deck_problems,omitempty"`
}

func (t *tools) status(ctx context.Context, _ struct{}) (any, error) {
	result := statusResult{Game: t.look()}

	live, err := t.Opponents.Live(ctx)
	if errors.Is(err, opponents.ErrNotFound) {
		return nil, errors.New("this site has no Claude opponent, as another account already has that name")
	}
	if err != nil {
		return nil, err
	}

	format, err := t.Formats.ByID(ctx, live.FormatID)
	if err != nil {
		return nil, err
	}
	result.Format = format.Name

	problems, err := t.Opponents.Check(ctx, live)
	if err != nil {
		return nil, err
	}
	result.DeckProblems = problemTexts(problems)

	ids := make([]string, 0, len(live.Cards))
	for _, entry := range live.Cards {
		ids = append(ids, entry.CardID)
	}
	known, err := t.Catalogue.Cards(ctx, ids)
	if err != nil {
		return nil, err
	}
	result.Deck = make(map[string]int, len(live.Cards))
	for _, entry := range live.Cards {
		result.Deck[known[entry.CardID].Name] = entry.Count
	}
	return result, nil
}

// look is the current game as it stands, if there is one.
func (t *tools) look() computer.Situation {
	game, ok := t.Desk.Current()
	if !ok {
		return computer.Situation{Status: computer.StatusNoGame}
	}
	return game.Situation()
}

func problemTexts(problems []decks.Problem) []string {
	texts := make([]string, 0, len(problems))
	for _, problem := range problems {
		texts = append(texts, problem.Message())
	}
	return texts
}

// formatView is a format as list_formats shows it.
type formatView struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	AllSets          bool     `json:"all_sets,omitempty"`
	Sets             []string `json:"sets,omitempty"`
	MinCivilizations int      `json:"min_civilizations,omitempty"`
	MaxCivilizations int      `json:"max_civilizations,omitempty"`
	// LimitedCards counts the cards it bans or limits, which find_cards keeps
	// to for bans, and save_deck for all of them.
	LimitedCards int `json:"limited_cards,omitempty"`
}

func (t *tools) listFormats(ctx context.Context, _ struct{}) (any, error) {
	list, err := t.Formats.List(ctx)
	if err != nil {
		return nil, err
	}
	shown := make([]formatView, 0, len(list))
	for _, format := range list {
		shown = append(shown, formatView{
			ID: format.ID, Name: format.Name, Description: format.Description, AllSets: format.AllSets, Sets: format.Sets,
			MinCivilizations: format.MinCivilizations, MaxCivilizations: format.MaxCivilizations, LimitedCards: len(format.Limits),
		})
	}
	return shown, nil
}

// findCardsInput is a card search.
type findCardsInput struct {
	Words         string   `json:"words,omitempty" jsonschema:"words to look for in the name and rules text"`
	FormatID      string   `json:"format_id,omitempty" jsonschema:"keep to the cards this format allows"`
	Civilizations []string `json:"civilizations,omitempty" jsonschema:"keep to cards of these: Light, Water, Darkness, Fire, Nature"`
	Types         []string `json:"types,omitempty" jsonschema:"keep to these: Creature, Evolution Creature, Spell"`
	Races         []string `json:"races,omitempty" jsonschema:"keep to creatures of these races"`
	Keywords      []string `json:"keywords,omitempty" jsonschema:"keep to cards with these abilities, such as blocker or shield_trigger"`
	MinCost       *int     `json:"min_cost,omitempty" jsonschema:"the lowest mana cost"`
	MaxCost       *int     `json:"max_cost,omitempty" jsonschema:"the highest mana cost"`
	Page          int      `json:"page,omitempty" jsonschema:"the page of results, from 1"`
}

// cardView is a card as find_cards shows it.
type cardView struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Cost          int      `json:"cost"`
	Power         string   `json:"power,omitempty"`
	Civilizations []string `json:"civilizations"`
	Races         []string `json:"races,omitempty"`
	Text          string   `json:"text,omitempty"`
}

// cardsFound is a page of find_cards.
type cardsFound struct {
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Pages int        `json:"pages"`
	Cards []cardView `json:"cards"`
}

func (t *tools) findCards(ctx context.Context, input findCardsInput) (any, error) {
	filter := cards.Filter{
		Query:         strings.TrimSpace(input.Words),
		Civilizations: input.Civilizations,
		Types:         input.Types,
		Races:         input.Races,
		Keywords:      input.Keywords,
		MinCost:       input.MinCost,
		MaxCost:       input.MaxCost,
		PlayableOnly:  true,
		Sort:          cards.SortByCost,
		Page:          input.Page,
		PageSize:      cardsPerPage,
	}
	if input.FormatID != "" {
		format, err := t.Formats.ByID(ctx, input.FormatID)
		if err != nil {
			return nil, err
		}
		legal := decks.LegalityIn(format)
		filter.Legal = &legal
	}

	page, err := t.Catalogue.Search(ctx, filter)
	if err != nil {
		return nil, err
	}

	found := cardsFound{Total: page.Total, Page: page.Page, Pages: (page.Total + cardsPerPage - 1) / cardsPerPage, Cards: make([]cardView, 0, len(page.Cards))}
	for _, card := range page.Cards {
		found.Cards = append(found.Cards, cardView{
			ID: card.ID, Name: card.Name, Type: card.Type, Cost: card.Cost, Power: card.PowerText,
			Civilizations: card.Civilizations, Races: card.Races, Text: card.RulesText,
		})
	}
	return found, nil
}

// deckEntry is how many copies of a card a deck has.
type deckEntry struct {
	CardID string `json:"card_id" jsonschema:"the card's id, from find_cards"`
	Count  int    `json:"count" jsonschema:"how many copies, from 1 to 4"`
}

// saveDeckInput is a deck to save.
type saveDeckInput struct {
	FormatID string      `json:"format_id" jsonschema:"the format it is for, from list_formats"`
	Cards    []deckEntry `json:"cards" jsonschema:"the deck, 40 cards in all"`
}

// deckSaved is what save_deck answers.
type deckSaved struct {
	Saved    bool     `json:"saved"`
	Problems []string `json:"problems,omitempty"`
}

func (t *tools) saveDeck(ctx context.Context, input saveDeckInput) (any, error) {
	entries := make([]decks.Entry, 0, len(input.Cards))
	for _, entry := range input.Cards {
		entries = append(entries, decks.Entry{CardID: entry.CardID, Count: entry.Count})
	}

	problems, err := t.Opponents.SaveLiveDeck(ctx, input.FormatID, entries)
	if err != nil {
		return nil, err
	}
	return deckSaved{Saved: len(problems) == 0, Problems: problemTexts(problems)}, nil
}

func (t *tools) waitForTurn(ctx context.Context, _ struct{}) (any, error) {
	started := time.Now()
	game, ok := t.Desk.WaitForGame(ctx, longestWait)
	if !ok {
		return computer.Situation{Status: computer.StatusNoGame}, nil
	}
	return game.Wait(ctx, longestWait-time.Since(started)), nil
}

// currentGame returns the game under way, or says there is none.
func (t *tools) currentGame() (*computer.Live, error) {
	game, ok := t.Desk.Current()
	if !ok {
		return nil, errors.New("you are not in a game")
	}
	return game, nil
}

// after waits for the game to come back to the player once they have acted.
func after(ctx context.Context, game *computer.Live, acted error) (any, error) {
	if acted != nil {
		return nil, acted
	}
	return game.Wait(ctx, longestWait), nil
}

// coinTossInput is an answer to the coin toss.
type coinTossInput struct {
	Choice string `json:"choice" jsonschema:"heads or tails when calling, first or second when choosing who plays first"`
}

func (t *tools) coinToss(ctx context.Context, input coinTossInput) (any, error) {
	game, err := t.currentGame()
	if err != nil {
		return nil, err
	}

	switch input.Choice {
	case "heads", "tails":
		return after(ctx, game, game.CallCoin(input.Choice == "heads"))
	case "first", "second":
		return after(ctx, game, game.ChooseOrder(input.Choice == "first"))
	}
	return nil, fmt.Errorf("the choice must be heads, tails, first or second, not %q", input.Choice)
}

// makeMoveInput is a move of the player's turn.
type makeMoveInput struct {
	Move string `json:"move" jsonschema:"one of charge_mana, play, attack_player, attack_creature, tap_ability, end_turn"`
	Card string `json:"card,omitempty" jsonschema:"the id of the card it is made with, from your hand or battle zone; none for end_turn"`
}

func (t *tools) makeMove(ctx context.Context, input makeMoveInput) (any, error) {
	game, err := t.currentGame()
	if err != nil {
		return nil, err
	}
	return after(ctx, game, game.Move(input.Move, input.Card))
}

// answerInput is an answer to a question.
type answerInput struct {
	Cards  []string `json:"cards,omitempty" jsonschema:"the ids of the cards chosen, as the question says"`
	Count  int      `json:"count,omitempty" jsonschema:"the number or position in choices, as the question says"`
	Cancel bool     `json:"cancel,omitempty" jsonschema:"cancel, which also answers no to a yes or no question"`
}

func (t *tools) answerQuestion(ctx context.Context, input answerInput) (any, error) {
	game, err := t.currentGame()
	if err != nil {
		return nil, err
	}
	return after(ctx, game, game.Answer(slices.Clone(input.Cards), input.Count, input.Cancel))
}

// sayInput is a line of chat.
type sayInput struct {
	Message string `json:"message" jsonschema:"what to say"`
}

func (t *tools) say(_ context.Context, input sayInput) (any, error) {
	game, err := t.currentGame()
	if err != nil {
		return nil, err
	}
	message := strings.TrimSpace(input.Message)
	if message == "" || len(message) > maxSaid {
		return nil, fmt.Errorf("the message must be 1 to %d characters", maxSaid)
	}
	game.Say(message)
	return struct {
		Said bool `json:"said"`
	}{true}, nil
}

func (t *tools) resign(ctx context.Context, _ struct{}) (any, error) {
	game, err := t.currentGame()
	if err != nil {
		return nil, err
	}
	game.Resign()
	return game.Wait(ctx, longestWait), nil
}
