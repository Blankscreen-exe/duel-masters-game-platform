package games

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var engineIDFormat = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestPlayableCardsListsEveryEngineCard(t *testing.T) {
	playable := PlayableCards()

	require.Greater(t, len(playable), 800, "the engine implements several hundred cards")

	ids := make(map[string]bool, len(playable))
	for i, card := range playable {
		assert.Regexp(t, engineIDFormat, card.EngineID)
		assert.NotEmpty(t, card.Name, card.EngineID)
		assert.NotEmpty(t, card.Set, card.EngineID)
		assert.False(t, ids[card.EngineID], "%s is listed once", card.EngineID)
		ids[card.EngineID] = true

		if i > 0 {
			assert.Less(t, playable[i-1].EngineID, card.EngineID, "sorted by engine id")
		}
	}
}

func TestPlayableCardsIsTheSameEveryTime(t *testing.T) {
	assert.Equal(t, PlayableCards(), PlayableCards())
}
