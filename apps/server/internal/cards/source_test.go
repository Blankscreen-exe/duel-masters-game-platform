package cards

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/cards/cardstest"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/games"
)

var (
	printing          = cardstest.Printing
	writeCardDatabase = cardstest.Write
)

func TestReadSourceReadsPrintingsSetsAndPictures(t *testing.T) {
	dir := writeCardDatabase(t, []map[string]any{
		printing("0002", "Brawler Zyler", "DM-10", "5/110"),
		printing("0001", "Brawler Zyler", "DM-02", "80/110"),
		printing("0003", "Promo Dragon", "PROMO", "P1/Y1"),
	}, []string{"0001", "0002"}, []string{"dm-02"})

	source, err := ReadSource(dir)
	require.NoError(t, err)

	require.Len(t, source.Printings, 3)
	assert.True(t, source.Printings[0].HasImage)
	assert.False(t, source.Printings[2].HasImage, "picture 0003 is missing")
	assert.Equal(t, "Brawler Zyler", source.Printings[0].Name)
	require.NotNil(t, source.Printings[0].Power)
	assert.Equal(t, 2000, *source.Printings[0].Power)

	assert.Equal(t, []SourceSet{
		{Code: "DM-02", IconPath: "data/icons/sets/dm-02.png"},
		{Code: "DM-10"},
		{Code: "PROMO"},
	}, source.Sets, "numbered sets by number, promotional cards last")

	assert.Len(t, source.Fingerprint, 64)
}

func TestReadSourceFingerprintFollowsTheContent(t *testing.T) {
	first, err := ReadSource(writeCardDatabase(t, []map[string]any{printing("0001", "A", "DM-01", "1/1")}, nil, nil))
	require.NoError(t, err)
	same, err := ReadSource(writeCardDatabase(t, []map[string]any{printing("0001", "A", "DM-01", "1/1")}, nil, nil))
	require.NoError(t, err)
	changed, err := ReadSource(writeCardDatabase(t, []map[string]any{printing("0001", "B", "DM-01", "1/1")}, nil, nil))
	require.NoError(t, err)

	assert.Equal(t, first.Fingerprint, same.Fingerprint)
	assert.NotEqual(t, first.Fingerprint, changed.Fingerprint)
}

func TestReadSourceRefusesBrokenFiles(t *testing.T) {
	_, err := ReadSource(t.TempDir())
	assert.Error(t, err, "no cards.json")

	broken := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(broken, "data"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(broken, "data", "cards.json"), []byte("{not json"), 0o600))
	_, err = ReadSource(broken)
	assert.Error(t, err, "malformed JSON")

	noID := printing("", "Nameless id", "DM-01", "1/1")
	noName := printing("0002", " ", "DM-01", "2/1")
	noSet := printing("0003", "Setless", "", "3/1")
	twice := printing("0004", "Twice", "DM-01", "4/1")
	escape := printing("0005", "Escape", "DM-01", "5/1")
	escape["image_path"] = "../../secret.txt"
	absolute := printing("0006", "Absolute", "DM-01", "6/1")
	absolute["image_path"] = "/etc/passwd"
	drive := printing("0007", "Drive", "DM-01", "7/1")
	drive["image_path"] = "C:/Windows/win.ini"
	backslash := printing("0008", "Backslash", "DM-01", "8/1")
	backslash["image_path"] = `images\..\..\secret.txt`
	hidden := printing("0009", "Hidden", "DM-01", "9/1")
	hidden["image_path"] = "images/../../secret.txt"

	_, err = ReadSource(writeCardDatabase(t, []map[string]any{noID, noName, noSet, twice, twice, escape, absolute, drive, backslash, hidden}, nil, nil))
	require.Error(t, err)
	for _, problem := range []string{
		"has no id", "has no name", "has no set", "used twice",
		`"../../secret.txt" leaves the data folder`,
		`"/etc/passwd" leaves the data folder`,
		`"C:/Windows/win.ini" leaves the data folder`,
		`"images\\..\\..\\secret.txt" leaves the data folder`,
		`"images/../../secret.txt" leaves the data folder`,
	} {
		assert.Contains(t, err.Error(), problem)
	}
}

func TestBuildGroupsReprintsInReleaseOrder(t *testing.T) {
	later := printing("0002", "Solar Ray", "DM-10", "5/110")
	later["rules_text"] = "Reprinted text"
	source, err := ReadSource(writeCardDatabase(t, []map[string]any{
		later,
		printing("0001", "Solar Ray", "DM-01", "29/110"),
		printing("0003", "Aqua Hulcus", "DM-01", "33/110"),
	}, []string{"0001", "0002", "0003"}, nil))
	require.NoError(t, err)

	cards, problems := Build(source, nil)
	assert.Empty(t, problems)
	require.Len(t, cards, 2)

	assert.Equal(t, "Aqua Hulcus", cards[0].Name, "cards are in name order")
	solar := cards[1]
	assert.Equal(t, "0001", solar.ID, "a card is known by its first printing")
	assert.Equal(t, "Blocker", solar.RulesText, "its details are the first printing's")
	require.Len(t, solar.Printings, 2)
	assert.Equal(t, "DM-01", solar.Printings[0].SetCode)
	assert.Equal(t, "DM-10", solar.Printings[1].SetCode)
	assert.Equal(t, "images/0001.webp", solar.Printings[0].ImagePath)
	assert.False(t, solar.Playable())
}

