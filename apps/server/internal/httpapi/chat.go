package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

func (h *handlers) GetChat(ctx context.Context, _ api.GetChatRequestObject) (api.GetChatResponseObject, error) {
	recent, err := h.chat.Recent(ctx)
	if err != nil {
		return nil, err
	}

	shown := make([]api.ChatMessage, 0, len(recent))
	for _, message := range recent {
		shown = append(shown, apiChatMessage(message))
	}

	return api.GetChat200JSONResponse(shown), nil
}

func (h *handlers) PostChatMessage(ctx context.Context, request api.PostChatMessageRequestObject) (api.PostChatMessageResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.PostChatMessage401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	posted, err := h.chat.Post(ctx, user, request.Body.Text)
	var muted *chat.MutedError
	if errors.As(err, &muted) {
		return api.PostChatMessage403JSONResponse(mutedRefusal(muted.Mute)), nil
	}
	var tooFast *chat.TooFastError
	if errors.As(err, &tooFast) {
		seconds := tooFast.Seconds()
		return api.PostChatMessage429JSONResponse{TooFastJSONResponse: api.TooFastJSONResponse{
			Body:    apiError(codeTooFast, tooFast.Refusal()),
			Headers: api.TooFastResponseHeaders{RetryAfter: &seconds},
		}}, nil
	}
	if invalid, ok := invalidInput(err, nil); ok {
		return api.PostChatMessage400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.PostChatMessage201JSONResponse(apiChatMessage(posted)), nil
}

// mutedRefusal tells a muted player until when.
func mutedRefusal(mute users.Penalty) api.Error {
	if mute.Until == nil {
		return apiError(codeMuted, "An admin has muted you in the chat.")
	}
	until := mute.Until.UTC()
	refusal := apiError(codeMuted, "An admin has muted you in the chat until "+until.Format(time.RFC1123)+".")
	refusal.Until = &until
	return refusal
}

func apiChatMessage(message chat.Message) api.ChatMessage {
	return api.ChatMessage{Id: message.ID, Username: message.Username, Text: message.Text, PostedAt: message.PostedAt.UTC()}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
