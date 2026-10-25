package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// gameGone is the answer for a game that is not running.
const gameGone = "This game is not running: it may have just ended."

func (h *handlers) GetLiveGame(ctx context.Context, request api.GetLiveGameRequestObject) (api.GetLiveGameResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetLiveGame401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	role, err := h.games.RoleIn(user.ID, request.GameId)
	switch {
	case errors.Is(err, games.ErrNotFound):
		return api.GetLiveGame404JSONResponse{NotFoundJSONResponse: notFound(gameGone)}, nil
	case errors.Is(err, games.ErrNotWatchable):
		return api.GetLiveGame403JSONResponse{NotWatchableJSONResponse: notWatchable()}, nil
	case err != nil:
		return nil, err
	}

	game, ok := h.games.Game(request.GameId)
	if !ok {
		return api.GetLiveGame404JSONResponse{NotFoundJSONResponse: notFound(gameGone)}, nil
	}

	return api.GetLiveGame200JSONResponse{
		Id:          game.ID,
		Name:        game.Name,
		FormatName:  game.FormatName,
		Host:        apiGamePlayer(game.Host),
		Guest:       apiGamePlayer(game.Guest),
		Spectators:  game.Watchable(),
		Matchmaking: game.Matchmaking,
		Role:        api.GameRole(role),
	}, nil
}

func (h *handlers) OpenGameSocket(ctx context.Context, request api.OpenGameSocketRequestObject) (api.OpenGameSocketResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.OpenGameSocket401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	// Checked before the upgrade, so a refusal is an answer the page can read.
	_, err = h.games.RoleIn(user.ID, request.GameId)
	switch {
	case errors.Is(err, games.ErrNotFound):
		return api.OpenGameSocket404JSONResponse{NotFoundJSONResponse: notFound(gameGone)}, nil
	case errors.Is(err, games.ErrNotWatchable):
		return api.OpenGameSocket403JSONResponse{NotWatchableJSONResponse: notWatchable()}, nil
	case err != nil:
		return nil, err
	}

	return gameConnection{request: requestFrom(ctx), upgrader: h.upgrader, games: h.games, user: user, gameID: request.GameId}, nil
}

// gameConnection answers by turning the request into a game's websocket and
// playing or watching over it until it closes.
type gameConnection struct {
	request  *http.Request
	upgrader *websocket.Upgrader
	games    *games.Service
	user     users.User
	gameID   string
}

func (c gameConnection) VisitOpenGameSocketResponse(w http.ResponseWriter) error {
	conn, err := c.upgrader.Upgrade(w, c.request, nil)
	if err != nil {
		// The upgrader has answered already, such as 403 for another site's
		// page.
		return nil
	}

	// The game may have ended since it was checked; Connect then closes the
	// connection, which is all the duel board needs to know.
	_ = c.games.Connect(conn, c.user, c.gameID)
	return nil
}

func notWatchable() api.NotWatchableJSONResponse {
	return api.NotWatchableJSONResponse(apiError(codeNotWatchable, "This game cannot be watched: it is not listed, or does not allow spectators."))
}
