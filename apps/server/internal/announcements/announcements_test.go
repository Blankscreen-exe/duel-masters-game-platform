package announcements_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/adminlog"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/announcements"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/clocktest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/store/storetest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

type counter struct{ changes atomic.Int32 }

func (c *counter) AnnouncementsChanged() { c.changes.Add(1) }

type fixture struct {
	db       *gorm.DB
	service  *announcements.Service
	accounts *users.Service
	clock    *clocktest.Clock
	told     *counter
	log      *adminlog.Service
	admin    users.User
}

func newFixture(t *testing.T, db *gorm.DB) *fixture {
	t.Helper()

	clock := clocktest.New(time.Now().UTC())
	accounts := users.NewService(db, storetest.Logger())
	told := &counter{}
	f := &fixture{
		db:       db,
		service:  announcements.NewService(db, accounts, told, clock.Now, storetest.Logger()),
		accounts: accounts,
		clock:    clock,
		told:     told,
		log:      adminlog.NewService(db),
	}
	f.admin = f.join(t, "Admin")
	require.True(t, f.admin.IsAdmin())
	clock.Advance(time.Second)
	return f
}

// join adds a player, as of now on the fixture's clock.
func (f *fixture) join(t *testing.T, name string) users.User {
	t.Helper()

	user, err := f.accounts.Create(context.Background(), name, "hash")
	require.NoError(t, err)
	// Accounts take their time from the database's clock; the tests go by theirs.
	require.NoError(t, f.setJoined(user.ID))
	user, err = f.accounts.ByID(context.Background(), user.ID)
	require.NoError(t, err)
	return user
}

func (f *fixture) setJoined(id string) error {
	return f.db.Table("users").Where("id = ?", id).Update("created_at", f.clock.Now().UTC()).Error
}

// reread is a player's account as it now is.
func (f *fixture) reread(t *testing.T, user users.User) users.User {
	t.Helper()

	fresh, err := f.accounts.ByID(context.Background(), user.ID)
	require.NoError(t, err)
	return fresh
}

// post adds an announcement a moment later.
func (f *fixture) post(t *testing.T, title string, important bool) announcements.Announcement {
	t.Helper()

	f.clock.Advance(time.Second)
	posted, err := f.service.Create(context.Background(), f.admin, announcements.Input{Title: title, Body: "Details.", Important: important})
	require.NoError(t, err)
	return posted
}

func titles(list []announcements.Announcement) []string {
	found := make([]string, 0, len(list))
	for _, item := range list {
		found = append(found, item.Title)
	}
	return found
}

func TestAnImportantAnnouncementPopsUpUntilClosed(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		player := f.join(t, "Shobu")
		other := f.join(t, "Kokujo")

		f.post(t, "Plain news", false)
		first := f.post(t, "Maintenance tonight", true)
		f.post(t, "New set", true)

		mine, err := f.service.For(ctx, player)
		require.NoError(t, err)
		assert.Equal(t, 3, mine.Unread)
		assert.Equal(t, []string{"Maintenance tonight", "New set"}, titles(mine.Popups), "only the important, the oldest first")

		require.NoError(t, f.service.Dismiss(ctx, player, first.ID))
		require.NoError(t, f.service.Dismiss(ctx, player, first.ID), "closing it twice is no error")

		mine, err = f.service.For(ctx, player)
		require.NoError(t, err)
		assert.Equal(t, []string{"New set"}, titles(mine.Popups))

		theirs, err := f.service.For(ctx, other)
		require.NoError(t, err)
		assert.Len(t, theirs.Popups, 2, "one player closing it leaves it for the others")

		assert.ErrorIs(t, f.service.Dismiss(ctx, player, "no-such-id"), announcements.ErrNotFound)
	})
}

func TestNewPlayersAreNotGivenOldNews(t *testing.T) {
	f := newFixture(t, storetest.Open(t, "sqlite"))
	ctx := context.Background()

	f.post(t, "From before", true)
	f.clock.Advance(time.Second)
	newcomer := f.join(t, "Newcomer")
	f.post(t, "From after", true)

	mine, err := f.service.For(ctx, newcomer)
	require.NoError(t, err)
	assert.Equal(t, 1, mine.Unread)
	assert.Equal(t, []string{"From after"}, titles(mine.Popups))

	page, err := f.service.List(ctx, paging.Request{})
	require.NoError(t, err)
	assert.Equal(t, []string{"From after", "From before"}, titles(page.Announcements), "they can still read it all")
}

