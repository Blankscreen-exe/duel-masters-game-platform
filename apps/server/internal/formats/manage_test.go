package formats_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/formats"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby/lobbytest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// managing is a world with the format manager and an admin to use it.
type managing struct {
	*lobbytest.World
	manager *formats.Manager
	admin   users.User
	// sets are the card database's set codes.
	sets []string
}

func newManaging(t *testing.T, db *gorm.DB) managing {
	t.Helper()

	w := lobbytest.NewWorld(t, db)
	admin, err := users.NewService(db, storetest.Logger()).ByID(context.Background(), w.Shobu.ID)
	require.NoError(t, err)
	require.True(t, admin.IsAdmin(), "the first account is the admin")

	values, err := w.Catalogue.FilterValues(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, values.Sets)
	sets := make([]string, 0, len(values.Sets))
	for _, set := range values.Sets {
		sets = append(sets, set.Code)
	}

	return managing{World: w, manager: formats.NewManager(db, w.Formats, w.Catalogue), admin: admin, sets: sets}
}

// input is a valid format allowing the first set, banning one card and limiting
// another.
func (m managing) input(name string) formats.Input {
	return formats.Input{
		Name:        name,
		Description: "  Old cards only.  ",
		Sets:        []string{m.sets[0]},
		Limits:      map[string]int{m.CardIDs[0]: 0, m.CardIDs[1]: 2},
	}
}

func (m managing) logged(t *testing.T) []adminlog.Entry {
	t.Helper()

	page, err := adminlog.NewService(m.DB).List(context.Background(), adminlog.Query{Request: paging.Request{}})
	require.NoError(t, err)
	return page.Entries
}

func TestAnAdminAddsAFormat(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)

		added, err := m.manager.Create(ctx, m.admin, m.input("  Legends  "))
		require.NoError(t, err)
		assert.Equal(t, "Legends", added.Name, "trimmed")
		assert.Equal(t, "Old cards only.", added.Description, "trimmed")
		assert.False(t, added.AllSets)
		assert.Equal(t, []string{m.sets[0]}, added.Sets)
		assert.Equal(t, map[string]int{m.CardIDs[0]: 0, m.CardIDs[1]: 2}, added.Limits)
		assert.False(t, added.Retired)

		list, err := m.Formats.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 3)
		assert.Equal(t, added.ID, list[2].ID, "last in order")

		log := m.logged(t)
		require.NotEmpty(t, log)
		assert.Equal(t, adminlog.FormatAdded, log[0].Action)
		assert.Equal(t, "Legends", log[0].Subject)
	})
}

func TestAFormatAllowingEverySetListsNone(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		m := newManaging(t, db)
		input := m.input("Anything goes")
		input.AllSets = true

		added, err := m.manager.Create(context.Background(), m.admin, input)
		require.NoError(t, err)
		assert.True(t, added.AllSets)
		assert.Empty(t, added.Sets)
		assert.True(t, added.AllowsSet("DM-99"), "even one added later")
	})
}

func TestAFormatsInputIsChecked(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)

		for _, test := range []struct {
			name   string
			change func(*formats.Input)
			field  string
		}{
			{"no name", func(i *formats.Input) { i.Name = "   " }, "name"},
			{"a long name", func(i *formats.Input) { i.Name = strings.Repeat("x", formats.NameMaxLength+1) }, "name"},
			{"a name of two lines", func(i *formats.Input) { i.Name = "a\nb" }, "name"},
			{"no description", func(i *formats.Input) { i.Description = " " }, "description"},
			{"a long description", func(i *formats.Input) { i.Description = strings.Repeat("x", formats.DescriptionMaxLength+1) }, "description"},
			{"a description of two lines", func(i *formats.Input) { i.Description = "a\nb" }, "description"},
			{"no sets", func(i *formats.Input) { i.Sets = nil }, "sets"},
			{"an unknown set", func(i *formats.Input) { i.Sets = []string{"DM-99"} }, "sets"},
			{"a negative limit", func(i *formats.Input) { i.Limits = map[string]int{m.CardIDs[0]: -1} }, "limits"},
			{"too high a limit", func(i *formats.Input) { i.Limits = map[string]int{m.CardIDs[0]: formats.MaxLimit + 1} }, "limits"},
			{"an unknown card", func(i *formats.Input) { i.Limits = map[string]int{"no-such-card": 0} }, "limits"},
		} {
			input := m.input("Legends")
			test.change(&input)

			_, err := m.manager.Create(ctx, m.admin, input)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%s: got %v", test.name, err)
			assert.Equal(t, test.field, invalid.Field, test.name)
		}

		list, err := m.Formats.All(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 2, "nothing was added")
	})
}

