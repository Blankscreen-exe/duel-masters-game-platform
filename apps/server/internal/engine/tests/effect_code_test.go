package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNoRecoverInEffectCode keeps the path from a prompt to the match event loop
// clear of recovers.
//
// Player.NextAction aborts by panicking when the match is disposed while it is
// waiting, and the match event loop recovers that abort. A recover in between
// would swallow it, so the effect would carry on resolving for a game that no
// longer exists. If effect code ever genuinely needs one, it has to re-raise
// anything match.IsMatchDisposed reports.
func TestNoRecoverInEffectCode(t *testing.T) {
	recoverCall := regexp.MustCompile(`\brecover\(\)`)

	var offenders []string

	for _, directory := range []string{filepath.Join("..", "game", "cards"), filepath.Join("..", "game", "fx")} {
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			for number, line := range strings.Split(string(contents), "\n") {
				if recoverCall.MatchString(line) {
					offenders = append(offenders, filepath.ToSlash(path)+":"+strconv.Itoa(number+1)+": "+strings.TrimSpace(line))
				}
			}

			return nil
		})

		require.NoError(t, err)
	}

	require.Empty(
		t,
		offenders,
		"card and effect code must not recover, or it will swallow the abort Player.NextAction raises "+
			"when the match is disposed while a prompt is open:\n%s",
		strings.Join(offenders, "\n"),
	)
}
