// Package games is the server's only door to the rules engine in
// internal/engine: everything else learns about cards the engine can play,
// starts games, connects to them and hears how they end through this package.
package games

import (
	"sort"
	"sync"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards"
	enginecards "github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// registered makes sure the engine's cards are registered once: the engine
// keeps them in a map that is not safe to change while games run.
var registered sync.Once

// registerCards makes every card the engine implements known to its matches,
// before the first game starts.
func registerCards() {
	registered.Do(func() {
		for _, constructors := range enginecards.Sets {
			for id, construct := range *constructors {
				if construct != nil {
					match.AddCard(id, construct)
				}
			}
		}
	})
}

// PlayableCards lists every card the engine implements, sorted by engine id.
func PlayableCards() []cards.EngineCard {
	playable := make([]cards.EngineCard, 0)
	for set, constructors := range enginecards.Sets {
		for id, construct := range *constructors {
			if construct == nil {
				continue
			}

			// Constructing a card only sets its fields and handlers; it needs no
			// match.
			card := &match.Card{}
			construct(card)

			playable = append(playable, cards.EngineCard{EngineID: id, Name: card.Name, Set: set})
		}
	}

	sort.Slice(playable, func(i, j int) bool { return playable[i].EngineID < playable[j].EngineID })

	return playable
}
