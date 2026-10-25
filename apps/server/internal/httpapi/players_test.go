package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

// countEveryGame makes even a game of a few seconds count for ratings.
func (s playServer) countEveryGame(t *testing.T) {
	t.Helper()
	_, err := s.settings.Set(t.Context(), games.MinimumCountedDuration.Key, "0")
	require.NoError(t, err)
}

// recordedMatch reads a tab's live messages until the game's match_recorded
// arrives, and returns the match it carries.
func recordedMatch(t *testing.T, tab player, gameID string) api.Match {
	t.Helper()

	message := liveMessageWhere(t, tab.tab, "match_recorded", func(message map[string]any) bool {
		return message["match"].(map[string]any)["id"] == gameID
	})
	data, err := json.Marshal(message["match"])
	require.NoError(t, err)
	var match api.Match
	require.NoError(t, json.Unmarshal(data, &match))
	return match
}

func bronze(rating int, division api.RankDivision, label string) api.Rank {
	return api.Rank{Rating: rating, Tier: api.TierBronze, Division: &division, Label: label}
}

func TestPlayerPagesAreForPlayers(t *testing.T) {
	site := newPlayServer(t)
	site.player(t, "Shobu")

	for _, path := range []string{"/api/rankings", "/api/players/Shobu", "/api/players/Shobu/matches", "/api/matches/some-game"} {
		assert.Equal(t, http.StatusUnauthorized, site.call(http.MethodGet, path, nil, nil).Code, path)
	}
}

func TestAGameMovesRatingsThroughTheSite(t *testing.T) {
	site := newPlayServer(t)
	site.countEveryGame(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	mimi := site.player(t, "Mimi")

	me := decode[api.Account](t, site.call(http.MethodGet, "/api/auth/me", nil, shobu.session))
	assert.Equal(t, bronze(1000, api.RankDivisionII, "Bronze II"), me.Rank, "everyone starts at 1000")

	gameID := site.playUntilTheGuestSurrenders(t, shobu, kokujo)

	// Both players hear the game is saved, with what it did to their ratings.
	match := recordedMatch(t, shobu, gameID)
	assert.Equal(t, match, recordedMatch(t, kokujo, gameID))
	assert.Equal(t, gameID, match.Id)
	assert.Equal(t, api.MatchEndingSurrender, match.Ending)
	assert.True(t, match.Counted)
	require.NotNil(t, match.WinnerId)
	assert.Equal(t, match.Host.Id, *match.WinnerId)
	assert.Equal(t, "Shobu", match.Host.Username)
	assert.Equal(t, "Shobu's deck", match.Host.DeckName)
	assert.Equal(t, []string{"Fire"}, match.Host.Civilizations)
	require.NotNil(t, match.Host.RatingChange)
	assert.Equal(t, 16, *match.Host.RatingChange)
	assert.Equal(t, 1016, *match.Host.Rating)
	assert.Equal(t, -16, *match.Guest.RatingChange)
	assert.Equal(t, 984, *match.Guest.Rating)

	response := site.call(http.MethodGet, "/api/matches/"+gameID, nil, mimi.session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, match, decode[api.Match](t, response), "anyone may look it up")
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodGet, "/api/matches/no-such-game", nil, mimi.session).Code)

	me = decode[api.Account](t, site.call(http.MethodGet, "/api/auth/me", nil, shobu.session))
	assert.Equal(t, bronze(1016, api.RankDivisionII, "Bronze II"), me.Rank)

	// The lobby shows everyone the new ranks once the game is saved.
	liveMessageWhere(t, mimi.tab, "lobby", func(message map[string]any) bool {
		for _, online := range message["lobby"].(map[string]any)["online"].([]any) {
			player := online.(map[string]any)
			if player["username"] == "Shobu" {
				rank, shown := player["rank"].(map[string]any)
				return shown && rank["rating"] == float64(1016)
			}
		}
		return false
	})

	// The rankings list only those who have played, with the viewer's own
	// place.
	rankings := decode[api.Rankings](t, site.call(http.MethodGet, "/api/rankings", nil, kokujo.session))
	assert.Equal(t, 2, rankings.Total)
	require.Len(t, rankings.Players, 2)
	assert.Equal(t, api.RankedPlayer{Id: match.Host.Id, Username: "Shobu", Place: 1, Rank: bronze(1016, api.RankDivisionII, "Bronze II"), Wins: 1, Streak: 1}, rankings.Players[0])
	assert.Equal(t, "Kokujo", rankings.Players[1].Username)
	assert.Equal(t, -1, rankings.Players[1].Streak)
	require.NotNil(t, rankings.You)
	assert.Equal(t, rankings.Players[1], *rankings.You)
	assert.Nil(t, decode[api.Rankings](t, site.call(http.MethodGet, "/api/rankings", nil, mimi.session)).You, "Mimi is not ranked yet")

	found := decode[api.Rankings](t, site.call(http.MethodGet, "/api/rankings?search=KOKU&pageSize=1", nil, mimi.session))
	require.Len(t, found.Players, 1)
	assert.Equal(t, 2, found.Players[0].Place)
	assert.Equal(t, http.StatusBadRequest, site.call(http.MethodGet, "/api/rankings?pageSize=101", nil, mimi.session).Code)

	// Profiles and match history, by name in any case.
	profile := decode[api.PlayerProfile](t, site.call(http.MethodGet, "/api/players/kokujo", nil, mimi.session))
	assert.Equal(t, "Kokujo", profile.Username)
	assert.Equal(t, bronze(984, api.RankDivisionII, "Bronze II"), profile.Rank)
	require.NotNil(t, profile.Place)
	assert.Equal(t, 2, *profile.Place)
	assert.Equal(t, 0, profile.Wins)
	assert.Equal(t, 1, profile.Losses)
	assert.Equal(t, []api.CivilizationPlayed{{Civilization: "Fire", Games: 1}}, profile.FavouriteCivilizations)
	assert.Equal(t, []api.Match{match}, profile.RecentMatches)

	unranked := decode[api.PlayerProfile](t, site.call(http.MethodGet, "/api/players/Mimi", nil, mimi.session))
	assert.Nil(t, unranked.Place)
	assert.Empty(t, unranked.RecentMatches)
	assert.Equal(t, []api.CivilizationPlayed{}, unranked.FavouriteCivilizations)
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodGet, "/api/players/nobody", nil, mimi.session).Code)

	history := decode[api.MatchPage](t, site.call(http.MethodGet, "/api/players/SHOBU/matches", nil, mimi.session))
	assert.Equal(t, api.MatchPage{Matches: []api.Match{match}, Total: 1, Page: 1, PageSize: games.HistoryPageSize}, history)
	assert.Equal(t, http.StatusBadRequest, site.call(http.MethodGet, "/api/players/Shobu/matches?page=-1", nil, mimi.session).Code)
	assert.Equal(t, http.StatusNotFound, site.call(http.MethodGet, "/api/players/nobody/matches", nil, mimi.session).Code)
}

