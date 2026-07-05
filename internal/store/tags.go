package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func (s *Store) ListTags(ctx context.Context) ([]TagCount, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tag, count(*)::int FROM (
			SELECT unnest(tags) AS tag FROM memories
			UNION ALL
			SELECT unnest(tags) AS tag FROM notes
		) t
		GROUP BY tag
		ORDER BY count(*) DESC, tag`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (TagCount, error) {
		var tc TagCount
		err := row.Scan(&tc.Tag, &tc.Count)
		return tc, err
	})
}
