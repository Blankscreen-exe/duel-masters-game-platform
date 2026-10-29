package tests

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chatRefusalText = "You are posting too fast. Wait 3 seconds."

// waitForChat waits for a chat message to reach the player, since position
// since, and returns the chat they were sent.
func waitForChat(t *testing.T, scn *scenario.TestScenario, player *match.PlayerReference, since int, text string) []server.ChatMessage {
	t.Helper()

	for {
		require.NoError(t, scn.WaitForMessage(player, since, "chat"), "waiting for %q", text)
		log, err := scn.ChatLog(player, since)
		require.NoError(t, err)
		for _, message := range log {
			if message.Message == text {
				return log
			}
		}
		count, err := scn.MessageCount(player)
		require.NoError(t, err)
		since = count
	}
}

func TestARefusedChatMessageIsShownToNobodyButItsSender(t *testing.T) {
	// Player1 is refused; Player2 may chat.
	scn := scenario.New(scenario.WithChatRefusal(func(userID string) string {
		if userID == "1" {
			return chatRefusalText
		}
		return ""
	}))
	refused, allowed := scn.Match.Player1, scn.Match.Player2

	refusedSince, err := scn.MessageCount(refused)
	require.NoError(t, err)
	allowedSince, err := scn.MessageCount(allowed)
	require.NoError(t, err)

	require.NoError(t, scn.SendChat(refused, "spam"))
	log := waitForChat(t, scn, refused, refusedSince, chatRefusalText)
	for _, message := range log {
		assert.NotEqual(t, "spam", message.Message, "the sender does not see their refused message either")
		if message.Message == chatRefusalText {
			assert.Equal(t, "Server", message.Sender)
		}
	}

	require.NoError(t, scn.SendChat(allowed, "hello"))
	log = waitForChat(t, scn, allowed, allowedSince, "hello")
	for _, message := range log {
		assert.NotEqual(t, "spam", message.Message, "the opponent never sees the refused message")
		assert.NotEqual(t, chatRefusalText, message.Message, "only the sender is told why")
	}
	waitForChat(t, scn, refused, refusedSince, "hello")
}

func TestWithoutAChatRefusalEveryMessageIsShown(t *testing.T) {
	scn := scenario.New()
	sender, opponent := scn.Match.Player1, scn.Match.Player2

	since, err := scn.MessageCount(opponent)
	require.NoError(t, err)
	require.NoError(t, scn.SendChat(sender, "hello"))
	waitForChat(t, scn, opponent, since, "hello")
}
