package httpapi

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
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
	var tooFast *chat.TooFastError
	if errors.As(err, &tooFast) {
		seconds := int(math.Ceil(tooFast.RetryAfter.Seconds()))
		return api.PostChatMessage429JSONResponse{TooFastJSONResponse: api.TooFastJSONResponse{
			Body:    apiError(codeTooFast, fmt.Sprintf("You are posting too fast. Wait %d %s.", seconds, plural(seconds, "second", "seconds"))),
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

func apiChatMessage(message chat.Message) api.ChatMessage {
	return api.ChatMessage{Id: message.ID, Username: message.Username, Text: message.Text, PostedAt: message.PostedAt.UTC()}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
