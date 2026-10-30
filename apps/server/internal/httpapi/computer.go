package httpapi

import (
	"context"
	"errors"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
)

// opponentGone is the answer for a computer opponent that is not offered.
const opponentGone = "This computer opponent is not available any more."

func (h *handlers) ListComputerOpponents(ctx context.Context, _ api.ListComputerOpponentsRequestObject) (api.ListComputerOpponentsResponseObject, error) {
	if _, _, err := h.currentUser(ctx); errors.Is(err, auth.ErrNoSession) {
		return api.ListComputerOpponents401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}

	offers, err := h.challenges.Offers(ctx)
	if err != nil {
		return nil, err
	}

	shown := make(api.ListComputerOpponents200JSONResponse, 0, len(offers))
	for _, offer := range offers {
		shown = append(shown, api.ComputerOpponent{
			Id:            offer.ID,
			Name:          offer.Name,
			Description:   offer.Description,
			Style:         offer.Style,
			Strength:      api.ComputerOpponentStrength(offer.Strength),
			FormatId:      offer.FormatID,
			FormatName:    offer.FormatName,
			Civilizations: offer.Civilizations,
		})
	}
	return shown, nil
}

func (h *handlers) PlayComputerOpponent(ctx context.Context, request api.PlayComputerOpponentRequestObject) (api.PlayComputerOpponentResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.PlayComputerOpponent401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	game, err := h.challenges.Start(ctx, user, request.OpponentId, request.Body.DeckId)
	if refused, ok := deckOrInputRefusal(err); ok {
		return api.PlayComputerOpponent400JSONResponse{InvalidDeckOrInputJSONResponse: refused}, nil
	}
	if busy, ok := busyRefusal(err); ok {
		return api.PlayComputerOpponent409JSONResponse(busy), nil
	}
	switch {
	case err == nil:
		return api.PlayComputerOpponent201JSONResponse{GameId: game.ID}, nil
	case errors.Is(err, opponents.ErrNotFound):
		return api.PlayComputerOpponent404JSONResponse{NotFoundJSONResponse: notFound(opponentGone)}, nil
	case errors.Is(err, lobby.ErrPaused):
		return api.PlayComputerOpponent403JSONResponse{GamesPausedJSONResponse: gamesPaused()}, nil
	case errors.Is(err, games.ErrComputerGamesFull):
		return api.PlayComputerOpponent409JSONResponse(computerGamesFull()), nil
	case errors.Is(err, opponents.ErrUnavailable):
		return api.PlayComputerOpponent409JSONResponse(apiError(codeComputerUnavailable, "This computer opponent cannot play right now. Please choose another.")), nil
	default:
		return nil, err
	}
}

func computerGamesFull() api.Error {
	return apiError(codeComputerGamesFull, "The computer is playing as many games as it can right now. Please try again in a few minutes.")
}
