package paging_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/paging"
	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

func TestNormalizeFillsInTheFirstPageAndTheDefaultSize(t *testing.T) {
	request := paging.Request{}
	require.NoError(t, request.Normalize(25, 100))

	assert.Equal(t, paging.Request{Page: 1, PageSize: 25}, request)
	assert.Equal(t, 0, request.Offset())
}

func TestNormalizeKeepsAPageAskedFor(t *testing.T) {
	request := paging.Request{Page: 3, PageSize: 10}
	require.NoError(t, request.Normalize(25, 100))

	assert.Equal(t, paging.Request{Page: 3, PageSize: 10}, request)
	assert.Equal(t, 20, request.Offset())
}

func TestNormalizeRefusesPagesOutOfRange(t *testing.T) {
	for _, test := range []struct {
		request paging.Request
		field   string
	}{
		{paging.Request{Page: -1}, "page"},
		{paging.Request{PageSize: -1}, "pageSize"},
		{paging.Request{PageSize: 101}, "pageSize"},
	} {
		err := test.request.Normalize(25, 100)

		invalid, ok := validation.As(err)
		require.True(t, ok, "%+v: got %v", test.request, err)
		assert.Equal(t, test.field, invalid.Field)
	}
}
