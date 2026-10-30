package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

// Error codes are part of the API: clients rely on them, so they never change.
const (
	codeBadRequest           = "bad_request"
	codeInvalidInput         = "invalid_input"
	codeNotFound             = "not_found"
	codeMethodNotAllowed     = "method_not_allowed"
	codeUnsupportedMediaType = "unsupported_media_type"
	codeInternal             = "internal_error"

	codeUnauthenticated    = "unauthenticated"
	codeInvalidCredentials = "invalid_credentials"
	codeTooManyAttempts    = "too_many_attempts"
	codeSignupsClosed      = "signups_closed"
	codeUsernameTaken      = "username_taken"
	codeWrongPassword      = "wrong_password"

	codeTooManyDecks = "too_many_decks"
	codeTooFast      = "too_fast"

	codeDeckNotLegal         = "deck_not_legal"
	codeGamesPaused          = "games_paused"
	codeBusy                 = "busy"
	codeNotHost              = "not_host"
	codeOwnGame              = "own_game"
	codeHostDeckNotLegal     = "host_deck_not_legal"
	codeOpponentDeckNotLegal = "opponent_deck_not_legal"
	codeNotWatchable         = "not_watchable"
	codeComputerGamesFull    = "computer_games_full"
	codeComputerUnavailable  = "computer_unavailable"

	codeBanned = "banned"
	codeMuted  = "muted"

	codeForbidden   = "forbidden"
	codeOwnAccount  = "own_account"
	codeAdminBan    = "admin_ban"
	codeBannedAdmin = "banned_admin"
	codeLastAdmin   = "last_admin"
	codeComputer    = "computer_account"
)

// apiError is the contract's Error body.
func apiError(code, message string) api.Error {
	return api.Error{Code: code, Message: message}
}

// writeError answers with an error status and the contract's Error body.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status is already sent, so a failed write cannot be reported to the
	// client, and the connection's own error ends the request.
	_ = json.NewEncoder(w).Encode(apiError(code, message))
}
