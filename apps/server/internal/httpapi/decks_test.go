package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

// newDeckTestServer has ten playable cards, "Card 01" to "Card 10" in DM-01
// costing 1 to 10, enough for a legal deck; a playable promo-only card, "Promo
// Hero"; and "Future Card", which the engine cannot play.
func newDeckTestServer(t *testing.T) testServer {
	t.Helper()

	var printings []map[string]any
	var pictures []string
	var playable []games.PlayableCard
	for i := 1; i <= 10; i++ {
		id, name := fmt.Sprintf("%04d", i), fmt.Sprintf("Card %02d", i)
		printing := cardstest.Printing(id, name, "DM-01", id+"/110")
		printing["cost"] = i
		printings = append(printings, printing)
		pictures = append(pictures, id)
		playable = append(playable, games.PlayableCard{EngineID: "engine-" + id, Name: name})
	}

	promo := cardstest.Printing("9001", "Promo Hero", "PROMO", "P1/Y1")
	promo["civilizations"] = []string{"Nature"}
	printings = append(printings, promo, cardstest.Printing("1500", "Future Card", "DM-20", "1/55"))
	playable = append(playable, games.PlayableCard{EngineID: "engine-promo", Name: "Promo Hero"})

	return newTestServerWith(t, false, cardstest.Write(t, printings, pictures, nil), playable)
}

// legalDeckInput is 4 copies of each of the ten DM-01 cards.
func legalDeckInput(name string) api.DeckInput {
	input := api.DeckInput{Name: name, Cards: []api.DeckEntry{}}
	for i := 1; i <= 10; i++ {
		input.Cards = append(input.Cards, api.DeckEntry{CardId: fmt.Sprintf("%04d", i), Count: 4})
	}
	return input
}

