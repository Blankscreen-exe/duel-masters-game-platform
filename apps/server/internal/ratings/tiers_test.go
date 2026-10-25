package ratings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/ratings"
)

func TestRankOf(t *testing.T) {
	for _, test := range []struct {
		rating int
		label  string
	}{
		{-50, "Bronze III"},
		{900, "Bronze III"},
		{966, "Bronze III"},
		{967, "Bronze II"},
		{ratings.StartingRating, "Bronze II"},
		{1034, "Bronze I"},
		{1099, "Bronze I"},
		{1100, "Silver III"},
		{1166, "Silver III"},
		{1167, "Silver II"},
		{1233, "Silver II"},
		{1234, "Silver I"},
		{1300, "Gold III"},
		{1499, "Gold I"},
		{1500, "Platinum III"},
		{1699, "Platinum I"},
		{1700, "Diamond"},
		{2600, "Diamond"},
	} {
		rank := ratings.RankOf(test.rating)
		assert.Equal(t, test.label, rank.Label(), "rating %d", test.rating)
		assert.Equal(t, test.rating, rank.Rating)
	}
}

func TestRankOfNamesTierAndDivision(t *testing.T) {
	assert.Equal(t, ratings.Rank{Rating: 1350, Tier: ratings.Gold, Division: ratings.DivisionIII}, ratings.RankOf(1350))
	assert.Equal(t, ratings.Rank{Rating: 1800, Tier: ratings.Diamond}, ratings.RankOf(1800), "Diamond has no divisions")
}
