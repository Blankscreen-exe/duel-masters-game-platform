package httpapi

import (
	"context"
	"log/slog"
	"mime"
	"net/http"
	"runtime/debug"
	"time"
)

// healthPath is the health check. Successful checks are logged at debug
// level, since Docker and load balancers make them all the time.
const healthPath = basePath + "/health"

// responseRecorder notes the status and size of a response for the request
// log. Unwrap lets http.ResponseController reach the underlying writer, for
// flushing, deadlines and hijacking.
type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(body)
	r.bytes += n
	return n, err
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// logRequests logs every request once it has been answered.
func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		level := slog.LevelInfo
		switch {
		case recorder.status >= http.StatusInternalServerError:
			level = slog.LevelWarn
		case r.URL.Path == healthPath && recorder.status == http.StatusOK:
			level = slog.LevelDebug
		}

		logger.LogAttrs(r.Context(), level, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Int("bytes", recorder.bytes),
			slog.Duration("duration", time.Since(started)),
		)
	})
}

// recoverPanics turns a panicking handler into a 500 answer and a logged
// error with its stack, instead of a dropped connection. http.ErrAbortHandler
// is passed on, since it is how a handler deliberately aborts a response.
func recoverPanics(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if recovered == http.ErrAbortHandler {
				panic(recovered)
			}

			logger.ErrorContext(r.Context(), "a request handler panicked",
				"method", r.Method,
				"path", r.URL.Path,
				"panic", recovered,
				"stack", string(debug.Stack()),
			)

			if recorder, ok := w.(*responseRecorder); ok && recorder.wroteHeader {
				// Part of the answer is already sent; abort the connection so
				// the client sees an error rather than a truncated success.
				panic(http.ErrAbortHandler)
			}
			writeError(w, http.StatusInternalServerError, codeInternal, "Something went wrong.")
		}()

		next.ServeHTTP(w, r)
	})
}

// requireJSON refuses requests that change something (POST, PUT, PATCH,
// DELETE) unless they are sent as JSON. A browser sends JSON to another site
// only after asking that site's permission, which this server never gives, so
// a page elsewhere cannot make a logged-in player's browser change anything
// here (cross-site request forgery). Reading (GET, HEAD) needs no body.
func requireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				writeError(w, http.StatusUnsupportedMediaType, codeUnsupportedMediaType, "Send this request as JSON, with Content-Type: application/json.")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

type requestContextKey struct{}

// keepRequest puts the request into its own context. Generated handlers only
// see the context, and answering with a file needs the request's conditional
// headers (If-Modified-Since, Range).
func keepRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestContextKey{}, r)))
	})
}

// requestFrom returns the request keepRequest put into ctx.
func requestFrom(ctx context.Context) *http.Request {
	request, _ := ctx.Value(requestContextKey{}).(*http.Request)
	return request
}

// limitBodies makes reading a request body beyond limit bytes fail.
func limitBodies(limit int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}
