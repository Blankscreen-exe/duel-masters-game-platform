// Package games is the server's only door to the rules engine in
// internal/engine: everything else learns about cards the engine can play, and
// later starts and follows games, through this package.
package games

import (
	"sort"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/cards"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/game/match"
)

// PlayableCard is a card the rules engine implements.
type PlayableCard struct {
	// EngineID is the engine's id for the card, which games are played with.
	EngineID string
	// Name is the name the engine gives the card.
	Name string
	// Set is the engine's code for the set the card is registered in, such as
	// "dm-01".
	Set string
}

// PlayableCards lists every card the engine implements, sorted by engine id.
func PlayableCards() []PlayableCard {
	playable := make([]PlayableCard, 0)
	for set, constructors := range cards.Sets {
		for id, construct := range *constructors {
			if construct == nil {
				continue
			}

			// Constructing a card only sets its fields and handlers; it needs no
			// match.
			card := &match.Card{}
			construct(card)

			playable = append(playable, PlayableCard{EngineID: id, Name: card.Name, Set: set})
		}
	}

	sort.Slice(playable, func(i, j int) bool { return playable[i].EngineID < playable[j].EngineID })

	return playable
}
