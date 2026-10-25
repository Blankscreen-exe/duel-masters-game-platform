package httpapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/matchmaking"
)

func (h *handlers) GetSiteStats(context.Context, api.GetSiteStatsRequestObject) (api.GetSiteStatsResponseObject, error) {
	return api.GetSiteStats200JSONResponse(apiSiteStats(h.lobby.Snapshot())), nil
}

func (h *handlers) GetLobby(ctx context.Context, _ api.GetLobbyRequestObject) (api.GetLobbyResponseObject, error) {
	if _, _, err := h.currentUser(ctx); errors.Is(err, auth.ErrNoSession) {
		return api.GetLobby401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}

	return api.GetLobby200JSONResponse(apiLobby(h.lobby.Snapshot())), nil
}

func (h *handlers) OpenGame(ctx context.Context, request api.OpenGameRequestObject) (api.OpenGameResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.OpenGame401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	body := request.Body
	input := lobby.CreateInput{DeckID: body.DeckId, FormatID: body.FormatId, Visible: body.Listed, Spectators: body.Spectators}
	if body.Name != nil {
		input.Name = *body.Name
	}

	open, err := h.lobby.Create(ctx, user, input)
	if refused, ok := deckOrInputRefusal(err); ok {
		return api.OpenGame400JSONResponse{InvalidDeckOrInputJSONResponse: refused}, nil
	}
	if errors.Is(err, lobby.ErrPaused) {
		return api.OpenGame403JSONResponse{GamesPausedJSONResponse: gamesPaused()}, nil
	}
	if busy, ok := busyRefusal(err); ok {
		return api.OpenGame409JSONResponse{BusyJSONResponse: api.BusyJSONResponse(busy)}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.OpenGame201JSONResponse(apiOpenGame(open)), nil
}

func (h *handlers) GetOpenGame(ctx context.Context, request api.GetOpenGameRequestObject) (api.GetOpenGameResponseObject, error) {
	if _, _, err := h.currentUser(ctx); errors.Is(err, auth.ErrNoSession) {
		return api.GetOpenGame401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}

	open, err := h.lobby.OpenGame(request.OpenGameId)
	if errors.Is(err, lobby.ErrNotFound) {
		return api.GetOpenGame404JSONResponse{NotFoundJSONResponse: notFound(openGameGone)}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.GetOpenGame200JSONResponse(apiOpenGame(open)), nil
}

func (h *handlers) CancelOpenGame(ctx context.Context, request api.CancelOpenGameRequestObject) (api.CancelOpenGameResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.CancelOpenGame401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	err = h.lobby.Cancel(ctx, user, request.OpenGameId)
	switch {
	case err == nil:
		return api.CancelOpenGame204Response{}, nil
	case errors.Is(err, lobby.ErrNotFound):
		return api.CancelOpenGame404JSONResponse{NotFoundJSONResponse: notFound(openGameGone)}, nil
	case errors.Is(err, lobby.ErrNotHost):
		return api.CancelOpenGame403JSONResponse(apiError(codeNotHost, "Only the host can cancel the game.")), nil
	default:
		return nil, err
	}
}

func (h *handlers) JoinOpenGame(ctx context.Context, request api.JoinOpenGameRequestObject) (api.JoinOpenGameResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.JoinOpenGame401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	game, err := h.lobby.Join(ctx, user, request.OpenGameId, request.Body.DeckId)
	if refused, ok := deckOrInputRefusal(err); ok {
		return api.JoinOpenGame400JSONResponse{InvalidDeckOrInputJSONResponse: refused}, nil
	}
	if busy, ok := busyRefusal(err); ok {
		return api.JoinOpenGame409JSONResponse(busy), nil
	}
	switch {
	case err == nil:
		return api.JoinOpenGame201JSONResponse{GameId: game.ID}, nil
	case errors.Is(err, lobby.ErrNotFound):
		return api.JoinOpenGame404JSONResponse{NotFoundJSONResponse: notFound(openGameGone)}, nil
	case errors.Is(err, lobby.ErrPaused):
		return api.JoinOpenGame403JSONResponse{GamesPausedJSONResponse: gamesPaused()}, nil
	case errors.Is(err, lobby.ErrOwnGame):
		return api.JoinOpenGame409JSONResponse(apiError(codeOwnGame, "This is your own game; wait for an opponent to join it.")), nil
	case errors.Is(err, lobby.ErrHostDeckIllegal):
		return api.JoinOpenGame409JSONResponse(apiError(codeHostDeckNotLegal, "The host's deck no longer keeps to the format, so the game was closed.")), nil
	default:
		return nil, err
	}
}

func (h *handlers) GetQuickMatch(ctx context.Context, _ api.GetQuickMatchRequestObject) (api.GetQuickMatchResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetQuickMatch401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	status := api.GetQuickMatch200JSONResponse{Waiting: apiQueueCounts(h.quickMatch.Waiting())}
	if ticket, waiting := h.quickMatch.Ticket(user.ID); waiting {
		shown := apiTicket(ticket)
		status.Ticket = &shown
	}

	return status, nil
}

func (h *handlers) EnterQuickMatch(ctx context.Context, request api.EnterQuickMatchRequestObject) (api.EnterQuickMatchResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.EnterQuickMatch401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	entered, err := h.quickMatch.Enter(ctx, user, request.Body.FormatId, request.Body.DeckId)
	if refused, ok := deckOrInputRefusal(err); ok {
		return api.EnterQuickMatch400JSONResponse{InvalidDeckOrInputJSONResponse: refused}, nil
	}
	if errors.Is(err, lobby.ErrPaused) {
		return api.EnterQuickMatch403JSONResponse{GamesPausedJSONResponse: gamesPaused()}, nil
	}
	if busy, ok := busyRefusal(err); ok {
		return api.EnterQuickMatch409JSONResponse{BusyJSONResponse: api.BusyJSONResponse(busy)}, nil
	}
	if err != nil {
		return nil, err
	}

	if entered.Game != nil {
		return api.EnterQuickMatch200JSONResponse{GameId: &entered.Game.ID}, nil
	}
	ticket := apiTicket(entered.Ticket)
	return api.EnterQuickMatch200JSONResponse{Ticket: &ticket}, nil
}

func (h *handlers) LeaveQuickMatch(ctx context.Context, _ api.LeaveQuickMatchRequestObject) (api.LeaveQuickMatchResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.LeaveQuickMatch401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	h.quickMatch.Leave(user.ID)
	return api.LeaveQuickMatch204Response{}, nil
}

// openGameGone is the answer for an open game that does not exist or no
// longer does.
const openGameGone = "This game is no longer open: its host left, cancelled it, or someone else joined first."

func gamesPaused() api.GamesPausedJSONResponse {
	return api.GamesPausedJSONResponse(apiError(codeGamesPaused, "New games are paused for a moment. Please try again later."))
}

// deckOrInputRefusal turns a broken field or a deck breaking the format into
// the contract's answer.
func deckOrInputRefusal(err error) (api.InvalidDeckOrInputJSONResponse, bool) {
	if invalid, ok := invalidInput(err, nil); ok {
		return api.InvalidDeckOrInputJSONResponse(invalid), true
	}

	var illegal *games.IllegalDeckError
	if !errors.As(err, &illegal) {
		return api.InvalidDeckOrInputJSONResponse{}, false
	}

	problems := make([]api.DeckProblem, 0, len(illegal.Problems))
	reasons := make([]string, 0, len(illegal.Problems))
	for _, problem := range illegal.Problems {
		problems = append(problems, apiDeckProblem(problem))
		reasons = append(reasons, problem.Message())
	}

	message := "Your deck can no longer be played in this format."
	if len(reasons) > 0 {
		message = "Your deck cannot be played in this format: " + strings.Join(reasons, " ")
	}
	answer := apiError(codeDeckNotLegal, message)
	answer.Problems = &problems
	return api.InvalidDeckOrInputJSONResponse(answer), true
}

// busyRefusal says what a busy player is already doing.
func busyRefusal(err error) (api.Error, bool) {
	var busy *lobby.BusyError
	if !errors.As(err, &busy) {
		return api.Error{}, false
	}

	message := "You are already playing a game. Finish it first."
	switch busy.Current.Kind {
	case lobby.Hosting:
		message = "You are already hosting a game. Cancel it first."
	case lobby.Queued:
		message = fmt.Sprintf("You are already waiting for a quick match in %s. Leave it first.", busy.Current.FormatName)
	case lobby.Rematching:
		message = "You are deciding on a rematch. Accept or decline it first."
	}

	return apiError(codeBusy, message), true
}

func apiSiteStats(snapshot lobby.Snapshot) api.SiteStats {
	return api.SiteStats{PlayersOnline: len(snapshot.Online), GamesRunning: snapshot.GamesRunning}
}

func apiLobby(snapshot lobby.Snapshot) api.Lobby {
	open := make([]api.OpenGame, 0, len(snapshot.OpenGames))
	for _, game := range snapshot.OpenGames {
		open = append(open, apiOpenGame(game))
	}

	running := make([]api.RunningGame, 0, len(snapshot.Running))
	for _, game := range snapshot.Running {
		running = append(running, api.RunningGame{
			Id:          game.ID,
			Name:        game.Name,
			FormatId:    game.FormatID,
			FormatName:  game.FormatName,
			Host:        apiGamePlayer(game.Host),
			Guest:       apiGamePlayer(game.Guest),
			Spectators:  game.Watchable(),
			Matchmaking: game.Matchmaking,
			StartedAt:   game.StartedAt.UTC(),
		})
	}

	online := make([]api.OnlinePlayer, 0, len(snapshot.Online))
	for _, player := range snapshot.Online {
		shown := api.OnlinePlayer{Id: player.ID, Username: player.Username}
		if activity := player.Activity; activity != nil {
			shown.Activity = &api.PlayerActivity{Kind: api.PlayerActivityKind(activity.Kind)}
			if activity.GameID != "" {
				shown.Activity.GameId = &activity.GameID
			}
			if activity.FormatName != "" {
				shown.Activity.FormatName = &activity.FormatName
			}
		}
		online = append(online, shown)
	}

	return api.Lobby{OpenGames: open, RunningGames: running, Online: online, Paused: snapshot.Paused}
}

func apiOpenGame(open lobby.OpenGame) api.OpenGame {
	return api.OpenGame{
		Id:         open.ID,
		Name:       open.Name,
		Host:       api.LobbyPerson{Id: open.Host.ID, Username: open.Host.Username},
		FormatId:   open.FormatID,
		FormatName: open.FormatName,
		Listed:     open.Visible,
		Spectators: open.Spectators,
		CreatedAt:  open.CreatedAt.UTC(),
	}
}

func apiGamePlayer(player games.Player) api.GamePlayer {
	return api.GamePlayer{Id: player.ID, Username: player.Username, DeckName: player.DeckName}
}

func apiTicket(ticket matchmaking.Ticket) api.QuickMatchTicket {
	return api.QuickMatchTicket{FormatId: ticket.FormatID, FormatName: ticket.FormatName, DeckId: ticket.DeckID, Since: ticket.Since.UTC()}
}

// apiQueueCounts lists the formats with players waiting, by format id.
func apiQueueCounts(waiting map[string]int) []api.QueueCount {
	counts := make([]api.QueueCount, 0, len(waiting))
	for formatID, players := range waiting {
		counts = append(counts, api.QueueCount{FormatId: formatID, Players: players})
	}
	slices.SortFunc(counts, func(a, b api.QueueCount) int { return strings.Compare(a.FormatId, b.FormatId) })
	return counts
}
