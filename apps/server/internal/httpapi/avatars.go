package httpapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/avatars"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

func (h *handlers) ListAvatars(context.Context, api.ListAvatarsRequestObject) (api.ListAvatarsResponseObject, error) {
	ids := avatars.IDs()
	shown := make([]api.Avatar, 0, len(ids))
	for _, id := range ids {
		shown = append(shown, api.Avatar{Id: id, ImageUrl: avatarLocation(id)})
	}
	return api.ListAvatars200JSONResponse(shown), nil
}

func (h *handlers) GetAvatarImage(ctx context.Context, request api.GetAvatarImageRequestObject) (api.GetAvatarImageResponseObject, error) {
	picture, found := avatars.ByID(request.AvatarId)
	if !found {
		return api.GetAvatarImage404JSONResponse{NotFoundJSONResponse: notFound("No such picture.")}, nil
	}
	return servedAvatar{request: requestFrom(ctx), picture: picture}, nil
}

func (h *handlers) SetAvatar(ctx context.Context, request api.SetAvatarRequestObject) (api.SetAvatarResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.SetAvatar401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	err = h.users.SetAvatar(ctx, user.ID, request.Body.AvatarId)
	if invalid, ok := invalidInput(err, nil); ok {
		return api.SetAvatar400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	user.AvatarID = request.Body.AvatarId
	shown, err := h.account(ctx, user)
	if err != nil {
		return nil, err
	}
	return api.SetAvatar200JSONResponse(shown), nil
}

// avatarURL is where an account's picture is served, or nil while it has none,
// or has one no longer offered.
func avatarURL(avatarID string) *string {
	if _, found := avatars.ByID(avatarID); !found {
		return nil
	}
	location := avatarLocation(avatarID)
	return &location
}

func avatarLocation(avatarID string) string {
	return basePath + "/avatars/" + url.PathEscape(avatarID)
}

// servedAvatar answers with a picture the way servedFile does a file. A picture
// inside the program has no time it changed, so browsers check theirs by its
// version instead.
type servedAvatar struct {
	request *http.Request
	picture avatars.Image
}

func (a servedAvatar) VisitGetAvatarImageResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", pictureCacheControl)
	w.Header().Set("ETag", `"`+a.picture.Version+`"`)
	http.ServeContent(w, a.request, a.picture.ID, time.Time{}, bytes.NewReader(a.picture.Data))
	return nil
}
