package httpapi

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/realtime"
)

// LiveUpdates passes on what the features report to the browsers, as the
// contract's RealtimeMessage types over the live connection. The features
// know nothing of the contract or of websockets; they only tell their
// listener what happened.
type LiveUpdates struct {
	hub *realtime.Hub
}

// NewLiveUpdates sends the updates over hub.
func NewLiveUpdates(hub *realtime.Hub) *LiveUpdates {
	return &LiveUpdates{hub: hub}
}

var _ chat.Listener = (*LiveUpdates)(nil)

// MessagePosted shows a new chat message to everyone.
func (l *LiveUpdates) MessagePosted(message chat.Message) {
	l.hub.Broadcast(api.RealtimeChatMessage{Type: api.RealtimeChatMessageTypeChatMessage, Message: apiChatMessage(message)})
}
