package secrets

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBox(t *testing.T, fill byte) *Box {
	t.Helper()

	box, err := NewBox([]byte(strings.Repeat(string(rune(fill)), KeySize)))
	require.NoError(t, err)
	return box
}

func TestParseKey(t *testing.T) {
	key := strings.Repeat("k", KeySize)

	parsed, err := ParseKey(base64.StdEncoding.EncodeToString([]byte(key)))
	require.NoError(t, err)
	assert.Equal(t, []byte(key), parsed)

	parsed, err = ParseKey("  " + base64.StdEncoding.EncodeToString([]byte(key)) + "\n")
	require.NoError(t, err, "surrounding space from a file is forgiven")
	assert.Equal(t, []byte(key), parsed)

	_, err = ParseKey("not base64!")
	assert.ErrorContains(t, err, "base64")

	_, err = ParseKey(base64.StdEncoding.EncodeToString([]byte("too short")))
	assert.ErrorContains(t, err, "32 bytes")
}

func TestNewBoxRefusesAKeyOfTheWrongSize(t *testing.T) {
	_, err := NewBox([]byte("short"))
	assert.Error(t, err)
}

func TestSealThenOpen(t *testing.T) {
	box := newTestBox(t, 'a')

	sealed, err := box.Seal([]byte("the password"), "mail")
	require.NoError(t, err)
	assert.NotContains(t, sealed, "the password")

	opened, err := box.Open(sealed, "mail")
	require.NoError(t, err)
	assert.Equal(t, "the password", string(opened))
}

func TestSealingTwiceGivesDifferentText(t *testing.T) {
	box := newTestBox(t, 'a')

	first, err := box.Seal([]byte("same"), "mail")
	require.NoError(t, err)
	second, err := box.Seal([]byte("same"), "mail")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestOpenRefuses(t *testing.T) {
	box := newTestBox(t, 'a')
	sealed, err := box.Seal([]byte("the password"), "mail")
	require.NoError(t, err)

	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(sealed, sealedPrefix))
	require.NoError(t, err)
	raw[len(raw)-1] ^= 1
	tampered := sealedPrefix + base64.RawStdEncoding.EncodeToString(raw)

	cases := map[string]struct {
		box     *Box
		sealed  string
		purpose string
	}{
		"another key":     {newTestBox(t, 'b'), sealed, "mail"},
		"another purpose": {box, sealed, "other"},
		"tampered":        {box, tampered, "mail"},
		"plain text":      {box, "the password", "mail"},
		"too short":       {box, sealedPrefix + "AA", "mail"},
		"not base64":      {box, sealedPrefix + "!!!", "mail"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := test.box.Open(test.sealed, test.purpose)
			assert.ErrorIs(t, err, ErrUnreadable)
		})
	}
}