func (s testServer) formats(t *testing.T) (open, classic api.Format) {
	t.Helper()

	response := s.call(http.MethodGet, "/api/formats", nil, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	list := decode[[]api.Format](t, response)
	require.Len(t, list, 2)

	return list[0], list[1]
}

func (s testServer) createDeck(t *testing.T, session *http.Cookie, input api.DeckInput) api.Deck {
	t.Helper()

	response := s.call(http.MethodPost, "/api/decks", input, session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	return decode[api.Deck](t, response)
}

func TestFormatsAreListedForEveryone(t *testing.T) {
	server := newDeckTestServer(t)

	open, classic := server.formats(t)
	assert.Equal(t, "Open", open.Name)
	assert.True(t, open.AllSets)
	assert.Equal(t, []string{}, open.Sets)
	assert.Equal(t, []api.FormatCardLimit{}, open.Limits)
	assert.Equal(t, "DM-01 to DM-12", classic.Name)
	assert.False(t, classic.AllSets)
	assert.Len(t, classic.Sets, 12)
}

func TestDecksNeedALogin(t *testing.T) {
	server := newDeckTestServer(t)

	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/decks", nil},
		{http.MethodPost, "/api/decks", legalDeckInput("Deck")},
		{http.MethodPost, "/api/decks/text", api.DeckListText{Text: "4 Card 01"}},
		{http.MethodGet, "/api/decks/some-deck", nil},
		{http.MethodPut, "/api/decks/some-deck", legalDeckInput("Deck")},
		{http.MethodDelete, "/api/decks/some-deck", nil},
		{http.MethodGet, "/api/decks/some-deck/text", nil},
	} {
		response := server.call(request.method, request.path, request.body, nil)
		assert.Equal(t, http.StatusUnauthorized, response.Code, "%s %s", request.method, request.path)
		assert.Equal(t, codeUnauthenticated, decode[api.Error](t, response).Code)
	}
}

func TestCreatingADeckShowsItsLegalityInEveryFormat(t *testing.T) {
	server := newDeckTestServer(t)
	session := server.signUp(t, "Shobu")
	open, classic := server.formats(t)

	deck := server.createDeck(t, session, api.DeckInput{
		Name:  "Draft",
		Cards: []api.DeckEntry{{CardId: "9001", Count: 2}, {CardId: "0003", Count: 4}, {CardId: "0007", Count: 1}},
	})

	assert.Equal(t, "Draft", deck.Name)
	assert.Equal(t, 7, deck.CardCount)
	assert.Equal(t, []string{"Fire", "Nature"}, deck.Civilizations)
	assert.Nil(t, deck.CoverCardId)
	require.NotNil(t, deck.Cover)
	assert.Equal(t, "0007", deck.Cover.CardId, "the most expensive card")
	assert.Equal(t, "/api/cards/images/0007", *deck.Cover.ImageUrl)

	ids := []string{}
	for _, card := range deck.Cards {
		ids = append(ids, card.CardId)
		require.NotNil(t, card.Card)
		assert.Equal(t, card.CardId, card.Card.Id)
	}
	assert.Equal(t, []string{"0003", "9001", "0007"}, ids, "by cost: 3, 3 (Promo Hero is cost 3 too, after Card 03 by name), 7")

	require.Len(t, deck.Legality, 2)
	assert.Equal(t, api.DeckLegality{FormatId: open.Id, Legal: false, Problems: []api.DeckProblem{
		{Code: api.DeckProblemCodeDeckSize, Message: "The deck has 7 cards; it needs exactly 40."},
	}}, deck.Legality[0])
	promo := "9001"
	assert.Equal(t, api.DeckLegality{FormatId: classic.Id, Legal: false, Problems: []api.DeckProblem{
		{Code: api.DeckProblemCodeDeckSize, Message: "The deck has 7 cards; it needs exactly 40."},
		{Code: api.DeckProblemCodeNotInFormat, Message: "Promo Hero is not from a set this format allows.", CardId: &promo},
	}}, deck.Legality[1])
}

func TestALegalDeckIsListedWithItsFormats(t *testing.T) {
	server := newDeckTestServer(t)
	session := server.signUp(t, "Shobu")
	open, classic := server.formats(t)

	input := legalDeckInput("Fire Rush")
	cover := "0002"
	input.CoverCardId = &cover
	created := server.createDeck(t, session, input)
	assert.Equal(t, &cover, created.CoverCardId)
	for _, legality := range created.Legality {
		assert.True(t, legality.Legal)
		assert.Empty(t, legality.Problems)
	}

	empty := server.createDeck(t, session, api.DeckInput{Name: "Empty", Cards: []api.DeckEntry{}})

	response := server.call(http.MethodGet, "/api/decks", nil, session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	list := decode[[]api.DeckSummary](t, response)
	require.Len(t, list, 2)

	assert.Equal(t, empty.Id, list[0].Id, "the latest changed first")
	assert.Nil(t, list[0].Cover, "an empty deck has no cover")
	assert.Equal(t, []string{}, list[0].LegalFormatIds)
	assert.Equal(t, []string{}, list[0].Civilizations)

	assert.Equal(t, api.DeckSummary{
		Id:             created.Id,
		Name:           "Fire Rush",
		CardCount:      40,
		Civilizations:  []string{"Fire"},
		Cover:          &api.DeckCover{CardId: "0002", ImageUrl: created.Cover.ImageUrl},
		LegalFormatIds: []string{open.Id, classic.Id},
		UpdatedAt:      created.UpdatedAt,
	}, list[1])
}

func TestChangingAndDeletingADeck(t *testing.T) {
	server := newDeckTestServer(t)
	session := server.signUp(t, "Shobu")
	deck := server.createDeck(t, session, legalDeckInput("Before"))

	response := server.call(http.MethodPut, "/api/decks/"+deck.Id, api.DeckInput{Name: "After", Cards: []api.DeckEntry{{CardId: "0001", Count: 1}}}, session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	updated := decode[api.Deck](t, response)
	assert.Equal(t, "After", updated.Name)
	assert.Equal(t, 1, updated.CardCount)

	response = server.call(http.MethodGet, "/api/decks/"+deck.Id, nil, session)
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, updated, decode[api.Deck](t, response))

	response = server.call(http.MethodDelete, "/api/decks/"+deck.Id, nil, session)
	assert.Equal(t, http.StatusNoContent, response.Code, response.Body.String())

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		response = server.call(method, "/api/decks/"+deck.Id, nil, session)
		assert.Equal(t, http.StatusNotFound, response.Code, method)
		assert.Equal(t, codeNotFound, decode[api.Error](t, response).Code)
	}
}

func TestAnotherPlayersDeckIsNotFound(t *testing.T) {
	server := newDeckTestServer(t)
	owner := server.signUp(t, "Shobu")
	other := server.signUp(t, "Kokujo")
	deck := server.createDeck(t, owner, legalDeckInput("Mine"))

	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/decks/" + deck.Id, nil},
		{http.MethodPut, "/api/decks/" + deck.Id, legalDeckInput("Taken")},
		{http.MethodDelete, "/api/decks/" + deck.Id, nil},
		{http.MethodGet, "/api/decks/" + deck.Id + "/text", nil},
	} {
		response := server.call(request.method, request.path, request.body, other)
		assert.Equal(t, http.StatusNotFound, response.Code, "%s %s", request.method, request.path)
	}

	response := server.call(http.MethodGet, "/api/decks", nil, other)
	require.Equal(t, http.StatusOK, response.Code)
	assert.Empty(t, decode[[]api.DeckSummary](t, response))
}

