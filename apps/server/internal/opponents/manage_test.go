package opponents_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/computer"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/decks"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games/gamestest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby/lobbytest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/opponents/opponentstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// manager is a world with the opponents service and an admin to manage them.
type manager struct {
	*lobbytest.World
	opponents *opponents.Service
	admin     users.User
}

func newManager(t *testing.T, db *gorm.DB) manager {
	t.Helper()

	w := lobbytest.NewWorld(t, db, opponents.Settings()...)
	admin, err := users.NewService(db, storetest.Logger()).ByID(context.Background(), w.Shobu.ID)
	require.NoError(t, err)
	require.True(t, admin.IsAdmin(), "the first account is the admin")

	return manager{World: w, opponents: opponents.NewService(db, w.Catalogue, w.Formats), admin: admin}
}

// input is a valid opponent with a whole deck.
func (m manager) input(name string) opponents.Input {
	return opponents.Input{
		Name:        name,
		Description: "  Hits hard.  ",
		Style:       "rush",
		Strength:    computer.Easy,
		FormatID:    m.Open.ID,
		Rules:       decks.Rules{CardTypes: opponents.CardTypes},
		Cards:       gamestest.LegalDeck(m.CardIDs),
	}
}

// logged is what the admin log holds, the latest first.
func (m manager) logged(t *testing.T) []adminlog.Entry {
	t.Helper()

	page, err := adminlog.NewService(m.DB).List(context.Background(), adminlog.Query{Request: paging.Request{}})
	require.NoError(t, err)
	return page.Entries
}

func TestAnAdminAddsAnOpponentWhichStartsRetired(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)

		added, err := m.opponents.Create(ctx, m.admin, m.input("Grim"))
		require.NoError(t, err)
		assert.Equal(t, "Grim", added.Name)
		assert.Equal(t, "Hits hard.", added.Description, "trimmed")
		assert.Equal(t, computer.Easy, added.Strength)
		assert.False(t, added.Enabled, "not offered until an admin says so")
		assert.Equal(t, decks.Size, decks.Count(added.Cards))

		account, err := users.NewService(db, storetest.Logger()).ByID(ctx, added.UserID)
		require.NoError(t, err)
		assert.True(t, account.Computer, "under an account nobody can log in to")

		offered, err := m.opponents.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, offered)
		all, err := m.opponents.All(ctx)
		require.NoError(t, err)
		assert.Len(t, all, 1)

		second, err := m.opponents.Create(ctx, m.admin, m.input("Second"))
		require.NoError(t, err)
		all, err = m.opponents.All(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{added.ID, second.ID}, []string{all[0].ID, all[1].ID}, "the newest last")

		log := m.logged(t)
		require.Len(t, log, 2)
		assert.Equal(t, adminlog.ComputerOpponentAdded, log[0].Action)
		assert.Equal(t, "Second", log[0].Subject)
	})
}

func TestARetiredOpponentsDeckMayBeUnfinishedButAnOfferedOnesMayNot(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)

		unfinished := m.input("Grim")
		unfinished.Cards = gamestest.LegalDeck(m.CardIDs)[:2]
		added, err := m.opponents.Create(ctx, m.admin, unfinished)
		require.NoError(t, err, "work in progress can be saved")
		assert.Equal(t, 8, decks.Count(added.Cards))

		var unplayable *opponents.UnplayableError
		_, err = m.opponents.SetEnabled(ctx, m.admin, added.ID, true)
		require.ErrorAs(t, err, &unplayable, "but not offered")
		assert.Equal(t, decks.ProblemSize, unplayable.Problems[0].Kind)

		finished, err := m.opponents.Update(ctx, m.admin, added.ID, m.input("Grim"))
		require.NoError(t, err)
		assert.Equal(t, decks.Size, decks.Count(finished.Cards), "the deck is replaced, not added to")
		offered, err := m.opponents.SetEnabled(ctx, m.admin, added.ID, true)
		require.NoError(t, err)
		assert.True(t, offered.Enabled)

		_, err = m.opponents.Update(ctx, m.admin, added.ID, unfinished)
		require.ErrorAs(t, err, &unplayable, "an offered opponent must stay playable")
		kept, err := m.opponents.ByID(ctx, added.ID)
		require.NoError(t, err)
		assert.Equal(t, decks.Size, decks.Count(kept.Cards))

		retired, err := m.opponents.SetEnabled(ctx, m.admin, added.ID, false)
		require.NoError(t, err)
		assert.False(t, retired.Enabled)
		_, err = m.opponents.Update(ctx, m.admin, added.ID, unfinished)
		assert.NoError(t, err, "once retired it can be worked on")
	})
}

