package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/admin"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

const accountGone = "There is no account with this id."

// adminOutcome is how an admin's request went, for each handler to answer
// with its own response types: status 200 with the account, or a refusal.
type adminOutcome struct {
	status int
	user   api.AdminUser
	body   api.Error
}

func refused(status int, body api.Error) adminOutcome {
	return adminOutcome{status: status, body: body}
}

// currentAdmin returns the logged-in admin, or the refusal for anyone else:
// 401 for a visitor, 403 for a player. err is only for failures.
func (h *handlers) currentAdmin(ctx context.Context) (users.User, *adminOutcome, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		outcome := refused(http.StatusUnauthorized, api.Error(unauthenticated()))
		return users.User{}, &outcome, nil
	}
	if err != nil {
		return users.User{}, nil, err
	}
	if !user.IsAdmin() {
		outcome := refused(http.StatusForbidden, api.Error(forbidden()))
		return users.User{}, &outcome, nil
	}
	return user, nil, nil
}

// changeUser makes an admin's change to an account and shows the account as
// it left it, turning the guards' refusals into their answers. fields names
// the request's fields where they differ from the feature's.
func (h *handlers) changeUser(ctx context.Context, fields map[string]string, change func(actor users.User) (users.User, error)) (adminOutcome, error) {
	actor, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return adminOutcome{}, err
	}
	if refusal != nil {
		return *refusal, nil
	}

	changed, err := change(actor)
	if invalid, ok := invalidInput(err, fields); ok {
		return refused(http.StatusBadRequest, api.Error(invalid)), nil
	}
	if conflict, ok := adminConflict(err); ok {
		return refused(http.StatusConflict, conflict), nil
	}
	switch {
	case errors.Is(err, admin.ErrNotAdmin):
		return refused(http.StatusForbidden, api.Error(forbidden())), nil
	case errors.Is(err, users.ErrNotFound):
		return refused(http.StatusNotFound, api.Error(notFound(accountGone))), nil
	case err != nil:
		return adminOutcome{}, err
	}

	shown, err := h.adminUsers(ctx, []users.User{changed})
	if err != nil {
		return adminOutcome{}, err
	}
	return adminOutcome{status: http.StatusOK, user: shown[0]}, nil
}

// adminConflict turns a guard's refusal into the contract's answer.
func adminConflict(err error) (api.Error, bool) {
	switch {
	case errors.Is(err, admin.ErrOwnAccount):
		return apiError(codeOwnAccount, "You cannot do this to your own account."), true
	case errors.Is(err, admin.ErrAdminBan):
		return apiError(codeAdminBan, "An admin cannot be banned. Make them a player first."), true
	case errors.Is(err, admin.ErrBannedAdmin):
		return apiError(codeBannedAdmin, "A banned player cannot be made an admin. Lift the ban first."), true
	case errors.Is(err, admin.ErrLastAdmin):
		return apiError(codeLastAdmin, "The last admin cannot be made a player."), true
	case errors.Is(err, admin.ErrComputerAccount):
		return apiError(codeComputer, "A computer opponent's account cannot be changed."), true
	default:
		return api.Error{}, false
	}
}

func forbidden() api.ForbiddenJSONResponse {
	return api.ForbiddenJSONResponse(apiError(codeForbidden, "Only admins can do this."))
}

// minutes is a duration given in whole minutes, 0 for none.
func minutes(given *int) time.Duration {
	if given == nil {
		return 0
	}
	return time.Duration(*given) * time.Minute
}

// The request calls a ban or mute's duration durationMinutes.
var durationField = map[string]string{"duration": "durationMinutes"}