func TestLookingClearsTheUnreadCountButNotThePopups(t *testing.T) {
	f := newFixture(t, storetest.Open(t, "sqlite"))
	ctx := context.Background()
	player := f.join(t, "Shobu")

	f.post(t, "First", true)
	f.clock.Advance(time.Second)
	require.NoError(t, f.service.MarkSeen(ctx, player))

	mine, err := f.service.For(ctx, f.reread(t, player))
	require.NoError(t, err)
	assert.Zero(t, mine.Unread)
	assert.Len(t, mine.Popups, 1, "reading the page is not closing the popup")

	f.post(t, "Second", false)
	mine, err = f.service.For(ctx, f.reread(t, player))
	require.NoError(t, err)
	assert.Equal(t, 1, mine.Unread)
}

func TestChangingAnAnnouncement(t *testing.T) {
	ctx := context.Background()

	setUp := func(t *testing.T) (*fixture, users.User, announcements.Announcement) {
		t.Helper()

		f := newFixture(t, storetest.Open(t, "sqlite"))
		player := f.join(t, "Shobu")
		posted := f.post(t, "Maintenance", true)
		require.NoError(t, f.service.Dismiss(ctx, player, posted.ID))
		f.clock.Advance(time.Second)
		require.NoError(t, f.service.MarkSeen(ctx, player))
		f.clock.Advance(time.Second)
		return f, f.reread(t, player), posted
	}

	t.Run("a correction stays quiet", func(t *testing.T) {
		f, player, posted := setUp(t)

		changed, err := f.service.Update(ctx, f.admin, posted.ID, announcements.Input{Title: "Maintenance tonight", Body: "Fixed.", Important: true})
		require.NoError(t, err)
		assert.Equal(t, "Maintenance tonight", changed.Title)
		assert.True(t, changed.AnnouncedAt.Equal(posted.AnnouncedAt))

		mine, err := f.service.For(ctx, player)
		require.NoError(t, err)
		assert.Zero(t, mine.Unread)
		assert.Empty(t, mine.Popups)
	})

	t.Run("show again reaches those who closed it", func(t *testing.T) {
		f, player, posted := setUp(t)

		_, err := f.service.Update(ctx, f.admin, posted.ID, announcements.Input{Title: "Maintenance moved", Body: "Now tomorrow.", Important: true, ShowAgain: true})
		require.NoError(t, err)

		mine, err := f.service.For(ctx, player)
		require.NoError(t, err)
		assert.Equal(t, 1, mine.Unread)
		assert.Equal(t, []string{"Maintenance moved"}, titles(mine.Popups))
	})

	t.Run("show again means nothing for one not important", func(t *testing.T) {
		f, player, posted := setUp(t)

		_, err := f.service.Update(ctx, f.admin, posted.ID, announcements.Input{Title: "Done", Body: "All over.", Important: false, ShowAgain: true})
		require.NoError(t, err)

		mine, err := f.service.For(ctx, player)
		require.NoError(t, err)
		assert.Zero(t, mine.Unread)
		assert.Empty(t, mine.Popups)
	})

	t.Run("making one important sends it out", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player := f.join(t, "Shobu")
		posted := f.post(t, "Quiet news", false)

		f.clock.Advance(time.Second)
		_, err := f.service.Update(ctx, f.admin, posted.ID, announcements.Input{Title: "Loud news", Body: "Read this.", Important: true})
		require.NoError(t, err)

		mine, err := f.service.For(ctx, player)
		require.NoError(t, err)
		assert.Equal(t, []string{"Loud news"}, titles(mine.Popups))
	})

	t.Run("taking away important stops the popup", func(t *testing.T) {
		f := newFixture(t, storetest.Open(t, "sqlite"))
		player := f.join(t, "Shobu")
		posted := f.post(t, "Loud news", true)

		_, err := f.service.Update(ctx, f.admin, posted.ID, announcements.Input{Title: "Loud news", Body: "Over.", Important: false})
		require.NoError(t, err)

		mine, err := f.service.For(ctx, player)
		require.NoError(t, err)
		assert.Empty(t, mine.Popups)
	})

	t.Run("one that is gone", func(t *testing.T) {
		f, _, _ := setUp(t)

		_, err := f.service.Update(ctx, f.admin, "no-such-id", announcements.Input{Title: "A", Body: "B"})
		assert.ErrorIs(t, err, announcements.ErrNotFound)
	})
}