func TestAnOfferedOpponentKeepsToItsOwnRules(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)

		added, err := m.opponents.Create(ctx, m.admin, m.input("Grim"))
		require.NoError(t, err)
		_, err = m.opponents.SetEnabled(ctx, m.admin, added.ID, true)
		require.NoError(t, err)

		noSpells := m.input("Grim")
		noSpells.Rules.CardTypes = []string{"Castle"}
		var unplayable *opponents.UnplayableError
		_, err = m.opponents.Update(ctx, m.admin, added.ID, noSpells)
		require.ErrorAs(t, err, &unplayable)
		assert.Equal(t, decks.ProblemCardType, unplayable.Problems[0].Kind)
	})
}

func TestAChangeRenamesAndIsLogged(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)

		added, err := m.opponents.Create(ctx, m.admin, m.input("Grim"))
		require.NoError(t, err)

		changed := m.input("Grimmer")
		changed.Style = "wall"
		changed.Strength = computer.Normal
		changed.FormatID = m.Classic.ID
		changed.Rules = decks.Rules{MaxCivilizations: 5, CardTypes: []string{"Spell", "Creature"}}
		updated, err := m.opponents.Update(ctx, m.admin, added.ID, changed)
		require.NoError(t, err)
		assert.Equal(t, "Grimmer", updated.Name)
		assert.Equal(t, "wall", updated.Style)
		assert.Equal(t, computer.Normal, updated.Strength)
		assert.Equal(t, m.Classic.ID, updated.FormatID)
		assert.Equal(t, 5, updated.Rules.MaxCivilizations)
		assert.Equal(t, []string{"Creature", "Spell"}, updated.Rules.CardTypes, "in the game's order")
		assert.Equal(t, added.UserID, updated.UserID, "the same account, so its history stays")

		log := m.logged(t)
		assert.Equal(t, adminlog.ComputerOpponentChanged, log[0].Action)
		assert.Equal(t, "Grimmer", log[0].Subject)
		assert.Equal(t, "Was called Grim", log[0].Details)

		// Changing only the case of its own name is no clash with itself.
		_, err = m.opponents.Update(ctx, m.admin, added.ID, m.input("GRIMMER"))
		assert.NoError(t, err)
	})
}

func TestAnOpponentsNameCannotBeOneAnAccountHas(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)

		_, err := m.opponents.Create(ctx, m.admin, m.input("kokujo"))
		assert.ErrorIs(t, err, users.ErrUsernameTaken)

		added, err := m.opponents.Create(ctx, m.admin, m.input("Grim"))
		require.NoError(t, err)
		_, err = m.opponents.Update(ctx, m.admin, added.ID, m.input("Mimi"))
		assert.ErrorIs(t, err, users.ErrUsernameTaken)

		kept, err := m.opponents.ByID(ctx, added.ID)
		require.NoError(t, err)
		assert.Equal(t, "Grim", kept.Name, "nothing changed")
	})
}

func TestOpponentInputIsChecked(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)

		long := make([]byte, opponents.DescriptionMaxLength+1)
		for i := range long {
			long[i] = 'x'
		}
		for field, change := range map[string]func(*opponents.Input){
			"username":         func(i *opponents.Input) { i.Name = "a" },
			"description":      func(i *opponents.Input) { i.Description = " " },
			"style":            func(i *opponents.Input) { i.Style = "sulking" },
			"strength":         func(i *opponents.Input) { i.Strength = "hard" },
			"formatId":         func(i *opponents.Input) { i.FormatID = "nothing" },
			"maxCivilizations": func(i *opponents.Input) { i.Rules.MaxCivilizations = 6 },
			"cardTypes":        func(i *opponents.Input) { i.Rules.CardTypes = nil },
			"cards":            func(i *opponents.Input) { i.Cards = []decks.Entry{{CardID: "nothing", Count: 1}} },
		} {
			input := m.input("Grim")
			change(&input)
			_, err := m.opponents.Create(ctx, m.admin, input)
			invalid, ok := validation.As(err)
			require.True(t, ok, "%s: %v", field, err)
			assert.Equal(t, field, invalid.Field)
		}

		tooLong := m.input("Grim")
		tooLong.Description = string(long)
		_, err := m.opponents.Create(ctx, m.admin, tooLong)
		invalid, ok := validation.As(err)
		require.True(t, ok)
		assert.Equal(t, "description", invalid.Field)

		unknownType := m.input("Grim")
		unknownType.Rules.CardTypes = []string{"Creature", "Sandwich"}
		_, err = m.opponents.Create(ctx, m.admin, unknownType)
		invalid, ok = validation.As(err)
		require.True(t, ok)
		assert.Equal(t, "cardTypes", invalid.Field)

		all, err := m.opponents.All(ctx)
		require.NoError(t, err)
		assert.Empty(t, all, "nothing was added")
	})
}

