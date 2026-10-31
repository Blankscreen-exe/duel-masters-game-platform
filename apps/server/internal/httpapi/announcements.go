package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/announcements"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

const announcementGone = "There is no notification with this id."

func (h *handlers) GetAnnouncements(ctx context.Context, request api.GetAnnouncementsRequestObject) (api.GetAnnouncementsResponseObject, error) {
	page, err := h.announcements.List(ctx, pageRequest(request.Params.Page, request.Params.PageSize))
	if invalid, ok := invalidInput(err, nil); ok {
		return api.GetAnnouncements400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.GetAnnouncements200JSONResponse{
		Announcements: apiAnnouncements(page.Announcements),
		Total:         page.Total,
		Page:          page.Page,
		PageSize:      page.PageSize,
	}, nil
}

func (h *handlers) GetMyAnnouncements(ctx context.Context, _ api.GetMyAnnouncementsRequestObject) (api.GetMyAnnouncementsResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetMyAnnouncements401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	mine, err := h.announcements.For(ctx, user)
	if err != nil {
		return nil, err
	}
	return api.GetMyAnnouncements200JSONResponse{Unread: mine.Unread, Popups: apiAnnouncements(mine.Popups)}, nil
}

func (h *handlers) MarkAnnouncementsSeen(ctx context.Context, _ api.MarkAnnouncementsSeenRequestObject) (api.MarkAnnouncementsSeenResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.MarkAnnouncementsSeen401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	if err := h.announcements.MarkSeen(ctx, user); err != nil {
		return nil, err
	}
	return api.MarkAnnouncementsSeen204Response{}, nil
}

func (h *handlers) DismissAnnouncement(ctx context.Context, request api.DismissAnnouncementRequestObject) (api.DismissAnnouncementResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.DismissAnnouncement401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	err = h.announcements.Dismiss(ctx, user, request.AnnouncementId)
	switch {
	case err == nil:
		return api.DismissAnnouncement204Response{}, nil
	case errors.Is(err, announcements.ErrNotFound):
		return api.DismissAnnouncement404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(notFound(announcementGone))}, nil
	default:
		return nil, err
	}
}

func (h *handlers) CreateAnnouncement(ctx context.Context, request api.CreateAnnouncementRequestObject) (api.CreateAnnouncementResponseObject, error) {
	outcome, err := h.manageAnnouncement(ctx, func(actor users.User) (announcements.Announcement, error) {
		return h.announcements.Create(ctx, actor, announcementInput(*request.Body))
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.CreateAnnouncement201JSONResponse(outcome.announcement), nil
	case http.StatusBadRequest:
		return api.CreateAnnouncement400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.CreateAnnouncement401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	default:
		return api.CreateAnnouncement403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) UpdateAnnouncement(ctx context.Context, request api.UpdateAnnouncementRequestObject) (api.UpdateAnnouncementResponseObject, error) {
	outcome, err := h.manageAnnouncement(ctx, func(actor users.User) (announcements.Announcement, error) {
		return h.announcements.Update(ctx, actor, request.AnnouncementId, announcementInput(*request.Body))
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.UpdateAnnouncement200JSONResponse(outcome.announcement), nil
	case http.StatusBadRequest:
		return api.UpdateAnnouncement400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.UpdateAnnouncement401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.UpdateAnnouncement403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.UpdateAnnouncement404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) DeleteAnnouncement(ctx context.Context, request api.DeleteAnnouncementRequestObject) (api.DeleteAnnouncementResponseObject, error) {
	outcome, err := h.manageAnnouncement(ctx, func(actor users.User) (announcements.Announcement, error) {
		return announcements.Announcement{}, h.announcements.Delete(ctx, actor, request.AnnouncementId)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.DeleteAnnouncement204Response{}, nil
	case http.StatusUnauthorized:
		return api.DeleteAnnouncement401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.DeleteAnnouncement403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	default:
		return api.DeleteAnnouncement404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	}
}

// announcementOutcome is how an admin's change went, for each handler to answer
// with its own response types: status 200 with the announcement, or a refusal.
type announcementOutcome struct {
	status       int
	announcement api.Announcement
	body         api.Error
}

// manageAnnouncement makes an admin's change, turning refusals into their
// answers.
func (h *handlers) manageAnnouncement(ctx context.Context, act func(actor users.User) (announcements.Announcement, error)) (announcementOutcome, error) {
	actor, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return announcementOutcome{}, err
	}
	if refusal != nil {
		return announcementOutcome{status: refusal.status, body: refusal.body}, nil
	}

	changed, err := act(actor)
	if invalid, ok := invalidInput(err, nil); ok {
		return announcementOutcome{status: http.StatusBadRequest, body: api.Error(invalid)}, nil
	}
	switch {
	case errors.Is(err, announcements.ErrNotFound):
		return announcementOutcome{status: http.StatusNotFound, body: api.Error(notFound(announcementGone))}, nil
	case errors.Is(err, users.ErrNotAdmin):
		return announcementOutcome{status: http.StatusForbidden, body: api.Error(forbidden())}, nil
	case err != nil:
		return announcementOutcome{}, err
	}

	return announcementOutcome{status: http.StatusOK, announcement: apiAnnouncement(changed)}, nil
}

func announcementInput(body api.AnnouncementInput) announcements.Input {
	input := announcements.Input{Title: body.Title, Body: body.Body, Important: body.Important}
	if body.ShowAgain != nil {
		input.ShowAgain = *body.ShowAgain
	}
	return input
}

func apiAnnouncement(announcement announcements.Announcement) api.Announcement {
	return api.Announcement{
		Id:        announcement.ID,
		Title:     announcement.Title,
		Body:      announcement.Body,
		Important: announcement.Important,
		PostedAt:  announcement.AnnouncedAt.UTC(),
	}
}

func apiAnnouncements(list []announcements.Announcement) []api.Announcement {
	shown := make([]api.Announcement, 0, len(list))
	for _, announcement := range list {
		shown = append(shown, apiAnnouncement(announcement))
	}
	return shown
}
