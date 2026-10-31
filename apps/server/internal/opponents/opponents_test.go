package opponents_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// newService copies the real card database into db, as the server does at
// startup, and returns the opponents service. The opponents are not seeded
// yet.
func newService(t *testing.T, db *gorm.DB) *opponents.Service {
	t.Helper()

	dir := cardstest.RealDatabase(t)
	source, err := cards.ReadSource(dir)
	require.NoError(t, err)
	_, err = cards.Sync(context.Background(), db, source, games.PlayableCards(), storetest.Logger())
	require.NoError(t, err)

	return opponents.NewService(db, cards.NewCatalogue(db, dir), formats.NewService(db))
}

func seed(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, store.RunSeeds(db, opponents.Seeds(storetest.Logger())...))
}

func TestTheStartingOpponentsArePlayable(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)
		seed(t, db)

		list, err := service.List(ctx)
		require.NoError(t, err)
		names := make([]string, 0, len(list))
		for _, opponent := range list {
			names = append(names, opponent.Name)
		}
		assert.Equal(t, []string{"Grimwhisper", "Blazerush", "Aegis", "Stormbinder", "Tidecaller", "Titangrove"}, names)

		accounts := users.NewService(db, storetest.Logger())
		for _, opponent := range list {
			assert.Equal(t, decks.Size, decks.Count(opponent.Cards), "%s has a whole deck", opponent.Name)
			_, ok := computer.StyleNamed(opponent.Style)
			assert.True(t, ok, "%s plays a style there is", opponent.Name)
			assert.Equal(t, computer.Normal, opponent.Strength)
			assert.True(t, opponent.Enabled)
			assert.Equal(t, opponents.BaseCardTypes, opponent.Rules.CardTypes)

			problems, err := service.Check(ctx, opponent)
			require.NoError(t, err)
			assert.Empty(t, problems, "%s's deck keeps to its format and its rules", opponent.Name)

			account, err := accounts.ByID(ctx, opponent.UserID)
			require.NoError(t, err)
			assert.True(t, account.Computer, "%s plays under a computer's account", opponent.Name)
			assert.Equal(t, opponent.Name, account.Username)

			got, err := service.Get(ctx, opponent.ID)
			require.NoError(t, err)
			assert.Equal(t, opponent, got)
		}

		exist, err := accounts.AnyExist(ctx)
		require.NoError(t, err)
		assert.False(t, exist, "so the first person to sign up still becomes the admin")
	})
}

func TestAnOpponentWhoseNameIsTakenIsLeftOut(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)
		accounts := users.NewService(db, storetest.Logger())
		person, err := accounts.Create(ctx, "aegis", "hash")
		require.NoError(t, err)

		seed(t, db)

		list, err := service.List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 5)
		for _, opponent := range list {
			assert.NotEqual(t, person.ID, opponent.UserID)
		}
		stored, err := accounts.ByID(ctx, person.ID)
		require.NoError(t, err)
		assert.False(t, stored.Computer, "the person keeps their account as it was")
	})
}

func TestARetiredOpponentIsNotOffered(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)
		seed(t, db)
		list, err := service.List(ctx)
		require.NoError(t, err)
		retired := list[0]

		require.NoError(t, db.Table("computer_opponents").Where("id = ?", retired.ID).Update("enabled", false).Error)

		_, err = service.Get(ctx, retired.ID)
		assert.ErrorIs(t, err, opponents.ErrNotFound)
		list, err = service.List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 5)
	})
}

func TestCheckHoldsADeckToItsOpponentsRules(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		service := newService(t, db)
		seed(t, db)
		list, err := service.List(ctx)
		require.NoError(t, err)

		var stormbinder opponents.Opponent
		for _, opponent := range list {
			if opponent.Name == "Stormbinder" {
				stormbinder = opponent
			}
		}
		require.NotEmpty(t, stormbinder.ID)

		kinds := func(opponent opponents.Opponent) []decks.ProblemKind {
			t.Helper()
			problems, err := service.Check(ctx, opponent)
			require.NoError(t, err)
			found := make([]decks.ProblemKind, 0, len(problems))
			for _, problem := range problems {
				found = append(found, problem.Kind)
			}
			return found
		}

		// Its Light and Water deck breaks a rule of one civilization.
		oneCivilization := stormbinder
		oneCivilization.Rules.MaxCivilizations = 1
		assert.Equal(t, []decks.ProblemKind{decks.ProblemTooManyCivilizations}, kinds(oneCivilization))

		// Its format can hold it to a number of civilizations too, and when both
		// set a most, the tighter alone is reported.
		setFormat := func(fewest, most int) {
			t.Helper()
			require.NoError(t, db.Table("formats").Where("id = ?", stormbinder.FormatID).
				Updates(map[string]any{"min_civilizations": fewest, "max_civilizations": most}).Error)
		}
		check := func(opponent opponents.Opponent) []decks.Problem {
			t.Helper()
			problems, err := service.Check(ctx, opponent)
			require.NoError(t, err)
			return problems
		}
		tooMany := func(limit int) []decks.Problem {
			return []decks.Problem{{Kind: decks.ProblemTooManyCivilizations, Count: 2, Limit: limit}}
		}

		setFormat(0, 1)
		assert.Equal(t, tooMany(1), check(stormbinder), "the format's alone")
		assert.Equal(t, tooMany(1), check(oneCivilization), "both the same")
		setFormat(0, 2)
		assert.Equal(t, tooMany(1), check(oneCivilization), "its own is tighter")
		noneOfItsOwn := stormbinder
		noneOfItsOwn.Rules.MaxCivilizations = 2
		setFormat(0, 1)
		assert.Equal(t, tooMany(1), check(noneOfItsOwn), "the format's is tighter")
		setFormat(3, 0)
		assert.Equal(t, []decks.ProblemKind{decks.ProblemTooFewCivilizations}, kinds(stormbinder))
		setFormat(0, 0)

		// And a card short is no deck.
		short := stormbinder
		short.Cards = stormbinder.Cards[1:]
		assert.Equal(t, []decks.ProblemKind{decks.ProblemSize}, kinds(short))
	})
}