func TestAFormatsNameIsItsOwnInAnyCase(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)

		_, err := m.manager.Create(ctx, m.admin, m.input("oPEN"))
		assert.ErrorIs(t, err, formats.ErrNameTaken)

		legends, err := m.manager.Create(ctx, m.admin, m.input("Legends"))
		require.NoError(t, err)
		_, err = m.manager.Update(ctx, m.admin, legends.ID, m.input("open"))
		assert.ErrorIs(t, err, formats.ErrNameTaken)

		renamed, err := m.manager.Update(ctx, m.admin, legends.ID, m.input("LEGENDS"))
		require.NoError(t, err, "its own name is free to it, in another case")
		assert.Equal(t, "LEGENDS", renamed.Name)

		retired, err := m.manager.SetRetired(ctx, m.admin, renamed.ID, true)
		require.NoError(t, err)
		_, err = m.manager.Create(ctx, m.admin, m.input(retired.Name))
		assert.ErrorIs(t, err, formats.ErrNameTaken, "a retired format keeps its name")
	})
}

func TestAnAdminChangesAFormat(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)
		added, err := m.manager.Create(ctx, m.admin, m.input("Legends"))
		require.NoError(t, err)

		changed, err := m.manager.Update(ctx, m.admin, added.ID, formats.Input{
			Name:        "Masters",
			Description: "Every card.",
			AllSets:     true,
			Sets:        []string{m.sets[0]},
			Limits:      map[string]int{m.CardIDs[2]: 1},
		})
		require.NoError(t, err)
		assert.Equal(t, "Masters", changed.Name)
		assert.Equal(t, "Every card.", changed.Description)
		assert.True(t, changed.AllSets)
		assert.Empty(t, changed.Sets, "the old sets are gone")
		assert.Equal(t, map[string]int{m.CardIDs[2]: 1}, changed.Limits, "the old limits are gone")

		log := m.logged(t)
		assert.Equal(t, adminlog.FormatChanged, log[0].Action)
		assert.Equal(t, "Masters", log[0].Subject)
		assert.Equal(t, "Was called Legends", log[0].Details)

		_, err = m.manager.Update(ctx, m.admin, "nothing", m.input("Other"))
		assert.ErrorIs(t, err, formats.ErrNotFound)
	})
}

func TestARetiredFormatIsNoLongerOffered(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)

		retired, err := m.manager.SetRetired(ctx, m.admin, m.Classic.ID, true)
		require.NoError(t, err)
		assert.True(t, retired.Retired)
		assert.Equal(t, adminlog.FormatRetired, m.logged(t)[0].Action)

		list, err := m.Formats.List(ctx)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, m.Open.ID, list[0].ID, "players cannot choose it")

		all, err := m.Formats.All(ctx)
		require.NoError(t, err)
		assert.Len(t, all, 2, "admins still see it")

		found, err := m.Formats.ByID(ctx, m.Classic.ID)
		require.NoError(t, err, "what already uses it still finds it")
		assert.True(t, found.Retired)
		assert.Equal(t, m.Classic.Sets, found.Sets, "with its rules")

		back, err := m.manager.SetRetired(ctx, m.admin, m.Classic.ID, false)
		require.NoError(t, err)
		assert.False(t, back.Retired)
		assert.Equal(t, adminlog.FormatRestored, m.logged(t)[0].Action)
		list, err = m.Formats.List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 2)
	})
}

