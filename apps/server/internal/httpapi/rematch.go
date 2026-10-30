package httpapi

import (
	"context"
	"errors"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/rematch"
)

// rematchGone is the answer for a rematch that is not, or no longer, being
// decided.
const rematchGone = "There is no rematch to decide: it was declined, a player left, or the offer ran out."

func (h *handlers) GetRematch(ctx context.Context, request api.GetRematchRequestObject) (api.GetRematchResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetRematch401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.rematches.Get(user.ID, request.GameId)
	if errors.Is(err, rematch.ErrNotFound) {
		return api.GetRematch404JSONResponse{NotFoundJSONResponse: notFound(rematchGone)}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.GetRematch200JSONResponse(apiRematch(shown, user.ID)), nil
}

func (h *handlers) ChooseRematchDeck(ctx context.Context, request api.ChooseRematchDeckRequestObject) (api.ChooseRematchDeckResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.ChooseRematchDeck401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.rematches.ChooseDeck(ctx, user, request.GameId, request.Body.DeckId)
	if refused, ok := deckOrInputRefusal(err); ok {
		return api.ChooseRematchDeck400JSONResponse{InvalidDeckOrInputJSONResponse: refused}, nil
	}
	if errors.Is(err, rematch.ErrNotFound) {
		return api.ChooseRematchDeck404JSONResponse{NotFoundJSONResponse: notFound(rematchGone)}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.ChooseRematchDeck200JSONResponse(apiRematch(shown, user.ID)), nil
}

func (h *handlers) AgreeToRematch(ctx context.Context, request api.AgreeToRematchRequestObject) (api.AgreeToRematchResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.AgreeToRematch401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	started, shown, err := h.rematches.Ready(ctx, user, request.GameId)
	if refused, ok := deckOrInputRefusal(err); ok {
		return api.AgreeToRematch400JSONResponse{InvalidDeckOrInputJSONResponse: refused}, nil
	}
	switch {
	case errors.Is(err, rematch.ErrNotFound):
		return api.AgreeToRematch404JSONResponse{NotFoundJSONResponse: notFound(rematchGone)}, nil
	case errors.Is(err, lobby.ErrPaused):
		return api.AgreeToRematch403JSONResponse{GamesPausedJSONResponse: gamesPaused()}, nil
	case errors.Is(err, rematch.ErrOpponentDeckIllegal):
		return api.AgreeToRematch409JSONResponse(apiError(codeOpponentDeckNotLegal, "Your opponent's deck no longer keeps to the format. Once they choose another, agree again.")), nil
	case errors.Is(err, games.ErrComputerGamesFull):
		return api.AgreeToRematch409JSONResponse(computerGamesFull()), nil
	case err != nil:
		return nil, err
	case started != nil:
		return api.AgreeToRematch201JSONResponse{GameId: started.ID}, nil
	default:
		return api.AgreeToRematch200JSONResponse(apiRematch(shown, user.ID)), nil
	}
}

func (h *handlers) DeclineRematch(ctx context.Context, request api.DeclineRematchRequestObject) (api.DeclineRematchResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.DeclineRematch401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	err = h.rematches.Decline(user.ID, request.GameId)
	switch {
	case err == nil:
		return api.DeclineRematch204Response{}, nil
	case errors.Is(err, rematch.ErrNotFound):
		return api.DeclineRematch404JSONResponse{NotFoundJSONResponse: notFound(rematchGone)}, nil
	default:
		return nil, err
	}
}

// apiRematch is a rematch as one of its players sees it: only their own
// deck's id is shown.
func apiRematch(shown rematch.Rematch, viewerID string) api.Rematch {
	own, opponent := shown.Seats(viewerID)

	you := apiRematchSeat(*own)
	you.Deck.Id = &own.Deck.ID

	answer := api.Rematch{
		GameId:     shown.GameID,
		Name:       shown.Name,
		FormatId:   shown.FormatID,
		FormatName: shown.FormatName,
		You:        you,
		Opponent:   apiRematchSeat(*opponent),
		ExpiresAt:  shown.ExpiresAt.UTC(),
	}
	if shown.CallerID != "" {
		answer.TossCallerId = &shown.CallerID
	}
	return answer
}

func apiRematchSeat(seat rematch.Seat) api.RematchSeat {
	return api.RematchSeat{
		Id:       seat.UserID,
		Username: seat.Username,
		Deck:     api.RematchDeck{Name: seat.Deck.Name, Civilizations: seat.Deck.Civilizations},
		Ready:    seat.Ready,
		Computer: seat.Computer,
	}
}
