package httpapi

import (
	"context"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

// healthPingTimeout bounds the database check, so a hung database makes the
// health check fail rather than hang.
const healthPingTimeout = 2 * time.Second

func (h *handlers) GetHealth(ctx context.Context, _ api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	pingCtx, cancel := context.WithTimeout(ctx, healthPingTimeout)
	defer cancel()

	if err := h.db.PingContext(pingCtx); err != nil {
		return api.GetHealth503JSONResponse{Status: api.HealthStatusUnavailable, Database: api.HealthStatusUnavailable}, nil
	}

	return api.GetHealth200JSONResponse{Status: api.HealthStatusOk, Database: api.HealthStatusOk}, nil
}
