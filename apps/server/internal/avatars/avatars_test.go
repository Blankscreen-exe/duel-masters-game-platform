package avatars_test

import (
	"bytes"
	"image"
	_ "image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/avatars"
)

func TestEveryPictureIsAWholeOneOfTheSameSize(t *testing.T) {
	ids := avatars.IDs()
	require.Len(t, ids, 170)
	assert.IsIncreasing(t, ids, "in order of name, each once")

	versions := map[string]string{}
	for _, id := range ids {
		picture, found := avatars.ByID(id)
		require.True(t, found, id)
		assert.Equal(t, id, picture.ID)

		decoded, format, err := image.Decode(bytes.NewReader(picture.Data))
		require.NoError(t, err, id)
		assert.Equal(t, "png", format, id)
		assert.Equal(t, image.Pt(62, 53), decoded.Bounds().Size(), id)

		assert.NotContains(t, versions, picture.Version, "%s is the same picture as another", id)
		versions[picture.Version] = id
	}
}

func TestAnUnknownPictureIsNotFound(t *testing.T) {
	for _, id := range []string{"", "nothing", "../avatars", "dm01-creatures-01.png"} {
		_, found := avatars.ByID(id)
		assert.False(t, found, id)
	}
}