func (h *handlers) GetAdminUsers(ctx context.Context, request api.GetAdminUsersRequestObject) (api.GetAdminUsersResponseObject, error) {
	_, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.GetAdminUsers401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.GetAdminUsers403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	query := users.Query{Request: pageRequest(request.Params.Page, request.Params.PageSize)}
	if request.Params.Search != nil {
		query.Search = *request.Params.Search
	}
	if request.Params.Status != nil {
		query.Status = users.Status(*request.Params.Status)
	}
	page, err := h.users.Search(ctx, query, h.now())
	if invalid, ok := invalidInput(err, nil); ok {
		return api.GetAdminUsers400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	shown, err := h.adminUsers(ctx, page.Users)
	if err != nil {
		return nil, err
	}
	return api.GetAdminUsers200JSONResponse{Users: shown, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

func (h *handlers) BanUser(ctx context.Context, request api.BanUserRequestObject) (api.BanUserResponseObject, error) {
	input := admin.BanInput{Duration: minutes(request.Body.DurationMinutes), Reason: request.Body.Reason}
	outcome, err := h.changeUser(ctx, durationField, func(actor users.User) (users.User, error) {
		return h.admin.Ban(ctx, actor, request.UserId, input)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.BanUser200JSONResponse(outcome.user), nil
	case http.StatusBadRequest:
		return api.BanUser400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.BanUser401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.BanUser403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.BanUser404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.BanUser409JSONResponse{AdminConflictJSONResponse: api.AdminConflictJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) UnbanUser(ctx context.Context, request api.UnbanUserRequestObject) (api.UnbanUserResponseObject, error) {
	outcome, err := h.changeUser(ctx, nil, func(actor users.User) (users.User, error) {
		return h.admin.Unban(ctx, actor, request.UserId)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.UnbanUser200JSONResponse(outcome.user), nil
	case http.StatusUnauthorized:
		return api.UnbanUser401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.UnbanUser403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.UnbanUser404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.UnbanUser409JSONResponse{AdminConflictJSONResponse: api.AdminConflictJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) MuteUser(ctx context.Context, request api.MuteUserRequestObject) (api.MuteUserResponseObject, error) {
	duration := minutes(request.Body.DurationMinutes)
	outcome, err := h.changeUser(ctx, durationField, func(actor users.User) (users.User, error) {
		return h.admin.Mute(ctx, actor, request.UserId, duration)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.MuteUser200JSONResponse(outcome.user), nil
	case http.StatusBadRequest:
		return api.MuteUser400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.MuteUser401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.MuteUser403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.MuteUser404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.MuteUser409JSONResponse{AdminConflictJSONResponse: api.AdminConflictJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) UnmuteUser(ctx context.Context, request api.UnmuteUserRequestObject) (api.UnmuteUserResponseObject, error) {
	outcome, err := h.changeUser(ctx, nil, func(actor users.User) (users.User, error) {
		return h.admin.Unmute(ctx, actor, request.UserId)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.UnmuteUser200JSONResponse(outcome.user), nil
	case http.StatusUnauthorized:
		return api.UnmuteUser401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.UnmuteUser403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.UnmuteUser404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.UnmuteUser409JSONResponse{AdminConflictJSONResponse: api.AdminConflictJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) SetUserRole(ctx context.Context, request api.SetUserRoleRequestObject) (api.SetUserRoleResponseObject, error) {
	role := users.Role(request.Body.Role)
	if role != users.RolePlayer && role != users.RoleAdmin {
		field := "role"
		invalid := apiError(codeInvalidInput, "Role must be player or admin.")
		invalid.Field = &field
		return api.SetUserRole400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(invalid)}, nil
	}

	outcome, err := h.changeUser(ctx, nil, func(actor users.User) (users.User, error) {
		return h.admin.SetRole(ctx, actor, request.UserId, role)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.SetUserRole200JSONResponse(outcome.user), nil
	case http.StatusUnauthorized:
		return api.SetUserRole401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.SetUserRole403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.SetUserRole404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.SetUserRole409JSONResponse{AdminConflictJSONResponse: api.AdminConflictJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) SetUserPassword(ctx context.Context, request api.SetUserPasswordRequestObject) (api.SetUserPasswordResponseObject, error) {
	outcome, err := h.changeUser(ctx, nil, func(actor users.User) (users.User, error) {
		return h.admin.SetPassword(ctx, actor, request.UserId, request.Body.Password)
	})
	if err != nil {
		return nil, err
	}

	switch outcome.status {
	case http.StatusOK:
		return api.SetUserPassword204Response{}, nil
	case http.StatusBadRequest:
		return api.SetUserPassword400JSONResponse{InvalidInputJSONResponse: api.InvalidInputJSONResponse(outcome.body)}, nil
	case http.StatusUnauthorized:
		return api.SetUserPassword401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(outcome.body)}, nil
	case http.StatusForbidden:
		return api.SetUserPassword403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(outcome.body)}, nil
	case http.StatusNotFound:
		return api.SetUserPassword404JSONResponse{NotFoundJSONResponse: api.NotFoundJSONResponse(outcome.body)}, nil
	default:
		return api.SetUserPassword409JSONResponse{AdminConflictJSONResponse: api.AdminConflictJSONResponse(outcome.body)}, nil
	}
}

func (h *handlers) GetAdminLog(ctx context.Context, request api.GetAdminLogRequestObject) (api.GetAdminLogResponseObject, error) {
	_, refusal, err := h.currentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		if refusal.status == http.StatusUnauthorized {
			return api.GetAdminLog401JSONResponse{UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(refusal.body)}, nil
		}
		return api.GetAdminLog403JSONResponse{ForbiddenJSONResponse: api.ForbiddenJSONResponse(refusal.body)}, nil
	}

	query := adminlog.Query{Request: pageRequest(request.Params.Page, request.Params.PageSize)}
	if request.Params.Action != nil {
		query.Action = adminlog.Action(*request.Params.Action)
	}
	page, err := h.adminLog.List(ctx, query)
	if invalid, ok := invalidInput(err, nil); ok {
		return api.GetAdminLog400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	entries := make([]api.AdminLogEntry, 0, len(page.Entries))
	for _, entry := range page.Entries {
		shown := api.AdminLogEntry{
			Id:        entry.ID,
			AdminName: entry.AdminName,
			Action:    api.AdminLogAction(entry.Action),
			Subject:   entry.Subject,
			Details:   entry.Details,
			At:        entry.At.UTC(),
		}
		if entry.AdminID != "" {
			shown.AdminId = &entry.AdminID
		}
		entries = append(entries, shown)
	}
	return api.GetAdminLog200JSONResponse{Entries: entries, Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

// adminUsers shows accounts as the admin panel does, in the same order.
func (h *handlers) adminUsers(ctx context.Context, accounts []users.User) ([]api.AdminUser, error) {
	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.ID)
	}
	standings, err := h.ratings.Standings(ctx, ids)
	if err != nil {
		return nil, err
	}

	now := h.now()
	shown := make([]api.AdminUser, 0, len(accounts))
	for _, account := range accounts {
		standing, found := standings[account.ID]
		if !found {
			return nil, fmt.Errorf("reading %s's rank: %w", account.Username, ratings.ErrNotFound)
		}

		user := api.AdminUser{
			Id:            account.ID,
			Username:      account.Username,
			Email:         nonEmpty(account.Email),
			EmailVerified: account.IsVerified(),
			Role:          api.Role(account.Role),
			Computer:      account.Computer,
			JoinedAt:      account.CreatedAt.UTC(),
			Rank:          apiRank(standing.Rank),
			Wins:          standing.Wins,
			Losses:        standing.Losses,
			Online:        h.hub.IsOnline(account.ID),
			Mute:          apiMute(account, now),
		}
		if ban, banned := account.BanAt(now); banned {
			user.Ban = &api.Ban{Since: ban.Since.UTC(), Until: utcOrNil(ban.Until), Reason: ban.Reason}
		}
		shown = append(shown, user)
	}
	return shown, nil
}

// apiMute is an account's mute as the API shows it, or nil while it is not
// muted.
func apiMute(account users.User, now time.Time) *api.Penalty {
	mute, muted := account.MuteAt(now)
	if !muted {
		return nil
	}
	return &api.Penalty{Since: mute.Since.UTC(), Until: utcOrNil(mute.Until)}
}

func utcOrNil(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}
