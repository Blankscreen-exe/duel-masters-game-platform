package httpapi

import (
	"log/slog"
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

// limitBodies makes reading a request body beyond limit bytes fail.
func limitBodies(limit int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}
