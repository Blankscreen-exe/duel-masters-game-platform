package cards

// EngineCard is a card the rules engine implements, as the catalogue links it
// to the card database. Package games lists them.
type EngineCard struct {
	// EngineID is the engine's id for the card, which games are played with.
	EngineID string
	// Name is the name the engine gives the card.
	Name string
	// Set is the engine's code for the set the card is registered in, such as
	// "dm-01".
	Set string
}
