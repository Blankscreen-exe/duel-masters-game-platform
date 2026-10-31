package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// sessionCookieName names the session cookie, as the contract's sessionCookie
// security scheme does.
const sessionCookieName = "session"

type sessionContextKey struct{}

// requestSession is a request's session token, and the account it belongs to
// once a handler has asked for it.
type requestSession struct {
	token string

	once sync.Once
	user users.User
	err  error
}

// readSession puts the request's session token into its context. The account
// is looked up only when a handler asks for it, so requests that do not need
// one, such as health checks, cost no database query.
func readSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := &requestSession{}
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			session.token = cookie.Value
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionContextKey{}, session)))
	})
}

// sessionFrom returns the request's session, which readSession always sets.
func sessionFrom(ctx context.Context) *requestSession {
	if session, ok := ctx.Value(sessionContextKey{}).(*requestSession); ok {
		return session
	}

	return &requestSession{}
}

// currentUser returns the logged-in account and its session token. It returns
// auth.ErrNoSession when nobody is logged in.
func (h *handlers) currentUser(ctx context.Context) (users.User, string, error) {
	session := sessionFrom(ctx)
	session.once.Do(func() {
		session.user, session.err = h.auth.Authenticate(ctx, session.token)
	})

	return session.user, session.token, session.err
}

// unconfirmedUser returns the logged-in account, whether or not its email
// address is confirmed, for the few answers about confirming it. Everything
// else uses currentUser, which treats a player not yet confirmed as nobody.
func (h *handlers) unconfirmedUser(ctx context.Context) (users.User, error) {
	return h.auth.AuthenticateUnconfirmed(ctx, sessionFrom(ctx).token)
}

// sessionCookie sets the cookie holding a new session: sent with every request
// to the site, never readable by page scripts, and not sent along with
// requests other sites trigger.
func (h *handlers) sessionCookie(session auth.Session) string {
	return (&http.Cookie{
		Name:     sessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

// clearedSessionCookie removes the session cookie.
func (h *handlers) clearedSessionCookie() string {
	return (&http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	}).String()
}
