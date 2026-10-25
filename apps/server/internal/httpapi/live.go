package httpapi

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/realtime"
)

// LiveUpdates passes on what the features report to the browsers, as the
// contract's RealtimeMessage types over the live connection, and tells the
// features who is online. The features know nothing of the contract or of
// websockets; they only tell their listener what happened.
type LiveUpdates struct {
	hub *realtime.Hub
}

// NewLiveUpdates sends the updates over hub.
func NewLiveUpdates(hub *realtime.Hub) *LiveUpdates {
	return &LiveUpdates{hub: hub}
}

var (
	_ chat.Listener   = (*LiveUpdates)(nil)
	_ lobby.Listener  = (*LiveUpdates)(nil)
	_ lobby.Directory = (*LiveUpdates)(nil)
)

// MessagePosted shows a new chat message to everyone.
func (l *LiveUpdates) MessagePosted(message chat.Message) {
	l.hub.Broadcast(api.RealtimeChatMessage{Type: api.RealtimeChatMessageTypeChatMessage, Message: apiChatMessage(message)})
}

// LobbyChanged shows players the lobby as it now is, and everyone the site's
// numbers.
func (l *LiveUpdates) LobbyChanged(snapshot lobby.Snapshot) {
	l.hub.BroadcastPlayers(api.RealtimeLobby{Type: api.RealtimeLobbyTypeLobby, Lobby: apiLobby(snapshot)})
	l.hub.Broadcast(api.RealtimeSiteStats{Type: api.RealtimeSiteStatsTypeSiteStats, Stats: apiSiteStats(snapshot)})
}

// GameStarted tells a game's two players to go to it, from the lobby or
// quick match.
func (l *LiveUpdates) GameStarted(game games.Game) {
	started := api.RealtimeGameStarted{Type: api.RealtimeGameStartedTypeGameStarted, GameId: game.ID}
	l.hub.SendTo(game.Host.ID, started)
	l.hub.SendTo(game.Guest.ID, started)
}

// OpenGameExpired tells a host nobody joined their game in time.
func (l *LiveUpdates) OpenGameExpired(open lobby.OpenGame) {
	l.hub.SendTo(open.Host.ID, api.RealtimeOpenGameExpired{Type: api.RealtimeOpenGameExpiredTypeOpenGameExpired, OpenGame: apiOpenGame(open)})
}

// QueueChanged shows players how many wait in each format for a quick match.
func (l *LiveUpdates) QueueChanged(waiting map[string]int) {
	l.hub.BroadcastPlayers(api.RealtimeQuickMatchQueue{Type: api.RealtimeQuickMatchQueueTypeQuickMatchQueue, Waiting: apiQueueCounts(waiting)})
}

// Online lists the players with the site open, by name.
func (l *LiveUpdates) Online() []lobby.Person {
	viewers := l.hub.Online()
	people := make([]lobby.Person, 0, len(viewers))
	for _, viewer := range viewers {
		people = append(people, lobby.Person{ID: viewer.UserID, Username: viewer.Username})
	}
	return people
}

// IsOnline reports whether a player has the site open.
func (l *LiveUpdates) IsOnline(userID string) bool {
	return l.hub.IsOnline(userID)
}
