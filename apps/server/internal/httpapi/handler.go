// Package httpapi serves the HTTP API described by api/openapi.yaml. The
// routing and request/response types are generated into package api; this
// package implements the handlers and wraps them with the middleware every
// request passes through.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

// basePath is where the API is served, matching the contract's server URL.
const basePath = "/api"

// maxRequestBodyBytes bounds every request body.
const maxRequestBodyBytes = 1 << 20

// Pinger checks that the database answers; *sql.DB is one.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Dependencies are what the handlers need.
type Dependencies struct {
	DB     Pinger
	Logger *slog.Logger
}

// handlers implements the operations of the API contract.
type handlers struct {
	db Pinger
}

var _ api.StrictServerInterface = (*handlers)(nil)

// NewHandler returns the API: every operation of the contract under /api,
// JSON errors for unknown endpoints and methods, and the middleware that logs
// requests, recovers from panics and bounds request bodies.
func NewHandler(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	strict := api.NewStrictHandlerWithOptions(&handlers{db: deps.DB}, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: badRequest,
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			deps.Logger.ErrorContext(r.Context(), "handling a request failed", "method", r.Method, "path", r.URL.Path, "error", err)
			writeError(w, http.StatusInternalServerError, codeInternal, "Something went wrong.")
		},
	})
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{
		BaseURL:          basePath,
		BaseRouter:       mux,
		ErrorHandlerFunc: badRequest,
	})
	mux.HandleFunc(basePath+"/", unknownEndpoint(mux))

	return logRequests(deps.Logger, recoverPanics(deps.Logger, limitBodies(maxRequestBodyBytes, mux)))
}

func badRequest(w http.ResponseWriter, _ *http.Request, err error) {
	writeError(w, http.StatusBadRequest, codeBadRequest, err.Error())
}

// probedMethods are the methods checked when a request's method does not
// match its endpoint, to list the ones that do.
var probedMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

// unknownEndpoint answers requests no operation matches: 405 with the allowed
// methods when the path exists with another method, 404 otherwise.
func unknownEndpoint(mux *http.ServeMux) http.HandlerFunc {
	catchAll := basePath + "/"

	return func(w http.ResponseWriter, r *http.Request) {
		var allowed []string
		for _, method := range probedMethods {
			probe := r.Clone(r.Context())
			probe.Method = method
			if _, pattern := mux.Handler(probe); pattern != "" && pattern != catchAll {
				allowed = append(allowed, method)
			}
		}

		if len(allowed) > 0 {
			for _, method := range allowed {
				w.Header().Add("Allow", method)
			}
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "This endpoint does not accept "+r.Method+".")
			return
		}

		writeError(w, http.StatusNotFound, codeNotFound, "No such endpoint.")
	}
}
