// Package cardstest lays out small card databases for tests, in the format of
// the real one (resources/duelmasters_db).
package cardstest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Printing is a card database entry with every field the catalogue reads
// filled in. Tests change the fields that matter to them.
func Printing(id, name, set, collector string) map[string]any {
	return map[string]any{
		"id":               id,
		"name":             name,
		"card_type":        "Creature",
		"cost":             3,
		"power":            2000,
		"power_text":       "2000",
		"civilizations":    []string{"Fire"},
		"races":            []string{"Human"},
		"rarity":           "Common",
		"rarity_code":      "c",
		"collector_number": collector,
		"set_code":         set,
		"rules_text":       "Blocker",
		"abilities":        []map[string]any{{"text": "Blocker", "reminder": nil}},
		"keywords":         []string{"blocker"},
		"categories":       []string{},
		"image_path":       "images/" + id + ".webp",
	}
}

// Write lays out a card database in a new temporary folder and returns it:
// the printings, a picture for each id in pictures (its content is "picture
// <id>"), and an icon for each lowercase set code in setIcons.
func Write(t *testing.T, printings []map[string]any, pictures []string, setIcons []string) string {
	t.Helper()

	dir := t.TempDir()
	write := func(path string, content []byte) {
		full := filepath.Join(dir, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
		require.NoError(t, os.WriteFile(full, content, 0o600))
	}

	raw, err := json.Marshal(printings)
	require.NoError(t, err)
	write("data/cards.json", raw)

	for _, id := range pictures {
		write("data/images/"+id+".webp", []byte("picture "+id))
	}
	for _, set := range setIcons {
		write("data/icons/sets/"+set+".png", []byte("icon "+set))
	}

	return dir
}
