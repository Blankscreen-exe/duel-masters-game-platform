package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

// Error codes are part of the API: clients rely on them, so they never change.
const (
	codeBadRequest       = "bad_request"
	codeNotFound         = "not_found"
	codeMethodNotAllowed = "method_not_allowed"
	codeInternal         = "internal_error"
)

// writeError answers with an error status and the contract's Error body.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status is already sent, so a failed write cannot be reported to the
	// client, and the connection's own error ends the request.
	_ = json.NewEncoder(w).Encode(api.Error{Code: code, Message: message})
}
