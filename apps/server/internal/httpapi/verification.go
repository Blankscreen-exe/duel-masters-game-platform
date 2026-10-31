package httpapi

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// nonEmpty is a text the API leaves out when it is empty.
func nonEmpty(text string) *string {
	if text == "" {
		return nil
	}
	return &text
}

func (h *handlers) GetVerification(ctx context.Context, _ api.GetVerificationRequestObject) (api.GetVerificationResponseObject, error) {
	user, err := h.unconfirmedUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetVerification401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.verification(ctx, user)
	if err != nil {
		return nil, err
	}
	return api.GetVerification200JSONResponse(shown), nil
}

func (h *handlers) ConfirmEmail(ctx context.Context, request api.ConfirmEmailRequestObject) (api.ConfirmEmailResponseObject, error) {
	user, err := h.unconfirmedUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.ConfirmEmail401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	confirmed, err := h.auth.ConfirmCode(ctx, user, request.Body.Code)
	switch {
	case err == nil:
		shown, err := h.account(ctx, confirmed)
		if err != nil {
			return nil, err
		}
		return api.ConfirmEmail200JSONResponse(shown), nil
	case errors.Is(err, users.ErrEmailConfirmed):
		return api.ConfirmEmail409JSONResponse{EmailConfirmedJSONResponse: api.EmailConfirmedJSONResponse(emailConfirmed())}, nil
	case errors.Is(err, auth.ErrWrongCode):
		return api.ConfirmEmail400JSONResponse(apiError(codeWrongCode, "That code is wrong. Check the email and try again.")), nil
	case errors.Is(err, auth.ErrCodeExpired):
		return api.ConfirmEmail400JSONResponse(apiError(codeCodeExpired, "That code no longer works. Ask for a new one.")), nil
	}

	if invalid, ok := invalidInput(err, nil); ok {
		return api.ConfirmEmail400JSONResponse(invalid), nil
	}

	return nil, err
}

func (h *handlers) SendVerificationCode(ctx context.Context, _ api.SendVerificationCodeRequestObject) (api.SendVerificationCodeResponseObject, error) {
	user, err := h.unconfirmedUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.SendVerificationCode401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	err = h.auth.SendCode(ctx, user)
	var (
		tooSoon *auth.ResendTooSoonError
		failed  *auth.MailFailedError
	)
	switch {
	case err == nil:
		shown, err := h.verification(ctx, user)
		if err != nil {
			return nil, err
		}
		return api.SendVerificationCode200JSONResponse(shown), nil
	case errors.Is(err, users.ErrEmailConfirmed):
		return api.SendVerificationCode409JSONResponse(emailConfirmed()), nil
	case errors.Is(err, auth.ErrMailUnavailable):
		return api.SendVerificationCode409JSONResponse(apiError(codeMailUnavailable, "The site cannot send email yet. An admin can let you in, or try again later.")), nil
	case errors.As(err, &tooSoon):
		return api.SendVerificationCode429JSONResponse{TooManyAttemptsJSONResponse: resendTooSoon(tooSoon)}, nil
	case errors.As(err, &failed):
		return api.SendVerificationCode502JSONResponse(apiError(codeMailFailed, "The code could not be sent. Try again in a moment.")), nil
	}

	return nil, err
}

func (h *handlers) ChangeVerificationEmail(ctx context.Context, request api.ChangeVerificationEmailRequestObject) (api.ChangeVerificationEmailResponseObject, error) {
	user, err := h.unconfirmedUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.ChangeVerificationEmail401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	changed, _, err := h.auth.ChangeEmail(ctx, user, request.Body.Email)
	var tooSoon *auth.ResendTooSoonError
	switch {
	case err == nil:
		shown, err := h.verification(ctx, changed)
		if err != nil {
			return nil, err
		}
		return api.ChangeVerificationEmail200JSONResponse(shown), nil
	case errors.Is(err, users.ErrEmailConfirmed):
		return api.ChangeVerificationEmail409JSONResponse(emailConfirmed()), nil
	case errors.Is(err, users.ErrEmailTaken):
		return api.ChangeVerificationEmail409JSONResponse(apiError(codeEmailTaken, "That email address is already in use.")), nil
	case errors.As(err, &tooSoon):
		return api.ChangeVerificationEmail429JSONResponse{TooManyAttemptsJSONResponse: resendTooSoon(tooSoon)}, nil
	}

	if invalid, ok := invalidInput(err, nil); ok {
		return api.ChangeVerificationEmail400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}

	return nil, err
}

func emailConfirmed() api.Error {
	return apiError(codeEmailConfirmed, "Your email address is already confirmed.")
}

// resendTooSoon answers with how long to wait, in whole seconds rounded up.
func resendTooSoon(err *auth.ResendTooSoonError) api.TooManyAttemptsJSONResponse {
	seconds := int(math.Ceil(err.RetryAfter.Seconds()))

	return api.TooManyAttemptsJSONResponse{
		Body:    apiError(codeTooManyAttempts, fmt.Sprintf("Wait %d %s before asking for another code.", seconds, plural(seconds, "second", "seconds"))),
		Headers: api.TooManyAttemptsResponseHeaders{RetryAfter: &seconds},
	}
}

// verification is how confirming an account's address stands, as the API shows
// it.
func (h *handlers) verification(ctx context.Context, user users.User) (api.Verification, error) {
	status, err := h.auth.VerificationOf(ctx, user)
	if err != nil {
		return api.Verification{}, err
	}

	return api.Verification{
		Email:           status.Email,
		MailWorks:       status.MailWorks,
		CodeSentAt:      utcOrNil(status.CodeSentAt),
		CodeExpiresAt:   utcOrNil(status.CodeExpiresAt),
		ResendInSeconds: int(math.Ceil(status.ResendIn.Seconds())),
		DeleteAt:        utcOrNil(status.DeleteAt),
	}, nil
}