func TestBuildKeepsCardsWhoseNamesDifferOnlyInPunctuationApart(t *testing.T) {
	source, err := ReadSource(writeCardDatabase(t, []map[string]any{
		printing("0001", "Aqua-Hulcus", "DM-01", "1/1"),
		printing("0002", "Aqua Hulcus", "DM-01", "2/1"),
	}, []string{"0001", "0002"}, nil))
	require.NoError(t, err)

	cards, _ := Build(source, nil)
	assert.Len(t, cards, 2)
}

func TestBuildLinksTheEngineCards(t *testing.T) {
	source, err := ReadSource(writeCardDatabase(t, []map[string]any{
		printing("0001", "Bolshack Dragon", "DM-01", "69/110"),
		printing("0002", "Überdragon Zaschack", "PROMO", "L6/Y2"),
		printing("0003", "Hypersprint Warrior Uzesol", "DM-12", "49/55"),
		printing("0004", "Not Yet Implemented", "DM-20", "1/1"),
	}, []string{"0001", "0002", "0003", "0004"}, nil))
	require.NoError(t, err)

	cards, problems := Build(source, []games.PlayableCard{
		{EngineID: "engine-bolshack", Name: "Bolshack Dragon", Set: "dm-01"},
		{EngineID: "engine-zaschack", Name: "Uberdragon Zaschack", Set: "promo"},
		{EngineID: "engine-uzesol", Name: "Hypersprint Warior Uzesol", Set: "dm-12"},
	})
	assert.Empty(t, problems)

	engineIDs := map[string]string{}
	for _, card := range cards {
		engineIDs[card.Name] = card.EngineID
	}
	assert.Equal(t, map[string]string{
		"Bolshack Dragon":            "engine-bolshack",
		"Überdragon Zaschack":        "engine-zaschack",
		"Hypersprint Warrior Uzesol": "engine-uzesol",
		"Not Yet Implemented":        "",
	}, engineIDs, "matched exactly, ignoring accents, and through the alias; the rest are not playable")
}

func TestBuildReportsWhatItCannotLink(t *testing.T) {
	source, err := ReadSource(writeCardDatabase(t, []map[string]any{
		printing("0001", "Bolshack Dragon", "DM-01", "69/110"),
		printing("0002", "Aqua Hulcus", "DM-01", "33/110"),
		printing("0003", "Aqua-Hulcus", "DM-02", "1/1"),
		printing("0004", "Pictureless", "DM-01", "4/110"),
	}, []string{"0001", "0002", "0003"}, nil))
	require.NoError(t, err)

	_, problems := Build(source, []games.PlayableCard{
		{EngineID: "engine-1", Name: "Bolshack Dragon"},
		{EngineID: "engine-2", Name: "bolshack dragon"},
		{EngineID: "engine-3", Name: "Missing Card"},
		{EngineID: "engine-4", Name: "AQUA HULCUS!"},
	})

	assert.ElementsMatch(t, []string{
		`the engine's "bolshack dragon" (engine-2) and engine-1 are both "Bolshack Dragon" in the card database`,
		`the engine's "Missing Card" (engine-3) is not in the card database`,
		`the engine's "AQUA HULCUS!" (engine-4) matches several cards in the card database`,
		`printing 0004 of "Pictureless" has no picture`,
	}, problems)
}

func TestCollectorNumbersSortByNumber(t *testing.T) {
	numbers := []string{"S10/S10", "110/110", "S2/S10", "9/110", "L6/Y2", "S1/S10"}
	sorted := append([]string(nil), numbers...)
	sortCollectorNumbers(sorted)

	assert.Equal(t, []string{"S1/S10", "S2/S10", "L6/Y2", "9/110", "S10/S10", "110/110"}, sorted)
}

func sortCollectorNumbers(numbers []string) {
	for i := 1; i < len(numbers); i++ {
		for j := i; j > 0 && compareCollectorNumbers(numbers[j-1], numbers[j]) > 0; j-- {
			numbers[j-1], numbers[j] = numbers[j], numbers[j-1]
		}
	}
}

// realCardDatabase is the card database linked into the repository.
const realCardDatabase = "../../../../resources/duelmasters_db"

func TestTheRealCardDatabaseHasEveryEngineCard(t *testing.T) {
	if _, err := os.Stat(realCardDatabase); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("the card database must be checked out on CI (git submodule update --init): %v", err)
		}
		t.Skip("the card database is not checked out; run git submodule update --init")
	}

	source, err := ReadSource(realCardDatabase)
	require.NoError(t, err)

	playable := games.PlayableCards()
	cards, problems := Build(source, playable)
	assert.Empty(t, problems)

	linked := 0
	for _, card := range cards {
		if card.Playable() {
			linked++
			require.NotEmpty(t, card.Printings[0].ImagePath, "%s has a picture", card.Name)
		}
	}
	assert.Equal(t, len(playable), linked, "every card the engine plays is in the catalogue")
}
