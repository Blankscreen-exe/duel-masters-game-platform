package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/admin"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/settings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

const settingGone = "There is no setting with this key."

func (h *handlers) GetAdminSettings(ctx context.Context, _ api.GetAdminSettingsRequestObject) (api.GetAdminSettingsResponseObject, error) {
	_, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.GetAdminSettings401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.GetAdminSettings403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	entries := h.settings.List()
	shown := make(api.GetAdminSettings200JSONResponse, 0, len(entries))
	for _, entry := range entries {
		shown = append(shown, apiSetting(entry))
	}
	return shown, nil
}

func (h *handlers) SetAdminSetting(ctx context.Context, request api.SetAdminSettingRequestObject) (api.SetAdminSettingResponseObject, error) {
	outcome, err := h.changeSetting(ctx, func(actor users.User) (settings.Entry, error) {
		return h.admin.SetSetting(ctx, actor, request.Key, request.Body.Value)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.SetAdminSetting200JSONResponse(outcome.setting), nil
	case http.StatusBadRequest:
		return api.SetAdminSetting400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.SetAdminSetting401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.SetAdminSetting403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.SetAdminSetting404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) ResetAdminSetting(ctx context.Context, request api.ResetAdminSettingRequestObject) (api.ResetAdminSettingResponseObject, error) {
	outcome, err := h.changeSetting(ctx, func(actor users.User) (settings.Entry, error) {
		return h.admin.ResetSetting(ctx, actor, request.Key)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.ResetAdminSetting200JSONResponse(outcome.setting), nil
	case http.StatusUnauthorized:
		return api.ResetAdminSetting401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.ResetAdminSetting403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.ResetAdminSetting404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	}
}

// settingOutcome is how a change to a setting went, for each handler to answer
// with its own response types: status 200 with the setting, or a refusal.
type settingOutcome struct {
	status  int
	setting api.AdminSetting
	body    api.Error
}

// changeSetting makes an admin's change to a setting, shows it to everyone it
// concerns, and turns refusals into their answers.
func (h *handlers) changeSetting(ctx context.Context, change func(actor users.User) (settings.Entry, error)) (settingOutcome, error) {
	actor, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return settingOutcome{}, err
	}
	if refusal != nil {
		return settingOutcome{status: refusal.status, body: refusal.body}, nil
	}

	entry, err := change(actor)
	var invalid *settings.InvalidValueError
	switch {
	case errors.As(err, &invalid):
		field := "value"
		body := apiError(codeInvalidInput, "This value "+invalid.Reason+".")
		body.Field = &field
		return settingOutcome{status: http.StatusBadRequest, body: body}, nil
	case errors.Is(err, settings.ErrUnknownSetting):
		return settingOutcome{status: http.StatusNotFound, body: api.Error(notFound(settingGone))}, nil
	case errors.Is(err, admin.ErrNotAdmin):
		return settingOutcome{status: http.StatusForbidden, body: api.Error(forbidden())}, nil
	case err != nil:
		return settingOutcome{}, err
	}

	h.lobby.SettingsChanged()
	return settingOutcome{status: http.StatusOK, setting: apiSetting(entry)}, nil
}

// apiSetting is a setting as the admin panel shows it.
func apiSetting(entry settings.Entry) api.AdminSetting {
	shown := api.AdminSetting{
		Key:         entry.Key,
		Group:       entry.Group,
		Description: entry.Description,
		Kind:        api.AdminSettingKind(entry.Kind),
		Default:     entry.Default,
		Value:       entry.Value,
		IsDefault:   entry.IsDefault,
	}
	switch entry.Kind {
	case settings.KindNumber:
		shown.Min, shown.Max = &entry.Min, &entry.Max
	case settings.KindText:
		shown.MaxLength = &entry.MaxLength
	}
	if !entry.IsDefault {
		updatedAt := entry.UpdatedAt.UTC()
		shown.UpdatedAt = &updatedAt
	}
	return shown
}
