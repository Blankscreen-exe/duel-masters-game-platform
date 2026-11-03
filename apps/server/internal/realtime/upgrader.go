package realtime

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// NewUpgrader returns what turns requests into websocket connections, for
// pages of the site only: a connection's Origin must be one of origins, such
// as "https://duelmasters.example". Browsers always send it, so a request
// without one is refused too. This is what stops another site's page from
// opening a connection with a player's login cookie.
//
// An origin whose host starts with "*." stands for any one subdomain there:
// "https://*.trycloudflare.com" allows "https://abc-def.trycloudflare.com"
// but not "https://trycloudflare.com" or "https://a.b.trycloudflare.com".
// That suits tunnels, whose address changes every time they start.
func NewUpgrader(origins []string) *websocket.Upgrader {
	allowed := make([]string, 0, len(origins))
	for _, origin := range origins {
		allowed = append(allowed, strings.ToLower(strings.TrimRight(origin, "/")))
	}

	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return originAllowed(strings.ToLower(r.Header.Get("Origin")), allowed)
		},
	}
}

// originAllowed reports whether origin, lower-cased, is one of allowed or
// matches one of its "*." entries.
func originAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, entry := range allowed {
		if strings.Contains(entry, "*") {
			if matchesWildcard(origin, entry) {
				return true
			}
		} else if origin == entry {
			return true
		}
	}
	return false
}

// matchesWildcard reports whether origin is entry with its "*" replaced by a
// single hostname label.
func matchesWildcard(origin, entry string) bool {
	scheme, host, found := strings.Cut(entry, "://")
	if !found || !strings.HasPrefix(host, "*.") {
		return false
	}
	prefix, suffix := scheme+"://", host[1:]
	if !strings.HasPrefix(origin, prefix) || !strings.HasSuffix(origin, suffix) {
		return false
	}
	label := origin[len(prefix) : len(origin)-len(suffix)]
	return isLabel(label)
}

// isLabel reports whether text is one hostname label: letters, digits and
// hyphens, nothing else.
func isLabel(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}
