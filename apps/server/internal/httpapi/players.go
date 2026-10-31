package httpapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/auth"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// recentMatches is how many matches a profile shows.
const recentMatches = 5

const (
	playerGone = "There is no player with this name."
	matchGone  = "There is no finished game with this id: it may still be being saved."
)

func (h *handlers) GetRankings(ctx context.Context, request api.GetRankingsRequestObject) (api.GetRankingsResponseObject, error) {
	user, _, err := h.currentUser(ctx)
	if errors.Is(err, auth.ErrNoSession) {
		return api.GetRankings401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	}
	if err != nil {
		return nil, err
	}

	query := ratings.RankingsQuery{Request: pageRequest(request.Params.Page, request.Params.PageSize)}
	if request.Params.Search != nil {
		query.Search = *request.Params.Search
	}
	rankings, err := h.ratings.Rankings(ctx, query)
	if invalid, ok := invalidInput(err, nil); ok {
		return api.GetRankings400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	players := make([]api.RankedPlayer, 0, len(rankings.Players))
	for _, player := range rankings.Players {
		players = append(players, apiRankedPlayer(player))
	}
	shown := api.GetRankings200JSONResponse{Players: players, Total: rankings.Total, Page: rankings.Page, PageSize: rankings.PageSize}

	standing, err := h.ratings.Standing(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	place, err := h.ratings.Place(ctx, standing)
	if err != nil {
		return nil, err
	}
	if place > 0 {
		you := apiRankedPlayer(ratings.Placed{Standing: standing, Place: place})
		shown.You = &you
	}

	return shown, nil
}

func (h *handlers) GetPlayer(ctx context.Context, request api.GetPlayerRequestObject) (api.GetPlayerResponseObject, error) {
	if _, _, err := h.currentUser(ctx); errors.Is(err, auth.ErrNoSession) {
		return api.GetPlayer401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}

	player, err := h.person(ctx, request.Username)
	if errors.Is(err, users.ErrNotFound) {
		return api.GetPlayer404JSONResponse{NotFoundJSONResponse: notFound(playerGone)}, nil
	}
	if err != nil {
		return nil, err
	}

	standing, err := h.ratings.Standing(ctx, player.ID)
	if err != nil {
		return nil, err
	}
	place, err := h.ratings.Place(ctx, standing)
	if err != nil {
		return nil, err
	}
	favourites, err := h.games.FavouriteCivilizations(ctx, player.ID)
	if err != nil {
		return nil, err
	}
	recent, err := h.games.History(ctx, player.ID, paging.Request{PageSize: recentMatches})
	if err != nil {
		return nil, err
	}

	profile := api.GetPlayer200JSONResponse{
		Id:                     player.ID,
		Username:               player.Username,
		JoinedAt:               player.CreatedAt.UTC(),
		Rank:                   apiRank(standing.Rank),
		Wins:                   standing.Wins,
		Losses:                 standing.Losses,
		Streak:                 standing.Streak,
		FavouriteCivilizations: make([]api.CivilizationPlayed, 0, len(favourites)),
		RecentMatches:          apiMatches(recent.Games),
		AvatarUrl:              avatarURL(player.AvatarID),
	}
	if place > 0 {
		profile.Place = &place
	}
	for _, favourite := range favourites {
		profile.FavouriteCivilizations = append(profile.FavouriteCivilizations, api.CivilizationPlayed{Civilization: favourite.Civilization, Games: favourite.Games})
	}

	return profile, nil
}

func (h *handlers) GetPlayerMatches(ctx context.Context, request api.GetPlayerMatchesRequestObject) (api.GetPlayerMatchesResponseObject, error) {
	if _, _, err := h.currentUser(ctx); errors.Is(err, auth.ErrNoSession) {
		return api.GetPlayerMatches401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}

	player, err := h.person(ctx, request.Username)
	if errors.Is(err, users.ErrNotFound) {
		return api.GetPlayerMatches404JSONResponse{NotFoundJSONResponse: notFound(playerGone)}, nil
	}
	if err != nil {
		return nil, err
	}

	page, err := h.games.History(ctx, player.ID, pageRequest(request.Params.Page, request.Params.PageSize))
	if invalid, ok := invalidInput(err, nil); ok {
		return api.GetPlayerMatches400JSONResponse{InvalidInputJSONResponse: invalid}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.GetPlayerMatches200JSONResponse{Matches: apiMatches(page.Games), Total: page.Total, Page: page.Page, PageSize: page.PageSize}, nil
}

func (h *handlers) GetMatch(ctx context.Context, request api.GetMatchRequestObject) (api.GetMatchResponseObject, error) {
	if _, _, err := h.currentUser(ctx); errors.Is(err, auth.ErrNoSession) {
		return api.GetMatch401JSONResponse{UnauthenticatedJSONResponse: unauthenticated()}, nil
	} else if err != nil {
		return nil, err
	}

	finished, err := h.games.Recorded(ctx, request.GameId)
	if errors.Is(err, games.ErrNotRecorded) {
		return api.GetMatch404JSONResponse{NotFoundJSONResponse: notFound(matchGone)}, nil
	}
	if err != nil {
		return nil, err
	}

	return api.GetMatch200JSONResponse(apiMatch(finished)), nil
}

// account is a user as the API shows them to themselves, with their rank
// and any mute.
func (h *handlers) account(ctx context.Context, user users.User) (api.Account, error) {
	standing, err := h.ratings.Standing(ctx, user.ID)
	if err != nil {
		return api.Account{}, fmt.Errorf("reading %s's rank: %w", user.Username, err)
	}
	return api.Account{
		Id:            user.ID,
		Username:      user.Username,
		Email:         nonEmpty(user.Email),
		EmailVerified: user.IsVerified(),
		Role:          api.Role(user.Role),
		CreatedAt:     user.CreatedAt.UTC(),
		Rank:          apiRank(standing.Rank),
		Mute:          apiMute(user, h.now()),
		AvatarUrl:     avatarURL(user.AvatarID),
	}, nil
}

// ranks are players' ranks, by id, as names are shown with them.
type ranks map[string]ratings.Rank

// of is a player's rank as the API shows it, or nil when it is not known.
func (r ranks) of(userID string) *api.Rank {
	rank, ok := r[userID]
	if !ok {
		return nil
	}
	shown := apiRank(rank)
	return &shown
}

// ranksOf reads the ranks of players, by id.
func ranksOf(ctx context.Context, service *ratings.Service, userIDs []string) (ranks, error) {
	standings, err := service.Standings(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	found := make(ranks, len(standings))
	for id, standing := range standings {
		found[id] = standing.Rank
	}
	return found, nil
}

// pageRequest is the page a list is asked for, its parameters left out for
// the list's defaults.
func pageRequest(page, pageSize *int) paging.Request {
	var request paging.Request
	if page != nil {
		request.Page = *page
	}
	if pageSize != nil {
		request.PageSize = *pageSize
	}
	return request
}

func apiRank(rank ratings.Rank) api.Rank {
	shown := api.Rank{Rating: rank.Rating, Tier: api.Tier(rank.Tier), Label: rank.Label()}
	if rank.Division != "" {
		division := api.RankDivision(rank.Division)
		shown.Division = &division
	}
	return shown
}

func apiRankedPlayer(player ratings.Placed) api.RankedPlayer {
	return api.RankedPlayer{
		Id:       player.UserID,
		Username: player.Username,
		Place:    player.Place,
		Rank:     apiRank(player.Rank),
		Wins:     player.Wins,
		Losses:   player.Losses,
		Streak:   player.Streak,
	}
}

// person finds a player by name. Computer opponents have no page of their
// own, as it would show the games of everyone who practised against them, so
// it returns users.ErrNotFound for them too.
func (h *handlers) person(ctx context.Context, username string) (users.User, error) {
	player, err := h.users.ByUsername(ctx, username)
	if err == nil && player.Computer {
		return users.User{}, users.ErrNotFound
	}
	return player, err
}

func apiMatches(records []games.Record) []api.Match {
	matches := make([]api.Match, 0, len(records))
	for _, finished := range records {
		matches = append(matches, apiMatch(finished))
	}
	return matches
}

// apiMatch is a finished game as the API shows it. Whether it was listed is
// left out: history shows every game alike.
func apiMatch(finished games.Record) api.Match {
	match := api.Match{
		Id:              finished.ID,
		Name:            finished.Name,
		FormatName:      finished.FormatName,
		Ending:          api.MatchEnding(finished.Ending),
		Counted:         finished.Counted,
		Turns:           finished.Turns,
		DurationSeconds: int(finished.Duration().Seconds()),
		StartedAt:       finished.StartedAt.UTC(),
		EndedAt:         finished.EndedAt.UTC(),
		Host:            apiMatchPlayer(finished.Host),
		Guest:           apiMatchPlayer(finished.Guest),
	}
	if finished.WinnerID != "" {
		match.WinnerId = &finished.WinnerID
	}
	return match
}

func apiMatchPlayer(player games.RecordPlayer) api.MatchPlayer {
	shown := api.MatchPlayer{Id: player.ID, Username: player.Username, DeckName: player.DeckName, Civilizations: player.Civilizations, Computer: player.Computer}
	if shown.Civilizations == nil {
		shown.Civilizations = []string{}
	}
	if rating := player.Rating; rating != nil {
		shown.RatingChange, shown.Rating = &rating.Change, &rating.Rating
	}
	return shown
}
