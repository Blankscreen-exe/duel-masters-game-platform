package cards

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cnd"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/tests/scenario"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const candyDropUID = "596f5b72-2502-4120-81f9-9ff9a17271d8"

func TestCandyDrop(t *testing.T) {
	scn := scenario.New()
	player := scn.Match.CurrentPlayer()
	opponent := scn.Match.PlayerRef(scn.Match.Opponent(player.Player))

	player.Player.SpawnCard(candyDropUID, match.HAND)
	for range 3 {
		player.Player.SpawnCard(candyDropUID, match.MANAZONE)
	}

	card, err := scn.FindCard(player.Player, match.HAND, candyDropUID)
	require.NoError(t, err)
	require.NoError(t, scn.ActionPlayCard(player, card.ID))

	require.NoError(t, scn.ActionEndTurn(player))
	require.NoError(t, scn.ActionEndTurn(opponent))

	candyDrop, err := scn.FindCard(player.Player, match.BATTLEZONE, candyDropUID)
	require.NoError(t, err)
	assert.True(t, candyDrop.HasCondition(cnd.CantBeBlocked))
	assert.Equal(t, 1000, scn.Match.GetPower(candyDrop, false))
}
