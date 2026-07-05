package store

import (
	"context"
)

// SearchNotes ranks notes via their chunks. Implemented with the notes
// milestone; returns no results until then.
func (s *Store) SearchNotes(ctx context.Context, p SearchParams) ([]SearchResult, error) {
	return nil, nil
}
