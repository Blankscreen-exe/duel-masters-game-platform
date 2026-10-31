package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/admin"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/mail"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

func (h *handlers) LetUserIn(ctx context.Context, request api.LetUserInRequestObject) (api.LetUserInResponseObject, error) {
	outcome, err := h.changeUser(ctx, nil, func(actor users.User) (users.User, error) {
		return h.admin.LetIn(ctx, actor, request.UserId)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.LetUserIn200JSONResponse(outcome.user), nil
	case http.StatusUnauthorized:
		return api.LetUserIn401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.LetUserIn403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.LetUserIn404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.LetUserIn409JSONResponse{AdminConflictJSONResponse: api.AdminConflictJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) GetMailSettings(ctx context.Context, _ api.GetMailSettingsRequestObject) (api.GetMailSettingsResponseObject, error) {
	_, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.GetMailSettings401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.GetMailSettings403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	shown, err := h.mailSettings(ctx)
	if err != nil {
		return nil, err
	}
	return api.GetMailSettings200JSONResponse(shown), nil
}

func (h *handlers) SetMailSettings(ctx context.Context, request api.SetMailSettingsRequestObject) (api.SetMailSettingsResponseObject, error) {
	actor, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.SetMailSettings401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.SetMailSettings403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	body := request.Body
	err = h.admin.SetMailSettings(ctx, actor, mail.Input{
		Host:        body.Host,
		Port:        body.Port,
		Security:    mail.Security(body.Security),
		Username:    body.Username,
		Password:    body.Password,
		FromAddress: body.FromAddress,
		FromName:    body.FromName,
	})
	switch {
	case errors.Is(err, mail.ErrNoSecretKey):
		return api.SetMailSettings409JSONResponse(apiError(codeNoSecretKey, "The server has no APP_SECRET_KEY to encrypt these with. Set it and restart the server.")), nil
	case errors.Is(err, admin.ErrNotAdmin):
		return api.SetMailSettings403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(forbidden())}, nil
	}
	if invalid, ok := invalidInput(err, nil); ok {
		return api.SetMailSettings400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.mailSettings(ctx)
	if err != nil {
		return nil, err
	}
	return api.SetMailSettings200JSONResponse(shown), nil
}

func (h *handlers) ClearMailSettings(ctx context.Context, _ api.ClearMailSettingsRequestObject) (api.ClearMailSettingsResponseObject, error) {
	actor, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.ClearMailSettings401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.ClearMailSettings403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	err = h.admin.ClearMailSettings(ctx, actor)
	if errors.Is(err, admin.ErrNotAdmin) {
		return api.ClearMailSettings403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(forbidden())}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.mailSettings(ctx)
	if err != nil {
		return nil, err
	}
	return api.ClearMailSettings200JSONResponse(shown), nil
}

func (h *handlers) SendTestMail(ctx context.Context, request api.SendTestMailRequestObject) (api.SendTestMailResponseObject, error) {
	_, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.SendTestMail401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.SendTestMail403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	if err := mail.CheckAddress(request.Body.To); err != nil {
		invalid, _ := invalidInput(validation.New("to", "must be an email address"), nil)
		return api.SendTestMail400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}

	err = h.mail.Send(ctx, request.Body.To, "Test email from Duel Masters", "The site can send email: this test came through the mail server set up in the admin panel.")
	switch {
	case err == nil:
		return api.SendTestMail204Response{}, nil
	case errors.Is(err, mail.ErrNotConfigured), errors.Is(err, mail.ErrUnreadable):
		return api.SendTestMail409JSONResponse(apiError(codeMailUnavailable, "No mail server is set up that can be used.")), nil
	default:
		// Only admins get here, and the reason is what they need to fix it. It
		// never holds the password.
		return api.SendTestMail502JSONResponse(apiError(codeMailFailed, err.Error())), nil
	}
}

// mailSettings is the mail server's settings as the API shows them.
func (h *handlers) mailSettings(ctx context.Context) (api.MailSettings, error) {
	status, err := h.mail.Status(ctx)
	if err != nil {
		return api.MailSettings{}, err
	}

	shown := api.MailSettings{
		Configured:   status.Configured,
		KeyAvailable: status.KeyAvailable,
		Unreadable:   status.Unreadable,
		HasPassword:  status.HasPassword,
		UpdatedAt:    utcOrNil(status.UpdatedAt),
	}
	if status.Configured {
		security := api.MailSecurity(status.Security)
		shown.Host = &status.Host
		shown.Port = &status.Port
		shown.Security = &security
		shown.Username = &status.Username
		shown.FromAddress = &status.FromAddress
		shown.FromName = &status.FromName
	}

	return shown, nil
}
