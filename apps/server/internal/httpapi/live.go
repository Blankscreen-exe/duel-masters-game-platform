package httpapi

import (
	"context"
	"log/slog"
	"time"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/admin"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/chat"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/realtime"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/rematch"
)

// LiveUpdates passes on what the features report to the browsers, as the
// contract's RealtimeMessage types over the live connection, and tells the
// features who is online. The features know nothing of the contract or of
// websockets; they only tell their listener what happened.
type LiveUpdates struct {
	hub     *realtime.Hub
	ratings *ratings.Service
	logger  *slog.Logger
}

// rankTimeout bounds reading the ranks a live update shows, so a slow
// database delays an update only so long.
const rankTimeout = 5 * time.Second

// NewLiveUpdates sends the updates over hub, showing players' ranks from
// ratingService.
func NewLiveUpdates(hub *realtime.Hub, ratingService *ratings.Service, logger *slog.Logger) *LiveUpdates {
	return &LiveUpdates{hub: hub, ratings: ratingService, logger: logger}
}

var (
	_ admin.Listener   = (*LiveUpdates)(nil)
	_ chat.Listener    = (*LiveUpdates)(nil)
	_ games.Listener   = (*LiveUpdates)(nil)
	_ lobby.Listener   = (*LiveUpdates)(nil)
	_ lobby.Directory  = (*LiveUpdates)(nil)
	_ rematch.Listener = (*LiveUpdates)(nil)
)

// MessagePosted shows a new chat message to everyone.
func (l *LiveUpdates) MessagePosted(message chat.Message) {
	l.hub.Broadcast(api.RealtimeChatMessage{Type: api.RealtimeChatMessageTypeChatMessage, Message: apiChatMessage(message)})
}

// LobbyChanged shows players the lobby as it now is, and everyone the site's
// numbers. Should the players' ranks not be read, the lobby is shown without
// them rather than not at all.
func (l *LiveUpdates) LobbyChanged(snapshot lobby.Snapshot) {
	ctx, cancel := context.WithTimeout(context.Background(), rankTimeout)
	defer cancel()
	shownRanks, err := ranksOf(ctx, l.ratings, lobbyPlayers(snapshot))
	if err != nil {
		l.logger.Error("reading the ranks the lobby shows failed", "error", err)
	}

	l.hub.BroadcastPlayers(api.RealtimeLobby{Type: api.RealtimeLobbyTypeLobby, Lobby: apiLobby(snapshot, shownRanks)})
	l.hub.Broadcast(api.RealtimeSiteStats{Type: api.RealtimeSiteStatsTypeSiteStats, Stats: apiSiteStats(snapshot)})
}

// GameStarted tells a game's two players to go to it, from the lobby, quick
// match or a rematch.
func (l *LiveUpdates) GameStarted(game games.Game) {
	started := api.RealtimeGameStarted{Type: api.RealtimeGameStartedTypeGameStarted, GameId: game.ID}
	l.hub.SendTo(game.Host.ID, started)
	l.hub.SendTo(game.Guest.ID, started)
}

// OpenGameExpired tells a host nobody joined their game in time.
func (l *LiveUpdates) OpenGameExpired(open lobby.OpenGame) {
	l.hub.SendTo(open.Host.ID, api.RealtimeOpenGameExpired{Type: api.RealtimeOpenGameExpiredTypeOpenGameExpired, OpenGame: apiOpenGame(open, nil)})
}

// GameEnded is part of games.Listener; the players hear the game end from
// the game itself.
func (l *LiveUpdates) GameEnded(games.Ended) {}

// GameFinished tells a game's two players it is in the match history, with
// what it did to their ratings.
func (l *LiveUpdates) GameFinished(finished games.Record) {
	recorded := api.RealtimeMatchRecorded{Type: api.RealtimeMatchRecordedTypeMatchRecorded, Match: apiMatch(finished)}
	l.hub.SendTo(finished.Host.ID, recorded)
	l.hub.SendTo(finished.Guest.ID, recorded)
}

// GameClosed is part of games.Listener.
func (l *LiveUpdates) GameClosed(games.Game) {}

// AccountChanged tells a player an admin has changed their account, such as
// muting them, so their pages show it.
func (l *LiveUpdates) AccountChanged(userID string) {
	l.hub.SendTo(userID, api.RealtimeAccountChanged{Type: api.RealtimeAccountChangedTypeAccountChanged})
}

// QueueChanged shows players how many wait in each format for a quick match.
func (l *LiveUpdates) QueueChanged(waiting map[string]int) {
	l.hub.BroadcastPlayers(api.RealtimeQuickMatchQueue{Type: api.RealtimeQuickMatchQueueTypeQuickMatchQueue, Waiting: apiQueueCounts(waiting)})
}

// RematchChanged shows each of a rematch's players the rematch as they see it.
func (l *LiveUpdates) RematchChanged(shown rematch.Rematch) {
	for _, userID := range []string{shown.Host.UserID, shown.Guest.UserID} {
		l.hub.SendTo(userID, api.RealtimeRematch{Type: api.RealtimeRematchTypeRematch, Rematch: apiRematch(shown, userID)})
	}
}

// RematchEnded tells a rematch's players it will not be played, and why.
func (l *LiveUpdates) RematchEnded(shown rematch.Rematch, reason rematch.Reason, byUserID string) {
	ended := api.RealtimeRematchEnded{Type: api.RealtimeRematchEndedTypeRematchEnded, GameId: shown.GameID, Reason: api.RealtimeRematchEndedReason(reason)}
	for _, seat := range []rematch.Seat{shown.Host, shown.Guest} {
		if seat.UserID == byUserID {
			ended.By = &seat.Username
		}
	}

	l.hub.SendTo(shown.Host.UserID, ended)
	l.hub.SendTo(shown.Guest.UserID, ended)
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
