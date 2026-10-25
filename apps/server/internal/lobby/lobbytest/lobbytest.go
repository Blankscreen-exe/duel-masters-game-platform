// Package lobbytest is a directory of who is online that tests set.
package lobbytest

import (
	"slices"
	"strings"
	"sync"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/lobby"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/users"
)

// Directory is who is online, as a test sets it. It is safe for concurrent
// use.
type Directory struct {
	mu     sync.Mutex
	online map[string]lobby.Person
}

var _ lobby.Directory = (*Directory)(nil)

// NewDirectory returns a directory with nobody online.
func NewDirectory() *Directory {
	return &Directory{online: map[string]lobby.Person{}}
}

// Online lists who is online, by name.
func (d *Directory) Online() []lobby.Person {
	d.mu.Lock()
	defer d.mu.Unlock()

	people := make([]lobby.Person, 0, len(d.online))
	for _, person := range d.online {
		people = append(people, person)
	}
	slices.SortFunc(people, func(a, b lobby.Person) int { return strings.Compare(a.Username, b.Username) })
	return people
}

// IsOnline reports whether a player is online.
func (d *Directory) IsOnline(userID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, ok := d.online[userID]
	return ok
}

// Set puts a player online or takes them offline.
func (d *Directory) Set(user users.User, online bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if online {
		d.online[user.ID] = lobby.Person{ID: user.ID, Username: user.Username}
	} else {
		delete(d.online, user.ID)
	}
}
