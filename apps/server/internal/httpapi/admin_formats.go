package httpapi

import (
	"cmp"
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

const formatMissing = "There is no format with this id."

func (h *handlers) GetAdminFormats(ctx context.Context, _ api.GetAdminFormatsRequestObject) (api.GetAdminFormatsResponseObject, error) {
	_, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.GetAdminFormats401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.GetAdminFormats403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	list, err := h.formats.All(ctx)
	if err != nil {
		return nil, err
	}
	shown, err := h.apiAdminFormats(ctx, list)
	if err != nil {
		return nil, err
	}
	return api.GetAdminFormats200JSONResponse(shown), nil
}

func (h *handlers) GetAdminFormat(ctx context.Context, request api.GetAdminFormatRequestObject) (api.GetAdminFormatResponseObject, error) {
	outcome, err := h.manageFormat(ctx, func(users.User) ([]formats.Format, error) {
		format, err := h.formats.ByID(ctx, request.FormatId)
		return []formats.Format{format}, err
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.GetAdminFormat200JSONResponse(outcome.formats[0]), nil
	case http.StatusUnauthorized:
		return api.GetAdminFormat401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.GetAdminFormat403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.GetAdminFormat404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) CreateFormat(ctx context.Context, request api.CreateFormatRequestObject) (api.CreateFormatResponseObject, error) {
	outcome, err := h.manageFormat(ctx, func(actor users.User) ([]formats.Format, error) {
		format, err := h.formatManager.Create(ctx, actor, formatInput(*request.Body))
		return []formats.Format{format}, err
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.CreateFormat201JSONResponse(outcome.formats[0]), nil
	case http.StatusBadRequest:
		return api.CreateFormat400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.CreateFormat401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.CreateFormat403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.CreateFormat409JSONResponse{FormatNameTakenJSONResponse: api.FormatNameTakenJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) UpdateFormat(ctx context.Context, request api.UpdateFormatRequestObject) (api.UpdateFormatResponseObject, error) {
	outcome, err := h.manageFormat(ctx, func(actor users.User) ([]formats.Format, error) {
		format, err := h.formatManager.Update(ctx, actor, request.FormatId, formatInput(*request.Body))
		return []formats.Format{format}, err
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.UpdateFormat200JSONResponse(outcome.formats[0]), nil
	case http.StatusBadRequest:
		return api.UpdateFormat400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.UpdateFormat401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.UpdateFormat403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.UpdateFormat404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.UpdateFormat409JSONResponse{FormatNameTakenJSONResponse: api.FormatNameTakenJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) SetFormatRetired(ctx context.Context, request api.SetFormatRetiredRequestObject) (api.SetFormatRetiredResponseObject, error) {
	outcome, err := h.manageFormat(ctx, func(actor users.User) ([]formats.Format, error) {
		format, err := h.formatManager.SetRetired(ctx, actor, request.FormatId, request.Body.Retired)
		return []formats.Format{format}, err
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.SetFormatRetired200JSONResponse(outcome.formats[0]), nil
	case http.StatusUnauthorized:
		return api.SetFormatRetired401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.SetFormatRetired403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.SetFormatRetired404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.SetFormatRetired409JSONResponse{LastFormatJSONResponse: api.LastFormatJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) ReorderFormats(ctx context.Context, request api.ReorderFormatsRequestObject) (api.ReorderFormatsResponseObject, error) {
	outcome, err := h.manageFormat(ctx, func(actor users.User) ([]formats.Format, error) {
		return h.formatManager.Reorder(ctx, actor, request.Body.FormatIds)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.ReorderFormats200JSONResponse(outcome.formats), nil
	case http.StatusBadRequest:
		return api.ReorderFormats400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.ReorderFormats401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	default:
		return api.ReorderFormats403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	}
}

// formatOutcome is how an admin's request about formats went, for each handler
// to answer with its own response types: status 200 with the formats, or a
// refusal.
type formatOutcome struct {
	status  int
	formats []api.AdminFormat
	body    api.Error
}

// manageFormat carries out an admin's request and shows the formats as it left
// them, turning refusals into their answers.
func (h *handlers) manageFormat(ctx context.Context, act func(actor users.User) ([]formats.Format, error)) (formatOutcome, error) {
	actor, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return formatOutcome{}, err
	}
	if refusal != nil {
		return formatOutcome{status: refusal.status, body: refusal.body}, nil
	}

	changed, err := act(actor)
	if invalid, ok := invalidInput(err, nil); ok {
		return formatOutcome{status: http.StatusBadRequest, body: api.Error(invalid)}, nil
	}
	switch {
	case errors.Is(err, formats.ErrNotFound):
		return formatOutcome{status: http.StatusNotFound, body: api.Error(notFound(formatMissing))}, nil
	case errors.Is(err, formats.ErrNameTaken):
		return formatOutcome{status: http.StatusConflict, body: apiError(codeFormatNameTaken, "A format already has this name.")}, nil
	case errors.Is(err, formats.ErrLastFormat):
		return formatOutcome{status: http.StatusConflict, body: apiError(codeLastFormat, "This is the only format players can choose, so it cannot be retired. Add or bring back another first.")}, nil
	case errors.Is(err, users.ErrNotAdmin):
		return formatOutcome{status: http.StatusForbidden, body: api.Error(forbidden())}, nil
	case err != nil:
		return formatOutcome{}, err
	}

	shown, err := h.apiAdminFormats(ctx, changed)
	if err != nil {
		return formatOutcome{}, err
	}
	return formatOutcome{status: http.StatusOK, formats: shown}, nil
}

// formatInput is a format as the request gives it.
func formatInput(body api.FormatInput) formats.Input {
	limits := make(map[string]int, len(body.Limits))
	for _, limit := range body.Limits {
		limits[limit.CardId] = limit.MaxCopies
	}
	return formats.Input{
		Name:        body.Name,
		Description: body.Description,
		AllSets:     body.AllSets,
		Sets:        body.Sets,
		Limits:      limits,
	}
}

// apiAdminFormats are formats as the admin panel shows them, with their limited
// cards by name and the computer opponents offered in each.
func (h *handlers) apiAdminFormats(ctx context.Context, list []formats.Format) ([]api.AdminFormat, error) {
	var ids []string
	for _, format := range list {
		for id := range format.Limits {
			ids = append(ids, id)
		}
	}
	known, err := h.catalogue.Cards(ctx, ids)
	if err != nil {
		return nil, err
	}

	opponents, err := h.opponents.All(ctx)
	if err != nil {
		return nil, err
	}

	shown := make([]api.AdminFormat, 0, len(list))
	for _, format := range list {
		limits := make([]api.AdminFormatLimit, 0, len(format.Limits))
		for id, maxCopies := range format.Limits {
			limit := api.AdminFormatLimit{CardId: id, MaxCopies: maxCopies}
			if card, found := known[id]; found {
				one := apiCard(card)
				limit.Card = &one
			}
			limits = append(limits, limit)
		}
		slices.SortFunc(limits, func(a, b api.AdminFormatLimit) int {
			return cmp.Or(cmp.Compare(limitName(a), limitName(b)), cmp.Compare(a.CardId, b.CardId))
		})

		offered := []string{}
		for _, opponent := range opponents {
			if opponent.FormatID == format.ID && opponent.Enabled {
				offered = append(offered, opponent.Name)
			}
		}

		shown = append(shown, api.AdminFormat{
			Id:          format.ID,
			Name:        format.Name,
			Description: format.Description,
			AllSets:     format.AllSets,
			Sets:        format.Sets,
			Limits:      limits,
			Retired:     format.Retired,
			Opponents:   offered,
		})
	}
	return shown, nil
}

// limitName is what a limited card is listed by; a card gone from the card
// database has no name, so it comes first by its id.
func limitName(limit api.AdminFormatLimit) string {
	if limit.Card == nil {
		return ""
	}
	return limit.Card.Name
}
