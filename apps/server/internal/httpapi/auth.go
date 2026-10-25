package httpapi

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

func (h *handlers) SignUp(ctx context.Context, request api.SignUpRequestObject) (api.SignUpResponseObject, error) {
	user, session, err := h.auth.SignUp(ctx, request.Body.Username, request.Body.Password)
	switch {
	case err == nil:
		cookie := h.sessionCookie(session)
		return api.SignUp201JSONResponse{Body: account(user), Headers: api.SignUp201ResponseHeaders{SetCookie: &cookie}}, nil
	case errors.Is(err, auth.ErrSignupsClosed):
		return api.SignUp403JSONResponse(apiError(codeSignupsClosed, "Sign-ups are closed at the moment.")), nil
	case errors.Is(err, users.ErrUsernameTaken):
		return api.SignUp409JSONResponse(apiError(codeUsernameTaken, "That username is already taken.")), nil
	}

	if invalid, ok := invalidInput(err, nil); ok {
		return api.SignUp400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}

	return nil, err
}

func (h *handlers) LogIn(ctx context.Context, request api.LogInRequestObject) (api.LogInResponseObject, error) {
	user, session, err := h.auth.LogIn(ctx, request.Body.Username, request.Body.Password)
	if err == nil {
		cookie := h.sessionCookie(session)
		return api.LogIn200JSONResponse{Body: account(user), Headers: api.LogIn200ResponseHeaders{SetCookie: &cookie}}, nil
	}

	if errors.Is(err, auth.ErrInvalidCredentials) {
		return api.LogIn401JSONResponse(apiError(codeInvalidCredentials, "Wrong username or password.")), nil
	}
	var tooMany *auth.TooManyAttemptsError
	if errors.As(err, &tooMany) {
		return api.LogIn429JSONResponse{TooManyAttemptsJSONResponse: tooManyAttempts(tooMany)}, nil
	}

	return nil, err
}

func (h *handlers) LogOut(ctx context.Context, _ api.LogOutRequestObject) (api.LogOutResponseObject, error) {
	if token := sessionFrom(ctx).token; token != "" {
		if err := h.auth.LogOut(ctx, token); err != nil {
			return nil, err
		}
	}

	cookie := h.clearedSessionCookie()
	return api.LogOut204Response{Headers: api.LogOut204ResponseHeaders{SetCookie: &cookie}}, nil
}

func (h *handlers) GetCurrentAccount(ctx context.Context, _ api.GetCurrentAccountRequestObject) (api.GetCurrentAccountResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetCurrentAccount401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.GetCurrentAccount200JSONResponse(account(user)), nil
}

func (h *handlers) ChangePassword(ctx context.Context, request api.ChangePasswordRequestObject) (api.ChangePasswordResponseObject, error) {
	user, token, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.ChangePassword401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	err = h.auth.ChangePassword(ctx, user, token, request.Body.CurrentPassword, request.Body.NewPassword)
	if err == nil {
		return api.ChangePassword204Response{}, nil
	}

	if errors.Is(err, auth.ErrWrongPassword) {
		return api.ChangePassword403JSONResponse(apiError(codeWrongPassword, "Your current password is wrong.")), nil
	}
	var tooMany *auth.TooManyAttemptsError
	if errors.As(err, &tooMany) {
		return api.ChangePassword429JSONResponse{TooManyAttemptsJSONResponse: tooManyAttempts(tooMany)}, nil
	}
	// The rules checked are the new password's; the request calls it
	// newPassword.
	if invalid, ok := invalidInput(err, map[string]string{"password": "newPassword"}); ok {
		return api.ChangePassword400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}

	return nil, err
}

// account is a user as the API shows it.
func account(user users.User) api.Account {
	return api.Account{Id: user.ID, Username: user.Username, Role: api.Role(user.Role), CreatedAt: user.CreatedAt.UTC()}
}

func unauthenticated() api.UnauthenticatedJSONResponse {
	return api.UnauthenticatedJSONResponse(apiError(codeUnauthenticated, "You are not logged in."))
}

// invalidInput turns a *validation.Error into the contract's answer, naming
// the field as the request does: fields maps the feature's field names to the
// request's where they differ.
func invalidInput(err error, fields map[string]string) (api.InvalidInputJSONResponse, bool) {
	invalid, ok := validation.As(err)
	if !ok {
		return api.InvalidInputJSONResponse{}, false
	}

	field := invalid.Field
	if renamed, ok := fields[field]; ok {
		field = renamed
	}

	answer := apiError(codeInvalidInput, fmt.Sprintf("%s %s.", displayName(invalid.Field), invalid.Reason))
	answer.Field = &field

	return api.InvalidInputJSONResponse(answer), true
}

// displayName is a field's name at the start of a sentence: "username" reads
// "Username".
func displayName(field string) string {
	if field == "" {
		return field
	}

	return strings.ToUpper(field[:1]) + field[1:]
}

// tooManyAttempts answers with how long to wait, rounded up to whole seconds
// in the Retry-After header and to whole minutes in the message.
func tooManyAttempts(err *auth.TooManyAttemptsError) api.TooManyAttemptsJSONResponse {
	seconds := int(math.Ceil(err.RetryAfter.Seconds()))
	minutes := (seconds + 59) / 60

	return api.TooManyAttemptsJSONResponse{
		Body:    apiError(codeTooManyAttempts, fmt.Sprintf("Too many failed attempts. Try again in %d %s.", minutes, plural(minutes, "minute", "minutes"))),
		Headers: api.TooManyAttemptsResponseHeaders{RetryAfter: &seconds},
	}
}
