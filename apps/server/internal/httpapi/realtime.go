package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/realtime"
)

func (h *handlers) OpenRealtime(ctx context.Context, _ api.OpenRealtimeRequestObject) (api.OpenRealtimeResponseObject, error) {
	viewer := realtime.Viewer{}
	user, _, err := h.currentUser(ctx)
	switch {
	case err == nil:
		viewer = realtime.Viewer{UserID: user.ID, Username: user.Username}
	case errors.Is(err, auth.ErrNoSession):
		// A visitor, who receives what everyone may see.
	default:
		return nil, err
	}

	return liveConnection{request: requestFrom(ctx), upgrader: h.upgrader, hub: h.hub, viewer: viewer}, nil
}

// liveConnection answers by turning the request into the live connection and
// keeping it until it closes.
type liveConnection struct {
	request  *http.Request
	upgrader *websocket.Upgrader
	hub      *realtime.Hub
	viewer   realtime.Viewer
}

func (c liveConnection) VisitOpenRealtimeResponse(w http.ResponseWriter) error {
	conn, err := c.upgrader.Upgrade(w, c.request, nil)
	if err != nil {
		// The upgrader has answered already, such as 403 for another site's
		// page; nothing more can be written.
		return nil
	}

	// A hub that has closed refuses the connection, closing it: the server
	// is stopping.
	_ = c.hub.Serve(conn, c.viewer)
	return nil
}
