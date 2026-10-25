package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/httpapi/api"
)

func TestEveryoneReadsTheChatAndPlayersWriteInIt(t *testing.T) {
	server := newTestServer(t, false)
	session := server.signUp(t, "Shobu")

	empty := server.call(http.MethodGet, "/api/chat", nil, nil)
	require.Equal(t, http.StatusOK, empty.Code)
	assert.Equal(t, []api.ChatMessage{}, decode[[]api.ChatMessage](t, empty))

	response := server.call(http.MethodPost, "/api/chat", api.ChatPost{Text: "  Anyone up for a duel?  "}, session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	posted := decode[api.ChatMessage](t, response)
	assert.Equal(t, "Shobu", posted.Username)
	assert.Equal(t, "Anyone up for a duel?", posted.Text)

	visitor := server.call(http.MethodGet, "/api/chat", nil, nil)
	require.Equal(t, http.StatusOK, visitor.Code)
	assert.Equal(t, []api.ChatMessage{posted}, decode[[]api.ChatMessage](t, visitor), "visitors read it too")
}

func TestPostingNeedsALogin(t *testing.T) {
	server := newTestServer(t, false)

	response := server.call(http.MethodPost, "/api/chat", api.ChatPost{Text: "hello"}, nil)
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestANewMessageReachesEveryOpenTab(t *testing.T) {
	site := newTestServer(t, false)
	live := site.liveServer(t)
	session := site.signUp(t, "Shobu")

	visitor, _, err := openLive(t, live, nil, testSiteOrigin)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return site.hub.Connections() == 1 }, 5*time.Second, 5*time.Millisecond)

	response := site.call(http.MethodPost, "/api/chat", api.ChatPost{Text: "hello, lobby"}, session)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	update := readLive(t, visitor)
	assert.Equal(t, "chat_message", update["type"])
	message := update["message"].(map[string]any)
	assert.Equal(t, "Shobu", message["username"])
	assert.Equal(t, "hello, lobby", message["text"])
}

func TestABrokenMessageNamesTheField(t *testing.T) {
	server := newTestServer(t, false)
	session := server.signUp(t, "Shobu")

	for text, message := range map[string]string{
		"   ":                    "Text must not be empty.",
		strings.Repeat("a", 301): "Text must be at most 300 characters.",
	} {
		response := server.call(http.MethodPost, "/api/chat", api.ChatPost{Text: text}, session)
		require.Equal(t, http.StatusBadRequest, response.Code)
		answer := decode[api.Error](t, response)
		assert.Equal(t, codeInvalidInput, answer.Code)
		assert.Equal(t, "text", *answer.Field)
		assert.Equal(t, message, answer.Message)
	}
}

func TestPostingTooFastIsAnsweredWithTheWait(t *testing.T) {
	server := newTestServer(t, false)
	session := server.signUp(t, "Shobu")

	for i := range 5 {
		response := server.call(http.MethodPost, "/api/chat", api.ChatPost{Text: fmt.Sprintf("spam %d", i)}, session)
		require.Equal(t, http.StatusCreated, response.Code)
	}

	server.clock.Advance(9 * time.Second)
	response := server.call(http.MethodPost, "/api/chat", api.ChatPost{Text: "one more"}, session)
	require.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.Equal(t, "1", response.Header().Get("Retry-After"))
	assert.Equal(t, api.Error{Code: codeTooFast, Message: "You are posting too fast. Wait 1 second."}, decode[api.Error](t, response))
}