func TestTheLobbyShowsPlayersRanks(t *testing.T) {
	site := newPlayServer(t)
	shobu := site.player(t, "Shobu")
	kokujo := site.player(t, "Kokujo")
	open := site.openGame(t, shobu, api.OpenGameInput{Listed: true, Spectators: true})
	require.NotNil(t, open.Host.Rank)
	assert.Equal(t, 1000, open.Host.Rank.Rating)

	shown := decode[api.Lobby](t, site.call(http.MethodGet, "/api/lobby", nil, kokujo.session))
	require.Len(t, shown.Online, 2)
	for _, player := range shown.Online {
		require.NotNil(t, player.Rank, player.Username)
		assert.Equal(t, "Bronze II", player.Rank.Label)
	}

	live := liveMessageWhere(t, kokujo.tab, "lobby", func(message map[string]any) bool {
		return len(message["lobby"].(map[string]any)["openGames"].([]any)) == 1
	})
	host := live["lobby"].(map[string]any)["openGames"].([]any)[0].(map[string]any)["host"].(map[string]any)
	assert.Equal(t, "Bronze II", host["rank"].(map[string]any)["label"], "live updates show ranks too")

	response := site.call(http.MethodPost, "/api/lobby/games/"+open.Id+"/join", api.JoinGameInput{DeckId: kokujo.deckID}, kokujo.session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	game := decode[api.LiveGame](t, site.call(http.MethodGet, "/api/games/"+decode[api.GameStarted](t, response).GameId, nil, kokujo.session))
	require.NotNil(t, game.Host.Rank)
	require.NotNil(t, game.Guest.Rank)
	assert.Equal(t, 1000, game.Guest.Rank.Rating, "the game shows both players' ranks")
}
