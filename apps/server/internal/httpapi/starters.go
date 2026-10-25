package httpapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

const starterGone = "There is no starter deck with this id."

func (h *handlers) ListStarterDecks(ctx context.Context, _ api.ListStarterDecksRequestObject) (api.ListStarterDecksResponseObject, error) {
	if _, _, err := h.currentUser(ctx); errors.Is(err, auth.ErrNoSession) {
		return api.ListStarterDecks401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}

	starters, err := h.decks.Starters(ctx)
	if err != nil {
		return nil, err
	}
	view, err := h.deckView(ctx, starters...)
	if err != nil {
		return nil, err
	}

	shown := make([]api.DeckSummary, 0, len(starters))
	for _, deck := range starters {
		shown = append(shown, view.summary(deck))
	}
	return api.ListStarterDecks200JSONResponse(shown), nil
}

func (h *handlers) CopyStarterDeck(ctx context.Context, request api.CopyStarterDeckRequestObject) (api.CopyStarterDeckResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.CopyStarterDeck401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	deck, err := h.decks.CopyStarter(ctx, user.ID, request.DeckId)
	var tooMany *decks.TooManyDecksError
	switch {
	case errors.As(err, &tooMany):
		return api.CopyStarterDeck409JSONResponse(tooManyDecks(tooMany)), nil
	case errors.Is(err, decks.ErrNotFound):
		return api.CopyStarterDeck404JSONResponse{NotFoundJSONResponse: notFound(starterGone)}, nil
	case err != nil:
		return nil, err
	}

	shown, err := h.showDeck(ctx, deck)
	if err != nil {
		return nil, err
	}
	return api.CopyStarterDeck201JSONResponse(shown), nil
}

// tooManyDecks tells a player they have as many decks as allowed.
func tooManyDecks(err *decks.TooManyDecksError) api.Error {
	return apiError(codeTooManyDecks, fmt.Sprintf("You can have at most %d decks. Delete one to make room.", err.Limit))
}
