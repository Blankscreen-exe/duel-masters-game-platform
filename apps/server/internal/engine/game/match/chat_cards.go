package match

import (
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/engine/server"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// recentCardsKept is how many of the cards that last moved or acted are
// remembered, to tell apart copies of a card that both players own when a
// chat line names one of them.
const recentCardsKept = 16

// chatCardZones are the zones a card named in a chat line can be in.
var chatCardZones = []string{BATTLEZONE, SPELLZONE, MANAZONE, GRAVEYARD, HAND, SHIELDZONE, DECK, HIDDENZONE}

// publicZones are the zones both players can see, whose cards the interface
// can find on the board.
var publicZones = map[string]bool{BATTLEZONE: true, SPELLZONE: true, MANAZONE: true, GRAVEYARD: true}

// noteRecent records that a card moved or acted, most recent last.
func (m *Match) noteRecent(cardID string) {
	m.recentCardsMutex.Lock()
	defer m.recentCardsMutex.Unlock()

	for i, noted := range m.recentCards {
		if noted == cardID {
			m.recentCards = append(m.recentCards[:i], m.recentCards[i+1:]...)
			break
		}
	}

	m.recentCards = append(m.recentCards, cardID)

	if len(m.recentCards) > recentCardsKept {
		m.recentCards = m.recentCards[len(m.recentCards)-recentCardsKept:]
	}
}

// recency ranks the cards that last moved or acted, the most recent highest.
// A card that did neither has no rank.
func (m *Match) recency() map[string]int {
	m.recentCardsMutex.Lock()
	defer m.recentCardsMutex.Unlock()

	rank := make(map[string]int, len(m.recentCards))
	for i, id := range m.recentCards {
		rank[id] = i + 1
	}

	return rank
}

// mentionedCards finds the cards a chat line names, in the order it names
// them, so the interface can show whose each one is and open it. A name both
// players have a copy of is taken to mean the copy that last moved or acted,
// and when neither has, whose card it is is left unknown. A card is only
// identified on the board while it is in a zone both players can see.
func (m *Match) mentionedCards(message string) []server.ChatCard {
	byName := m.cardsByName()
	if len(byName) == 0 {
		return nil
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	// The longest name is tried first, so a name that is part of a longer one
	// is not found inside it.
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })

	rank := m.recency()
	used := make(map[string]bool)
	mentioned := make([]server.ChatCard, 0)

	for i := 0; i < len(message); {
		name := nameAt(message, i, names)
		if name == "" {
			_, size := utf8.DecodeRuneInString(message[i:])
			i += size
			continue
		}

		copies := byName[name]
		mentioned = append(mentioned, chatCard(name, copies[0].ImageID, pickMentioned(copies, rank, used)))
		i += len(name)
	}

	if len(mentioned) == 0 {
		return nil
	}

	return mentioned
}

// cardsByName groups every card in the match by its name.
func (m *Match) cardsByName() map[string][]*Card {
	byName := make(map[string][]*Card)

	for _, p := range []*PlayerReference{m.Player1, m.Player2} {
		if p == nil || p.Player == nil {
			continue
		}

		for _, zone := range chatCardZones {
			cards, err := p.Player.Container(zone)
			if err != nil {
				continue
			}

			for _, card := range cards {
				if card.Name != "" {
					byName[card.Name] = append(byName[card.Name], card)
				}
			}
		}
	}

	return byName
}

// nameAt returns the name that starts at position i of the message as a whole
// word, or "" when none does.
func nameAt(message string, i int, names []string) string {
	if i > 0 {
		before, _ := utf8.DecodeLastRuneInString(message[:i])
		if isWordRune(before) {
			return ""
		}
	}

	for _, name := range names {
		if !strings.HasPrefix(message[i:], name) {
			continue
		}

		after, _ := utf8.DecodeRuneInString(message[i+len(name):])
		if i+len(name) < len(message) && isWordRune(after) {
			continue
		}

		return name
	}

	return ""
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// pickMentioned chooses which copy of a card a mention means: the one that
// last moved or acted and is not already named earlier in the line, or else
// the only player's copy there is. It returns nil when the copies belong to
// both players and none of them moved or acted.
func pickMentioned(copies []*Card, rank map[string]int, used map[string]bool) *Card {
	var picked *Card

	for _, card := range copies {
		if used[card.ID] || rank[card.ID] == 0 {
			continue
		}

		if picked == nil || rank[card.ID] > rank[picked.ID] {
			picked = card
		}
	}

	if picked == nil {
		for _, card := range copies {
			if card.Player != copies[0].Player {
				return nil
			}
		}

		for _, card := range copies {
			if !used[card.ID] && (picked == nil || publicZones[card.Zone] && !publicZones[picked.Zone]) {
				picked = card
			}
		}
	}

	if picked != nil {
		used[picked.ID] = true
	}

	return picked
}

// chatCard describes a mentioned card for the interface. Every copy of a card
// looks the same, so it can be shown even when which copy is meant is unknown,
// and then its owner is left empty. Its id is left empty unless it is where
// both players can see it.
func chatCard(name string, imageID string, card *Card) server.ChatCard {
	if card == nil {
		return server.ChatCard{Name: name, ImageID: imageID}
	}

	mentioned := server.ChatCard{
		Name:    name,
		ImageID: imageID,
		Owner:   card.Player.Username(),
	}

	if publicZones[card.Zone] {
		mentioned.CardID = card.ID
	}

	return mentioned
}