func TestAskingForWhatAlreadyHoldsRecordsNothing(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)

		_, err := m.manager.SetRetired(ctx, m.admin, m.Classic.ID, false)
		require.NoError(t, err)
		assert.Empty(t, m.logged(t))

		_, err = m.manager.SetRetired(ctx, m.admin, m.Classic.ID, true)
		require.NoError(t, err)
		_, err = m.manager.SetRetired(ctx, m.admin, m.Classic.ID, true)
		require.NoError(t, err)
		assert.Len(t, m.logged(t), 1)

		_, err = m.manager.SetRetired(ctx, m.admin, "nothing", true)
		assert.ErrorIs(t, err, formats.ErrNotFound)
	})
}

func TestTheLastFormatCannotBeRetired(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)

		_, err := m.manager.SetRetired(ctx, m.admin, m.Classic.ID, true)
		require.NoError(t, err)

		_, err = m.manager.SetRetired(ctx, m.admin, m.Open.ID, true)
		assert.ErrorIs(t, err, formats.ErrLastFormat)

		list, err := m.Formats.List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 1, "players can still choose it")
	})
}

func TestAnAdminOrdersTheFormats(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)

		ordered, err := m.manager.Reorder(ctx, m.admin, []string{m.Classic.ID, m.Open.ID})
		require.NoError(t, err)
		require.Len(t, ordered, 2)
		assert.Equal(t, m.Classic.ID, ordered[0].ID)

		list, err := m.Formats.List(ctx)
		require.NoError(t, err)
		assert.Equal(t, m.Classic.ID, list[0].ID, "what players see follows")

		log := m.logged(t)
		assert.Equal(t, adminlog.FormatsReordered, log[0].Action)
		assert.Equal(t, "DM-01 to DM-12, Open", log[0].Details)

		for _, ids := range [][]string{
			{m.Classic.ID},
			{m.Classic.ID, m.Classic.ID},
			{m.Classic.ID, "nothing"},
			{m.Classic.ID, m.Open.ID, "nothing"},
		} {
			_, err := m.manager.Reorder(ctx, m.admin, ids)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%v: got %v", ids, err)
			assert.Equal(t, "formatIds", invalid.Field)
		}
		list, err = m.Formats.List(ctx)
		require.NoError(t, err)
		assert.Equal(t, m.Classic.ID, list[0].ID, "a refused order changes nothing")
	})
}

func TestOnlyAnAdminManagesFormats(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)
		player, err := users.NewService(db, storetest.Logger()).ByID(ctx, m.Kokujo.ID)
		require.NoError(t, err)

		_, err = m.manager.Create(ctx, player, m.input("Legends"))
		assert.ErrorIs(t, err, users.ErrNotAdmin)
		_, err = m.manager.Update(ctx, player, m.Classic.ID, m.input("Legends"))
		assert.ErrorIs(t, err, users.ErrNotAdmin)
		_, err = m.manager.SetRetired(ctx, player, m.Classic.ID, true)
		assert.ErrorIs(t, err, users.ErrNotAdmin)
		_, err = m.manager.Reorder(ctx, player, []string{m.Classic.ID, m.Open.ID})
		assert.ErrorIs(t, err, users.ErrNotAdmin)
		assert.Empty(t, m.logged(t))
	})
}

func TestADecksLegalityFollowsAChangedFormat(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManaging(t, db)
		deck := m.Legal[m.Shobu.ID]

		_, _, err := m.Lobby.PrepareDeck(ctx, m.Shobu, m.Open.ID, deck.ID)
		require.NoError(t, err, "legal to begin with")

		_, err = m.manager.Update(ctx, m.admin, m.Open.ID, formats.Input{
			Name:        m.Open.Name,
			Description: m.Open.Description,
			AllSets:     true,
			Limits:      map[string]int{deck.Cards[0].CardID: 0},
		})
		require.NoError(t, err)

		_, _, err = m.Lobby.PrepareDeck(ctx, m.Shobu, m.Open.ID, deck.ID)
		assert.Error(t, err, "its card is banned now")
	})
}