func TestDeletingAnAnnouncement(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		player := f.join(t, "Shobu")
		posted := f.post(t, "Mistake", true)
		kept := f.post(t, "Kept", true)
		require.NoError(t, f.service.Dismiss(ctx, player, kept.ID))

		require.NoError(t, f.service.Delete(ctx, f.admin, posted.ID))

		mine, err := f.service.For(ctx, player)
		require.NoError(t, err)
		assert.Empty(t, mine.Popups)
		assert.Equal(t, 1, mine.Unread)

		assert.ErrorIs(t, f.service.Delete(ctx, f.admin, posted.ID), announcements.ErrNotFound)

		log, err := f.log.List(ctx, adminlog.Query{Action: adminlog.AnnouncementDeleted})
		require.NoError(t, err)
		require.Len(t, log.Entries, 1)
		assert.Equal(t, "Mistake", log.Entries[0].Subject)
		assert.Equal(t, "Admin", log.Entries[0].AdminName)
	})
}

func TestDeletingAPlayerKeepsTheAnnouncement(t *testing.T) {
	storetest.Each(t, func(t *testing.T, db *gorm.DB) {
		f := newFixture(t, db)
		ctx := context.Background()
		player := f.join(t, "Shobu")
		posted := f.post(t, "News", true)
		require.NoError(t, f.service.Dismiss(ctx, player, posted.ID))

		require.NoError(t, db.Delete(&users.User{}, "id = ?", player.ID).Error)

		page, err := f.service.List(ctx, paging.Request{})
		require.NoError(t, err)
		assert.Equal(t, 1, page.Total)
	})
}

func TestOnlyAdminsChangeAnnouncements(t *testing.T) {
	f := newFixture(t, storetest.Open(t, "sqlite"))
	ctx := context.Background()
	player := f.join(t, "Shobu")
	posted := f.post(t, "News", false)
	input := announcements.Input{Title: "Mine", Body: "Mine."}

	_, err := f.service.Create(ctx, player, input)
	assert.ErrorIs(t, err, users.ErrNotAdmin)
	_, err = f.service.Update(ctx, player, posted.ID, input)
	assert.ErrorIs(t, err, users.ErrNotAdmin)
	assert.ErrorIs(t, f.service.Delete(ctx, player, posted.ID), users.ErrNotAdmin)
}

func TestTheRulesOfTheText(t *testing.T) {
	f := newFixture(t, storetest.Open(t, "sqlite"))
	ctx := context.Background()

	cases := map[string]struct {
		input announcements.Input
		field string
	}{
		"no title":       {announcements.Input{Title: "  ", Body: "B"}, "title"},
		"long title":     {announcements.Input{Title: strings.Repeat("t", announcements.TitleMaxLength+1), Body: "B"}, "title"},
		"two-line title": {announcements.Input{Title: "A\nB", Body: "B"}, "title"},
		"no text":        {announcements.Input{Title: "T", Body: " \n "}, "body"},
		"long text":      {announcements.Input{Title: "T", Body: strings.Repeat("b", announcements.BodyMaxLength+1)}, "body"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := f.service.Create(ctx, f.admin, test.input)
			invalid, ok := validation.As(err)
			require.True(t, ok, "got %v", err)
			assert.Equal(t, test.field, invalid.Field)
		})
	}

	t.Run("text is tidied, keeping its lines", func(t *testing.T) {
		posted, err := f.service.Create(ctx, f.admin, announcements.Input{Title: " Title ", Body: " One\r\nTwo\n"})
		require.NoError(t, err)
		assert.Equal(t, "Title", posted.Title)
		assert.Equal(t, "One\nTwo", posted.Body)
	})
}

func TestListingGoesPageByPageTheLatestFirst(t *testing.T) {
	f := newFixture(t, storetest.Open(t, "sqlite"))
	ctx := context.Background()
	for _, title := range []string{"One", "Two", "Three"} {
		f.post(t, title, false)
	}

	page, err := f.service.List(ctx, paging.Request{Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total)
	assert.Equal(t, []string{"One"}, titles(page.Announcements))

	_, err = f.service.List(ctx, paging.Request{PageSize: announcements.MaxPageSize + 1})
	_, invalid := validation.As(err)
	assert.True(t, invalid)
}

func TestPlayersAreToldOfEveryChange(t *testing.T) {
	f := newFixture(t, storetest.Open(t, "sqlite"))
	ctx := context.Background()

	posted := f.post(t, "News", false)
	assert.EqualValues(t, 1, f.told.changes.Load())

	_, err := f.service.Update(ctx, f.admin, posted.ID, announcements.Input{Title: "News", Body: "More."})
	require.NoError(t, err)
	assert.EqualValues(t, 2, f.told.changes.Load())

	require.NoError(t, f.service.Delete(ctx, f.admin, posted.ID))
	assert.EqualValues(t, 3, f.told.changes.Load())

	_, err = f.service.Create(ctx, f.admin, announcements.Input{Title: ""})
	require.Error(t, err)
	assert.EqualValues(t, 3, f.told.changes.Load(), "a refused change tells nobody")
}