func TestOnlyAnAdminManagesOpponents(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)
		added, err := m.opponents.Create(ctx, m.admin, m.input("Grim"))
		require.NoError(t, err)

		_, err = m.opponents.Create(ctx, m.Kokujo, m.input("Another"))
		assert.ErrorIs(t, err, users.ErrNotAdmin)
		_, err = m.opponents.Update(ctx, m.Kokujo, added.ID, m.input("Another"))
		assert.ErrorIs(t, err, users.ErrNotAdmin)
		_, err = m.opponents.SetEnabled(ctx, m.Kokujo, added.ID, true)
		assert.ErrorIs(t, err, users.ErrNotAdmin)

		all, err := m.opponents.All(ctx)
		require.NoError(t, err)
		require.Len(t, all, 1)
		assert.Equal(t, "Grim", all[0].Name)
		assert.False(t, all[0].Enabled)
		assert.Len(t, m.logged(t), 1)
	})
}

func TestRetiringAndOfferingAreLoggedOnceEach(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)
		added, err := m.opponents.Create(ctx, m.admin, m.input("Grim"))
		require.NoError(t, err)

		_, err = m.opponents.SetEnabled(ctx, m.admin, added.ID, false)
		require.NoError(t, err)
		assert.Len(t, m.logged(t), 1, "already retired: nothing to record")

		_, err = m.opponents.SetEnabled(ctx, m.admin, added.ID, true)
		require.NoError(t, err)
		assert.Equal(t, adminlog.ComputerOpponentOffered, m.logged(t)[0].Action)

		_, err = m.opponents.SetEnabled(ctx, m.admin, added.ID, false)
		require.NoError(t, err)
		assert.Equal(t, adminlog.ComputerOpponentRetired, m.logged(t)[0].Action)

		_, err = m.opponents.SetEnabled(ctx, m.admin, "nothing", true)
		assert.ErrorIs(t, err, opponents.ErrNotFound)
		_, err = m.opponents.Update(ctx, m.admin, "nothing", m.input("Other"))
		assert.ErrorIs(t, err, opponents.ErrNotFound)
	})
}

func TestPlayersAreNotOfferedAnOpponentThatCannotPlay(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w := newChallengeWorld(t, db)

		offers, err := w.challenges.Offers(ctx)
		require.NoError(t, err)
		require.Len(t, offers, 1)

		// Its deck stops being whole, as a format's rules changing would make it
		// stop being legal.
		require.NoError(t, db.Table("computer_opponent_cards").
			Where("computer_opponent_id = ? AND card_id = ?", w.grim.ID, w.CardIDs[0]).Delete(nil).Error)
		offers, err = w.challenges.Offers(ctx)
		require.NoError(t, err)
		assert.Empty(t, offers)
	})
}

func TestPlayersAreNotOfferedAnOpponentWhoseFormatIsRetired(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		w := newChallengeWorld(t, db)

		require.NoError(t, db.Table("formats").Where("id = ?", w.Open.ID).Update("retired_at", time.Now()).Error)

		offers, err := w.challenges.Offers(ctx)
		require.NoError(t, err)
		assert.Empty(t, offers)

		_, err = w.challenges.Start(ctx, w.Shobu, w.grim.ID, w.Legal[w.Shobu.ID].ID)
		assert.ErrorIs(t, err, opponents.ErrUnavailable)
	})
}

func TestARetiredOpponentCannotBePlayedAfterAnAdminRetiresIt(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		ctx := context.Background()
		m := newManager(t, db)
		grim := opponentstest.Add(t, db, "Grim", m.Open.ID, gamestest.LegalDeck(m.CardIDs))

		_, err := m.opponents.SetEnabled(ctx, m.admin, grim.ID, false)
		require.NoError(t, err)

		_, err = m.opponents.Get(ctx, grim.ID)
		assert.ErrorIs(t, err, opponents.ErrNotFound)
	})
}
