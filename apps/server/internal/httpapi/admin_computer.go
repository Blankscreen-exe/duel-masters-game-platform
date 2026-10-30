package httpapi

import (
	"cmp"
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// An opponent's name follows a username's rules, under its own name here.
var opponentFields = map[string]string{"username": "name"}

const opponentMissing = "There is no computer opponent with this id."

func (h *handlers) GetAdminComputerOpponents(ctx context.Context, _ api.GetAdminComputerOpponentsRequestObject) (api.GetAdminComputerOpponentsResponseObject, error) {
	_, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.GetAdminComputerOpponents401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.GetAdminComputerOpponents403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	list, err := h.opponents.All(ctx)
	if err != nil {
		return nil, err
	}
	shown := make(api.GetAdminComputerOpponents200JSONResponse, 0, len(list))
	for _, opponent := range list {
		one, err := h.apiOpponent(ctx, opponent)
		if err != nil {
			return nil, err
		}
		shown = append(shown, one)
	}
	return shown, nil
}

func (h *handlers) GetComputerOpponentOptions(ctx context.Context, _ api.GetComputerOpponentOptionsRequestObject) (api.GetComputerOpponentOptionsResponseObject, error) {
	_, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.GetComputerOpponentOptions401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.GetComputerOpponentOptions403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	styles := make([]api.ComputerOpponentStyle, 0, len(computer.StyleNames()))
	for _, name := range computer.StyleNames() {
		style, _ := computer.StyleNamed(name)
		styles = append(styles, api.ComputerOpponentStyle{Name: name, Label: style.Label})
	}
	slices.SortFunc(styles, func(a, b api.ComputerOpponentStyle) int { return cmp.Compare(a.Label, b.Label) })

	return api.GetComputerOpponentOptions200JSONResponse{Styles: styles, CardTypes: opponents.CardTypes}, nil
}

func (h *handlers) GetAdminComputerOpponent(ctx context.Context, request api.GetAdminComputerOpponentRequestObject) (api.GetAdminComputerOpponentResponseObject, error) {
	outcome, err := h.manageOpponent(ctx, func(users.User) (opponents.Opponent, error) {
		return h.opponents.ByID(ctx, request.OpponentId)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.GetAdminComputerOpponent200JSONResponse(outcome.opponent), nil
	case http.StatusUnauthorized:
		return api.GetAdminComputerOpponent401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.GetAdminComputerOpponent403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.GetAdminComputerOpponent404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) CreateComputerOpponent(ctx context.Context, request api.CreateComputerOpponentRequestObject) (api.CreateComputerOpponentResponseObject, error) {
	outcome, err := h.manageOpponent(ctx, func(actor users.User) (opponents.Opponent, error) {
		return h.opponents.Create(ctx, actor, opponentInput(*request.Body))
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.CreateComputerOpponent201JSONResponse(outcome.opponent), nil
	case http.StatusBadRequest:
		return api.CreateComputerOpponent400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.CreateComputerOpponent401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.CreateComputerOpponent403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.CreateComputerOpponent409JSONResponse{ComputerOpponentNameTakenJSONResponse: api.ComputerOpponentNameTakenJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) UpdateComputerOpponent(ctx context.Context, request api.UpdateComputerOpponentRequestObject) (api.UpdateComputerOpponentResponseObject, error) {
	outcome, err := h.manageOpponent(ctx, func(actor users.User) (opponents.Opponent, error) {
		return h.opponents.Update(ctx, actor, request.OpponentId, opponentInput(*request.Body))
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.UpdateComputerOpponent200JSONResponse(outcome.opponent), nil
	case http.StatusBadRequest:
		return api.UpdateComputerOpponent400JSONResponse{InvalidDeckOrInputJSONResponse: api.InvalidDeckOrInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.UpdateComputerOpponent401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.UpdateComputerOpponent403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.UpdateComputerOpponent404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.UpdateComputerOpponent409JSONResponse{ComputerOpponentNameTakenJSONResponse: api.ComputerOpponentNameTakenJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) SetComputerOpponentOffered(ctx context.Context, request api.SetComputerOpponentOfferedRequestObject) (api.SetComputerOpponentOfferedResponseObject, error) {
	outcome, err := h.manageOpponent(ctx, func(actor users.User) (opponents.Opponent, error) {
		return h.opponents.SetEnabled(ctx, actor, request.OpponentId, request.Body.Offered)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.SetComputerOpponentOffered200JSONResponse(outcome.opponent), nil
	case http.StatusBadRequest:
		return api.SetComputerOpponentOffered400JSONResponse{InvalidDeckOrInputJSONResponse: api.InvalidDeckOrInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.SetComputerOpponentOffered401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.SetComputerOpponentOffered403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.SetComputerOpponentOffered404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	}
}

// opponentOutcome is how an admin's request about a computer opponent went, for
// each handler to answer with its own response types: status 200 with the
// opponent, or a refusal.
type opponentOutcome struct {
	status   int
	opponent api.AdminComputerOpponent
	body     api.Error
}

// manageOpponent carries out an admin's request and shows the opponent as it
// left it, turning refusals into their answers.
func (h *handlers) manageOpponent(ctx context.Context, act func(actor users.User) (opponents.Opponent, error)) (opponentOutcome, error) {
	actor, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return opponentOutcome{}, err
	}
	if refusal != nil {
		return opponentOutcome{status: refusal.status, body: refusal.body}, nil
	}

	opponent, err := act(actor)
	if invalid, ok := invalidInput(err, opponentFields); ok {
		return opponentOutcome{status: http.StatusBadRequest, body: api.Error(invalid)}, nil
	}
	var unplayable *opponents.UnplayableError
	switch {
	case errors.As(err, &unplayable):
		return opponentOutcome{status: http.StatusBadRequest, body: unplayableAnswer(unplayable)}, nil
	case errors.Is(err, opponents.ErrNotFound):
		return opponentOutcome{status: http.StatusNotFound, body: api.Error(notFound(opponentMissing))}, nil
	case errors.Is(err, users.ErrUsernameTaken):
		return opponentOutcome{status: http.StatusConflict, body: apiError(codeUsernameTaken, "An account already has this name.")}, nil
	case errors.Is(err, users.ErrNotAdmin):
		return opponentOutcome{status: http.StatusForbidden, body: api.Error(forbidden())}, nil
	case err != nil:
		return opponentOutcome{}, err
	}

	shown, err := h.apiOpponent(ctx, opponent)
	if err != nil {
		return opponentOutcome{}, err
	}
	return opponentOutcome{status: http.StatusOK, opponent: shown}, nil
}

// unplayableAnswer lists every way the deck breaks its rules.
func unplayableAnswer(unplayable *opponents.UnplayableError) api.Error {
	problems := make([]api.DeckProblem, 0, len(unplayable.Problems))
	reasons := make([]string, 0, len(unplayable.Problems))
	for _, problem := range unplayable.Problems {
		problems = append(problems, apiDeckProblem(problem))
		reasons = append(reasons, problem.Message())
	}

	answer := apiError(codeDeckNotLegal, unplayable.Name+"'s deck cannot be played, so players cannot be offered it: "+strings.Join(reasons, " "))
	answer.Problems = &problems
	return answer
}

// opponentInput is a computer opponent as the request gives it.
func opponentInput(body api.ComputerOpponentInput) opponents.Input {
	return opponents.Input{
		Name:        body.Name,
		Description: body.Description,
		Style:       body.Style,
		Strength:    computer.Strength(body.Strength),
		FormatID:    body.FormatId,
		Rules:       decks.Rules{MaxCivilizations: body.MaxCivilizations, CardTypes: body.CardTypes},
		Cards:       entriesOf(body.Cards),
	}
}

// entriesOf is a deck's cards as the request gives them.
func entriesOf(cards []api.DeckEntry) []decks.Entry {
	entries := make([]decks.Entry, 0, len(cards))
	for _, card := range cards {
		entries = append(entries, decks.Entry{CardID: card.CardId, Count: card.Count})
	}
	return entries
}

// apiOpponent is a computer opponent as the admin panel shows it, with whatever
// breaks its deck's rules.
func (h *handlers) apiOpponent(ctx context.Context, opponent opponents.Opponent) (api.AdminComputerOpponent, error) {
	format, err := h.formats.ByID(ctx, opponent.FormatID)
	if err != nil {
		return api.AdminComputerOpponent{}, err
	}
	known, err := h.catalogue.Cards(ctx, cardIDs(opponent.Cards))
	if err != nil {
		return api.AdminComputerOpponent{}, err
	}
	found, err := h.opponents.Check(ctx, opponent)
	if err != nil {
		return api.AdminComputerOpponent{}, err
	}

	problems := make([]api.DeckProblem, 0, len(found))
	for _, problem := range found {
		problems = append(problems, apiDeckProblem(problem))
	}

	return api.AdminComputerOpponent{
		Id:               opponent.ID,
		Name:             opponent.Name,
		Description:      opponent.Description,
		Style:            opponent.Style,
		Strength:         api.AdminComputerOpponentStrength(opponent.Strength),
		FormatId:         opponent.FormatID,
		FormatName:       format.Name,
		MaxCivilizations: opponent.Rules.MaxCivilizations,
		CardTypes:        opponent.Rules.CardTypes,
		Offered:          opponent.Enabled,
		Cards:            apiDeckCards(opponent.Cards, known, true),
		CardCount:        decks.Count(opponent.Cards),
		Civilizations:    decks.Civilizations(opponent.Cards, known),
		Problems:         problems,
	}, nil
}
