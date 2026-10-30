package httpapi

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

func (h *handlers) ListFormats(ctx context.Context, _ api.ListFormatsRequestObject) (api.ListFormatsResponseObject, error) {
	list, err := h.formats.List(ctx)
	if err != nil {
		return nil, err
	}

	shown := make([]api.Format, 0, len(list))
	for _, format := range list {
		shown = append(shown, apiFormat(format))
	}

	return api.ListFormats200JSONResponse(shown), nil
}

func (h *handlers) ListDecks(ctx context.Context, _ api.ListDecksRequestObject) (api.ListDecksResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.ListDecks401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	list, err := h.decks.List(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	view, err := h.deckView(ctx, list...)
	if err != nil {
		return nil, err
	}

	shown := make([]api.DeckSummary, 0, len(list))
	for _, deck := range list {
		shown = append(shown, view.summary(deck))
	}

	return api.ListDecks200JSONResponse(shown), nil
}

func (h *handlers) CreateDeck(ctx context.Context, request api.CreateDeckRequestObject) (api.CreateDeckResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.CreateDeck401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	deck, err := h.decks.Create(ctx, user.ID, deckInput(*request.Body))
	var tooMany *decks.TooManyDecksError
	if errors.As(err, &tooMany) {
		return api.CreateDeck409JSONResponse(tooManyDecks(tooMany)), nil
	}
	if invalid, ok := invalidInput(err, nil); ok {
		return api.CreateDeck400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.showDeck(ctx, deck)
	if err != nil {
		return nil, err
	}

	return api.CreateDeck201JSONResponse(shown), nil
}

func (h *handlers) GetDeck(ctx context.Context, request api.GetDeckRequestObject) (api.GetDeckResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetDeck401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	deck, err := h.decks.Get(ctx, user.ID, request.DeckId)
	if errors.Is(err, decks.ErrNotFound) {
		return api.GetDeck404JSONResponse{NotFoundJSONResponse: notFound("No such deck.")}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.showDeck(ctx, deck)
	if err != nil {
		return nil, err
	}

	return api.GetDeck200JSONResponse(shown), nil
}

func (h *handlers) UpdateDeck(ctx context.Context, request api.UpdateDeckRequestObject) (api.UpdateDeckResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.UpdateDeck401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	deck, err := h.decks.Update(ctx, user.ID, request.DeckId, deckInput(*request.Body))
	if errors.Is(err, decks.ErrNotFound) {
		return api.UpdateDeck404JSONResponse{NotFoundJSONResponse: notFound("No such deck.")}, nil
	}
	if invalid, ok := invalidInput(err, nil); ok {
		return api.UpdateDeck400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.showDeck(ctx, deck)
	if err != nil {
		return nil, err
	}

	return api.UpdateDeck200JSONResponse(shown), nil
}

func (h *handlers) DeleteDeck(ctx context.Context, request api.DeleteDeckRequestObject) (api.DeleteDeckResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.DeleteDeck401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	err = h.decks.Delete(ctx, user.ID, request.DeckId)
	if errors.Is(err, decks.ErrNotFound) {
		return api.DeleteDeck404JSONResponse{NotFoundJSONResponse: notFound("No such deck.")}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.DeleteDeck204Response{}, nil
}

func (h *handlers) GetDeckText(ctx context.Context, request api.GetDeckTextRequestObject) (api.GetDeckTextResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetDeckText401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	deck, err := h.decks.Get(ctx, user.ID, request.DeckId)
	if errors.Is(err, decks.ErrNotFound) {
		return api.GetDeckText404JSONResponse{NotFoundJSONResponse: notFound("No such deck.")}, nil
	}
	if err != nil {
		return nil, err
	}

	known, err := h.catalogue.Cards(ctx, cardIDs(deck.Cards))
	if err != nil {
		return nil, err
	}

	return api.GetDeckText200TextResponse(decks.Text(deck.Cards, known)), nil
}

func (h *handlers) ReadDeckList(ctx context.Context, request api.ReadDeckListRequestObject) (api.ReadDeckListResponseObject, error) {
	if _, _, err := h.currentUser(ctx); errors.Is(err, auth.ErrNoSession) {
		return api.ReadDeckList401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}

	imported, err := h.decks.Import(ctx, request.Body.Text)
	if invalid, ok := invalidInput(err, nil); ok {
		return api.ReadDeckList400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	known, err := h.catalogue.Cards(ctx, cardIDs(imported.Cards))
	if err != nil {
		return nil, err
	}

	problems := make([]api.DeckListProblem, 0, len(imported.Problems))
	for _, problem := range imported.Problems {
		problems = append(problems, api.DeckListProblem{Line: problem.Number, Text: problem.Text, Message: problem.Reason})
	}

	return api.ReadDeckList200JSONResponse{Cards: apiDeckCards(imported.Cards, known, false), Problems: problems}, nil
}

// deckInput is a deck as the request gives it.
func deckInput(body api.DeckInput) decks.Input {
	input := decks.Input{Name: body.Name, Cards: entriesOf(body.Cards)}
	if body.CoverCardId != nil {
		input.CoverCardID = *body.CoverCardId
	}

	return input
}

// deckView is what showing decks needs besides the decks themselves: their
// cards from the catalogue and every format, to check them against.
type deckView struct {
	known   map[string]cards.Card
	formats []formats.Format
}

// deckView loads the cards of the given decks, and the formats.
func (h *handlers) deckView(ctx context.Context, shown ...decks.Deck) (deckView, error) {
	var ids []string
	for _, deck := range shown {
		ids = append(ids, cardIDs(deck.Cards)...)
	}

	known, err := h.catalogue.Cards(ctx, ids)
	if err != nil {
		return deckView{}, err
	}
	list, err := h.formats.List(ctx)
	if err != nil {
		return deckView{}, err
	}

	return deckView{known: known, formats: list}, nil
}

// showDeck is a deck as the API shows it, with its legality in every format.
func (h *handlers) showDeck(ctx context.Context, deck decks.Deck) (api.Deck, error) {
	view, err := h.deckView(ctx, deck)
	if err != nil {
		return api.Deck{}, err
	}

	return view.deck(deck), nil
}

func (v deckView) deck(deck decks.Deck) api.Deck {
	legality := make([]api.DeckLegality, 0, len(v.formats))
	for _, format := range v.formats {
		found := decks.Check(deck.Cards, v.known, format)
		problems := make([]api.DeckProblem, 0, len(found))
		for _, problem := range found {
			problems = append(problems, apiDeckProblem(problem))
		}
		legality = append(legality, api.DeckLegality{FormatId: format.ID, Legal: len(found) == 0, Problems: problems})
	}

	shown := api.Deck{
		Id:            deck.ID,
		Name:          deck.Name,
		Cover:         v.cover(deck),
		Cards:         apiDeckCards(deck.Cards, v.known, true),
		CardCount:     decks.Count(deck.Cards),
		Civilizations: decks.Civilizations(deck.Cards, v.known),
		Legality:      legality,
		CreatedAt:     deck.CreatedAt.UTC(),
		UpdatedAt:     deck.UpdatedAt.UTC(),
	}
	if deck.CoverCardID != "" {
		shown.CoverCardId = &deck.CoverCardID
	}

	return shown
}

func (v deckView) summary(deck decks.Deck) api.DeckSummary {
	legal := []string{}
	for _, format := range v.formats {
		if len(decks.Check(deck.Cards, v.known, format)) == 0 {
			legal = append(legal, format.ID)
		}
	}

	return api.DeckSummary{
		Id:             deck.ID,
		Name:           deck.Name,
		CardCount:      decks.Count(deck.Cards),
		Civilizations:  decks.Civilizations(deck.Cards, v.known),
		Cover:          v.cover(deck),
		LegalFormatIds: legal,
		UpdatedAt:      deck.UpdatedAt.UTC(),
	}
}

// cover is the card shown for a deck, or nil for an empty deck.
func (v deckView) cover(deck decks.Deck) *api.DeckCover {
	id := decks.Cover(deck.CoverCardID, deck.Cards, v.known)
	if id == "" {
		return nil
	}

	cover := &api.DeckCover{CardId: id}
	if card, found := v.known[id]; found && len(card.Printings) > 0 {
		cover.ImageUrl = pictureURL(card.Printings[0])
	}

	return cover
}

// apiDeckCards are a deck's cards as the API shows them, by cost and then
// name when sorted, otherwise in their own order.
func apiDeckCards(entries []decks.Entry, known map[string]cards.Card, sorted bool) []api.DeckCard {
	ordered := append([]decks.Entry(nil), entries...)
	if sorted {
		decks.SortForDisplay(ordered, known)
	}

	shown := make([]api.DeckCard, 0, len(ordered))
	for _, entry := range ordered {
		card := api.DeckCard{CardId: entry.CardID, Count: entry.Count}
		if found, ok := known[entry.CardID]; ok {
			details := apiCard(found)
			card.Card = &details
		}
		shown = append(shown, card)
	}

	return shown
}

func apiFormat(format formats.Format) api.Format {
	limits := make([]api.FormatCardLimit, 0, len(format.Limits))
	for cardID, maxCopies := range format.Limits {
		limits = append(limits, api.FormatCardLimit{CardId: cardID, MaxCopies: maxCopies})
	}
	slices.SortFunc(limits, func(a, b api.FormatCardLimit) int { return cmp.Compare(a.CardId, b.CardId) })

	return api.Format{
		Id:          format.ID,
		Name:        format.Name,
		Description: format.Description,
		AllSets:     format.AllSets,
		Sets:        format.Sets,
		Limits:      limits,
	}
}

// apiDeckProblem is a way a deck breaks a format, as the API shows it.
func apiDeckProblem(problem decks.Problem) api.DeckProblem {
	shown := api.DeckProblem{Code: api.DeckProblemCode(problem.Kind), Message: problem.Message()}
	if problem.CardID != "" {
		shown.CardId = &problem.CardID
	}
	return shown
}

// cardIDs are the ids of the entries' cards.
func cardIDs(entries []decks.Entry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.CardID)
	}
	return ids
}
