package settings

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kind is the type of a setting's value.
type Kind string

const (
	KindBoolean Kind = "boolean"
	KindNumber  Kind = "number"
	KindText    Kind = "text"
)

// Definition describes one setting: its key, its type, its default, its limits
// and what it is for. Bool, Number and Text are the definitions features
// declare. A definition is a plain value, so it can be declared once as a
// package variable and passed both to NewService and to the typed getters.
type Definition interface {
	// Describe returns the setting's description, as shown in the admin panel.
	Describe() Info
	// canonical checks a raw value and returns it in its canonical form, which
	// is what is stored, and its typed value.
	canonical(raw string) (string, any, error)
}

// Info describes a setting for the admin panel.
type Info struct {
	Key         string
	Group       string
	Description string
	Kind        Kind
	Default     string
	// Min and Max bound a number setting.
	Min, Max int64
	// MaxLength bounds a text setting, in characters.
	MaxLength int
}

// Bool is a setting that is on or off, stored as "true" or "false".
type Bool struct {
	Key         string
	Group       string
	Description string
	Default     bool
}

func (b Bool) Describe() Info {
	return Info{Key: b.Key, Group: b.Group, Description: b.Description, Kind: KindBoolean, Default: strconv.FormatBool(b.Default)}
}

func (b Bool) canonical(raw string) (string, any, error) {
	switch strings.TrimSpace(raw) {
	case "true":
		return "true", true, nil
	case "false":
		return "false", false, nil
	default:
		return "", nil, errors.New(`must be "true" or "false"`)
	}
}

// Number is a whole-number setting between Min and Max, inclusive.
type Number struct {
	Key         string
	Group       string
	Description string
	Default     int64
	Min, Max    int64
}

func (n Number) Describe() Info {
	return Info{Key: n.Key, Group: n.Group, Description: n.Description, Kind: KindNumber, Default: strconv.FormatInt(n.Default, 10), Min: n.Min, Max: n.Max}
}

func (n Number) canonical(raw string) (string, any, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return "", nil, errors.New("must be a whole number")
	}
	if value < n.Min || value > n.Max {
		return "", nil, fmt.Errorf("must be between %d and %d", n.Min, n.Max)
	}

	return strconv.FormatInt(value, 10), value, nil
}

// Text is a setting holding text of at most MaxLength characters. It is
// stored exactly as given, without trimming.
type Text struct {
	Key         string
	Group       string
	Description string
	Default     string
	MaxLength   int
}

func (t Text) Describe() Info {
	return Info{Key: t.Key, Group: t.Group, Description: t.Description, Kind: KindText, Default: t.Default, MaxLength: t.MaxLength}
}

func (t Text) canonical(raw string) (string, any, error) {
	if !utf8.ValidString(raw) {
		return "", nil, errors.New("must be valid text")
	}
	if utf8.RuneCountInString(raw) > t.MaxLength {
		return "", nil, fmt.Errorf("must be at most %d characters", t.MaxLength)
	}

	return raw, raw, nil
}

// keyFormat is how setting keys look: dot-separated lowercase words, starting
// with the feature that owns the setting, such as "accounts.signups_open".
var keyFormat = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// maxKeyLength matches the size of the settings table's key column.
const maxKeyLength = 100

// check reports what is wrong with a definition, so a feature declaring a
// broken setting fails at startup rather than when an admin edits it.
func check(definition Definition) error {
	info := definition.Describe()

	var problems []error
	if !keyFormat.MatchString(info.Key) || len(info.Key) > maxKeyLength {
		problems = append(problems, fmt.Errorf("key %q must be dot-separated lowercase words, at most %d characters, such as \"accounts.signups_open\"", info.Key, maxKeyLength))
	}
	if strings.TrimSpace(info.Group) == "" {
		problems = append(problems, errors.New("group must not be empty"))
	}
	if strings.TrimSpace(info.Description) == "" {
		problems = append(problems, errors.New("description must not be empty"))
	}

	switch info.Kind {
	case KindNumber:
		if info.Min > info.Max {
			problems = append(problems, fmt.Errorf("min %d is above max %d", info.Min, info.Max))
		}
	case KindText:
		if info.MaxLength <= 0 {
			problems = append(problems, errors.New("max length must be positive"))
		}
	}

	if _, _, err := definition.canonical(info.Default); err != nil {
		problems = append(problems, fmt.Errorf("default %q %w", info.Default, err))
	}

	if err := errors.Join(problems...); err != nil {
		return fmt.Errorf("setting %q: %w", info.Key, err)
	}

	return nil
}
