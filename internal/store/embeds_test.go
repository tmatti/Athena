package store

import (
	"context"
	"testing"

	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/require"

	"github.com/tmatti/athena/internal/embed"
	"github.com/tmatti/athena/internal/testdb"
)

// idsOf returns the set of PendingEmbed ids for easy membership checks.
func idsOf(pending []PendingEmbed) map[string]bool {
	set := make(map[string]bool, len(pending))
	for _, p := range pending {
		set[p.ID] = true
	}
	return set
}

func TestListPendingEmbedsBackoffAndOrdering(t *testing.T) {
	s := New(testdb.Pool(t))
	ctx := context.Background()

	// A row that has failed many times with a recent attempt is in backoff.
	backedOff, err := s.CreateMemory(ctx, "often failing row", nil, nil)
	require.NoError(t, err)
	_, err = s.pool.Exec(ctx,
		`UPDATE memories SET embed_status = 'failed', embed_attempts = 6, embed_last_attempt_at = now()
		 WHERE id = $1`, backedOff.ID)
	require.NoError(t, err)

	// A fresh pending row (never attempted) must be eligible.
	fresh, err := s.CreateMemory(ctx, "fresh pending row", nil, nil)
	require.NoError(t, err)

	pending, err := s.ListPendingEmbeds(ctx, 10)
	require.NoError(t, err)
	got := idsOf(pending)
	require.True(t, got[fresh.ID], "fresh pending row should be eligible")
	require.False(t, got[backedOff.ID], "recently-failed high-attempt row should be in backoff")

	// Simulate enough time passing that the backed-off row is eligible again.
	_, err = s.pool.Exec(ctx,
		`UPDATE memories SET embed_last_attempt_at = NULL WHERE id = $1`, backedOff.ID)
	require.NoError(t, err)

	pending, err = s.ListPendingEmbeds(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	// Ordering: the fresh (0-attempt) row comes before the high-attempt one.
	require.Equal(t, fresh.ID, pending[0].ID)
	require.Equal(t, backedOff.ID, pending[1].ID)
}

func TestSetMemoryEmbeddingContentGuard(t *testing.T) {
	s := New(testdb.Pool(t))
	fe := &embed.FakeEmbedder{Dims: testDims}
	ctx := context.Background()

	m, err := s.CreateMemory(ctx, "the original content", nil, nil)
	require.NoError(t, err)

	vecs, err := fe.Embed(ctx, []string{"the original content"})
	require.NoError(t, err)
	vec := pgvector.NewVector(vecs[0])

	// A write against stale (non-matching) content must be a no-op: the row
	// stays pending with a NULL embedding.
	require.NoError(t, s.SetMemoryEmbedding(ctx, m.ID, "some different content", vec))
	got, err := s.GetMemory(ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, "pending", got.EmbedStatus)

	var hasEmbedding bool
	require.NoError(t, s.pool.QueryRow(ctx,
		`SELECT embedding IS NOT NULL FROM memories WHERE id = $1`, m.ID).Scan(&hasEmbedding))
	require.False(t, hasEmbedding, "stale write must not store an embedding")

	// A write with matching content succeeds and flips status to ok.
	require.NoError(t, s.SetMemoryEmbedding(ctx, m.ID, "the original content", vec))
	got, err = s.GetMemory(ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, "ok", got.EmbedStatus)
}
