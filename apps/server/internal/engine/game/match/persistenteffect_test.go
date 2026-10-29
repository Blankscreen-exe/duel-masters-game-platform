package match

import (
	"sync"
	"testing"
)

func newPersistentEffectTestMatch() *Match {
	return &Match{persistentEffects: make(map[int]PersistentEffect)}
}

// countingEffect is a persistent effect that counts the times it runs and
// hands out its exit function.
func countingEffect(runs *int, exit *func()) PersistentHandlerFunc {
	return func(_ *Context, e func()) {
		*runs++
		*exit = e
	}
}

func runPersistentEffects(m *Match) {
	for _, fx := range m.persistentEffects {
		fx.effect(nil, fx.exit)
	}
}

// Matches run on their own event loops at the same time, so adding persistent
// effects in one must not touch anything another match uses. Run under -race.
func TestMatchesAddPersistentEffectsAtTheSameTime(t *testing.T) {
	const matches = 4
	const effects = 200

	var wg sync.WaitGroup
	results := make([]*Match, matches)
	for i := range matches {
		results[i] = newPersistentEffectTestMatch()
		wg.Add(1)
		go func(m *Match) {
			defer wg.Done()
			for range effects {
				m.ApplyPersistentEffect(func(*Context, func()) {})
			}
		}(results[i])
	}
	wg.Wait()

	for i, m := range results {
		if got := len(m.persistentEffects); got != effects {
			t.Fatalf("match %d has %d persistent effects, want %d", i, got, effects)
		}
	}
}

func TestAPersistentEffectsExitRemovesOnlyItself(t *testing.T) {
	m := newPersistentEffectTestMatch()

	var firstRuns, secondRuns int
	var firstExit, secondExit func()
	m.ApplyPersistentEffect(countingEffect(&firstRuns, &firstExit))
	m.ApplyPersistentEffect(countingEffect(&secondRuns, &secondExit))

	runPersistentEffects(m)
	firstExit()
	runPersistentEffects(m)

	if firstRuns != 1 || secondRuns != 2 {
		t.Fatalf("after the first effect exits, runs are %d and %d, want 1 and 2", firstRuns, secondRuns)
	}

	// Exiting twice changes nothing.
	firstExit()
	runPersistentEffects(m)
	if secondRuns != 3 {
		t.Fatalf("a second exit of the first effect stopped the second one: runs %d, want 3", secondRuns)
	}

	secondExit()
	if len(m.persistentEffects) != 0 {
		t.Fatalf("%d persistent effects left after both exited", len(m.persistentEffects))
	}
}
