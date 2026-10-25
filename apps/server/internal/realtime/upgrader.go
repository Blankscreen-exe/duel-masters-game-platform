package realtime

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gorilla/websocket"
)

// NewUpgrader returns what turns requests into websocket connections, for
// pages of the site only: a connection's Origin must be one of origins, such
// as "https://duelmasters.example". Browsers always send it, so a request
// without one is refused too. This is what stops another site's page from
// opening a connection with a player's login cookie.
func NewUpgrader(origins []string) *websocket.Upgrader {
	allowed := make([]string, 0, len(origins))
	for _, origin := range origins {
		allowed = append(allowed, strings.ToLower(strings.TrimRight(origin, "/")))
	}

	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.ToLower(r.Header.Get("Origin"))
			return origin != "" && slices.Contains(allowed, origin)
		},
	}
}
