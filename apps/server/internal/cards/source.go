// Package cards is the card catalogue: every Duel Masters card in the card
// database (resources/duelmasters_db), with its printings and pictures, and
// whether the rules engine can play it.
//
// The card database's files are the source. At startup the server copies them
// into its own database, when they or the engine's cards have changed since the
// last copy; everything else reads the catalogue from the database. Pictures
// stay in the files, and only this package knows where they are.
package cards

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The card database's layout, relative to its folder.
const (
	cardsFile    = "data/cards.json"
	dataFolder   = "data"
	setIconsPath = "icons/sets"
)

// promoSetCode is the set code of promotional cards, which come after every
// numbered set.
const promoSetCode = "PROMO"

// SourcePrinting is one printing of a card as the card database lists it.
type SourcePrinting struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Type            string          `json:"card_type"`
	Cost            int             `json:"cost"`
	Power           *int            `json:"power"`
	PowerText       *string         `json:"power_text"`
	Civilizations   []string        `json:"civilizations"`
	Races           []string        `json:"races"`
	Rarity          string          `json:"rarity"`
	RarityCode      string          `json:"rarity_code"`
	CollectorNumber string          `json:"collector_number"`
	SetCode         string          `json:"set_code"`
	RulesText       string          `json:"rules_text"`
	Abilities       []SourceAbility `json:"abilities"`
	Keywords        []string        `json:"keywords"`
	Categories      []string        `json:"categories"`
	// ImagePath is the picture's path inside the data folder, such as
	// "images/0079.webp".
	ImagePath string `json:"image_path"`
	// HasImage reports whether the picture file exists.
	HasImage bool `json:"-"`
}

// SourceAbility is one line of a card's rules text.
type SourceAbility struct {
	Text     string  `json:"text"`
	Reminder *string `json:"reminder"`
}

// SourceSet is a set cards were printed in.
type SourceSet struct {
	Code string
	// IconPath is the set's icon inside the card database's folder, or empty
	// when it has none.
	IconPath string
}

// Source is what the card database's files hold.
type Source struct {
	// Dir is the card database's folder.
	Dir       string
	Printings []SourcePrinting
	// Sets are in release order: numbered sets first, promotional cards last.
	Sets []SourceSet
	// Fingerprint identifies the files' content, so an unchanged card database
	// need not be copied again.
	Fingerprint string
}

// ReadSource reads the card database in dir. It fails for missing or
// malformed files and for printings breaking the rules the catalogue relies
// on; a missing picture is recorded on its printing instead.
func ReadSource(dir string) (Source, error) {
	raw, err := os.ReadFile(filepath.Join(dir, cardsFile))
	if err != nil {
		return Source{}, fmt.Errorf("reading the card database: %w", err)
	}

	var printings []SourcePrinting
	if err := json.Unmarshal(raw, &printings); err != nil {
		return Source{}, fmt.Errorf("reading the card database's %s: %w", cardsFile, err)
	}

	if err := checkPrintings(printings); err != nil {
		return Source{}, fmt.Errorf("the card database's %s: %w", cardsFile, err)
	}

	for i := range printings {
		if printings[i].ImagePath == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, dataFolder, filepath.FromSlash(printings[i].ImagePath))); err == nil {
			printings[i].HasImage = true
		}
	}

	sum := sha256.Sum256(raw)

	return Source{
		Dir:         dir,
		Printings:   printings,
		Sets:        readSets(dir, printings),
		Fingerprint: fmt.Sprintf("%x", sum),
	}, nil
}

// checkPrintings reports printings without an id, name, type or set, ids used
// twice, and picture paths that would leave the data folder.
func checkPrintings(printings []SourcePrinting) error {
	var problems []error
	seen := make(map[string]bool, len(printings))

	for _, printing := range printings {
		label := fmt.Sprintf("printing %q (%s)", printing.ID, printing.Name)

		switch {
		case printing.ID == "":
			problems = append(problems, fmt.Errorf("a printing of %q has no id", printing.Name))
		case seen[printing.ID]:
			problems = append(problems, fmt.Errorf("%s: the id is used twice", label))
		}
		seen[printing.ID] = true

		if strings.TrimSpace(printing.Name) == "" {
			problems = append(problems, fmt.Errorf("%s has no name", label))
		}
		if printing.Type == "" {
			problems = append(problems, fmt.Errorf("%s has no type", label))
		}
		if printing.SetCode == "" {
			problems = append(problems, fmt.Errorf("%s has no set", label))
		}
		if !insideDataFolder(printing.ImagePath) {
			problems = append(problems, fmt.Errorf("%s: picture path %q leaves the data folder", label, printing.ImagePath))
		}
	}

	return errors.Join(problems...)
}

// insideDataFolder reports whether a picture path stays inside the data
// folder, so no path in the files can make the server read anything else. The
// files use forward slashes on every system, so the rules are the same
// everywhere: no leading slash, no backslash, no drive letter, and no way up
// out of the folder.
func insideDataFolder(picture string) bool {
	if picture == "" {
		return true
	}
	if strings.ContainsAny(picture, `\:`) || path.IsAbs(picture) {
		return false
	}

	clean := path.Clean(picture)
	return clean != ".." && !strings.HasPrefix(clean, "../")
}

// readSets lists the sets printings appear in, in release order, with their
// icons where the card database has one.
func readSets(dir string, printings []SourcePrinting) []SourceSet {
	codes := make(map[string]bool)
	for _, printing := range printings {
		codes[printing.SetCode] = true
	}

	sets := make([]SourceSet, 0, len(codes))
	for code := range codes {
		set := SourceSet{Code: code}
		icon := filepath.ToSlash(filepath.Join(setIconsPath, strings.ToLower(code)+".png"))
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(icon))); err == nil {
			set.IconPath = icon
		}
		sets = append(sets, set)
	}

	sort.Slice(sets, func(i, j int) bool { return setOrder(sets[i].Code) < setOrder(sets[j].Code) })

	return sets
}

// setOrder sorts numbered sets ("DM-07") by number, other codes after them
// alphabetically, and promotional cards last.
func setOrder(code string) string {
	if code == promoSetCode {
		return "2"
	}
	if number, ok := strings.CutPrefix(code, "DM-"); ok {
		if n, err := strconv.Atoi(number); err == nil {
			return fmt.Sprintf("0%06d", n)
		}
	}

	return "1" + code
}
