// Package paging holds how every list is asked for one page at a time, so
// the features check pages the same way.
package paging

import (
	"fmt"

	"github.com/Blankscreen-exe/duel-masters-game-platform/apps/server/internal/validation"
)

// Request asks for one page of a list.
type Request struct {
	// Page counts from 1; 0 means the first page.
	Page int
	// PageSize is how many items a page holds; 0 means the list's default.
	PageSize int
}

// Normalize fills in the first page and the default size, and returns a
// *validation.Error for a page below 1 or a size above maxSize.
func (r *Request) Normalize(defaultSize, maxSize int) error {
	switch {
	case r.Page < 0:
		return validation.New("page", "must be 1 or more")
	case r.PageSize < 0 || r.PageSize > maxSize:
		return validation.New("pageSize", fmt.Sprintf("must be between 1 and %d", maxSize))
	}

	if r.Page == 0 {
		r.Page = 1
	}
	if r.PageSize == 0 {
		r.PageSize = defaultSize
	}
	return nil
}

// Offset is how many items come before the page, once normalized.
func (r Request) Offset() int {
	return (r.Page - 1) * r.PageSize
}
