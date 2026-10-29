package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A player coming back takes their seat beside the event loop, so the game
// must be able to go on sending to them, and reading who they are, while
// their connection changes. Run under -race.
func TestComingBackWhileTheGameIsBusy(t *testing.T) {
	m, _, alice, bob, _ := startedMatch(t, lifecycleOptions(time.Hour))

	var mu sync.Mutex
	latest := alice
	current := func() client {
		mu.Lock()
		defer mu.Unlock()
		return latest
	}

	reconnected := make(chan struct{})
	go func() {
		defer close(reconnected)
		for range 10 {
			again := connect(m, hostID, "Alice")
			mu.Lock()
			latest = again
			mu.Unlock()
			time.Sleep(time.Millisecond)
		}
	}()

	// Ending turns sends both players the state, chat naming them, and
	// warnings to whoever's turn it is not.
	for range 10 {
		current().send(m, map[string]any{"header": "end_turn"})
		bob.send(m, map[string]any{"header": "end_turn"})
		time.Sleep(time.Millisecond)
	}
	<-reconnected

	require.Eventually(t, func() bool { return !m.HasPendingEvents() }, lifecycleTimeout, 5*time.Millisecond,
		"the turns sent are played")
	assert.False(t, m.IsEnding(), "the game goes on")

	final := connect(m, hostID, "Alice")
	final.waitFor(t, 0, "state_update")
	assert.True(t, m.Connected(hostID))
}