func TestSavingABrokenDeckNamesTheField(t *testing.T) {
	server := newDeckTestServer(t)
	session := server.signUp(t, "Shobu")
	deck := server.createDeck(t, session, api.DeckInput{Name: "Deck", Cards: []api.DeckEntry{}})
	cover := "0005"

	for _, test := range []struct {
		input   api.DeckInput
		field   string
		message string
	}{
		{api.DeckInput{Name: " ", Cards: []api.DeckEntry{}}, "name", "Name must not be empty."},
		{api.DeckInput{Name: "D", Cards: []api.DeckEntry{{CardId: "1500", Count: 1}}}, "cards", "Cards include Future Card, which cannot be played yet."},
		{api.DeckInput{Name: "D", Cards: []api.DeckEntry{{CardId: "0001", Count: 5}}}, "cards", "Cards must have 1 to 4 copies of Card 01, not 5."},
		{api.DeckInput{Name: "D", CoverCardId: &cover, Cards: []api.DeckEntry{{CardId: "0001", Count: 1}}}, "coverCardId", "CoverCardId must be one of the deck's cards."},
	} {
		for _, call := range []struct{ method, path string }{{http.MethodPost, "/api/decks"}, {http.MethodPut, "/api/decks/" + deck.Id}} {
			response := server.call(call.method, call.path, test.input, session)
			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
			answer := decode[api.Error](t, response)
			assert.Equal(t, codeInvalidInput, answer.Code)
			require.NotNil(t, answer.Field)
			assert.Equal(t, test.field, *answer.Field)
			assert.Equal(t, test.message, answer.Message)
		}
	}
}

func TestTooManyDecksIsAConflict(t *testing.T) {
	server := newDeckTestServer(t)
	session := server.signUp(t, "Shobu")
	_, err := server.settings.Set(context.Background(), decks.MaxPerPlayer.Key, "1")
	require.NoError(t, err)
	server.createDeck(t, session, api.DeckInput{Name: "Only", Cards: []api.DeckEntry{}})

	response := server.call(http.MethodPost, "/api/decks", api.DeckInput{Name: "Second", Cards: []api.DeckEntry{}}, session)
	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Equal(t, api.Error{Code: codeTooManyDecks, Message: "You can have at most 1 decks. Delete one to make room."}, decode[api.Error](t, response))
}

func TestADeckListWritesAndReadsBack(t *testing.T) {
	server := newDeckTestServer(t)
	session := server.signUp(t, "Shobu")
	deck := server.createDeck(t, session, api.DeckInput{Name: "List", Cards: []api.DeckEntry{{CardId: "0004", Count: 3}, {CardId: "0002", Count: 4}}})

	response := server.call(http.MethodGet, "/api/decks/"+deck.Id+"/text", nil, session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, "text/plain", response.Header().Get("Content-Type"))
	text := response.Body.String()
	assert.Equal(t, "4 Card 02\n3 Card 04\n", text)

	response = server.call(http.MethodPost, "/api/decks/text", api.DeckListText{Text: text + "2 Future Card\nnonsense line\n"}, session)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	reading := decode[api.DeckListReading](t, response)

	require.Len(t, reading.Cards, 2)
	assert.Equal(t, "0002", reading.Cards[0].CardId)
	assert.Equal(t, 4, reading.Cards[0].Count)
	require.NotNil(t, reading.Cards[0].Card)
	assert.Equal(t, "Card 02", reading.Cards[0].Card.Name)
	assert.Equal(t, "0004", reading.Cards[1].CardId)
	assert.Equal(t, []api.DeckListProblem{
		{Line: 3, Text: "2 Future Card", Message: "Future Card cannot be played yet."},
		{Line: 4, Text: "nonsense line", Message: `No card is called "nonsense line".`},
	}, reading.Problems)

	response = server.call(http.MethodPost, "/api/decks/text", api.DeckListText{Text: strings.Repeat("x", decks.TextMaxLength+1)}, session)
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "text", *decode[api.Error](t, response).Field)
}

func TestEveryDeckProblemIsInTheContract(t *testing.T) {
	for _, kind := range []decks.ProblemKind{
		decks.ProblemSize,
		decks.ProblemNotPlayable,
		decks.ProblemNotInFormat,
		decks.ProblemBanned,
		decks.ProblemOverLimit,
		decks.ProblemTooManyCopies,
	} {
		assert.True(t, api.DeckProblemCode(kind).Valid(), kind)
	}
}
