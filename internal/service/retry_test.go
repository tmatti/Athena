package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/tmatti/athena/internal/embed"
	"github.com/tmatti/athena/internal/store"
	"github.com/tmatti/athena/internal/testdb"
)

// poisonEmbedder fails any Embed call whose batch contains the poison text,
// but succeeds for batches of only good texts. This models an oversized row
// that permanently fails and would starve a shared batch call.
type poisonEmbedder struct {
	inner  embed.FakeEmbedder
	poison string
}

func (p *poisonEmbedder) Dimensions() int { return p.inner.Dimensions() }

func (p *poisonEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	for _, t := range texts {
		if t == p.poison {
			return nil, errors.New("poison text exceeds token limit")
		}
	}
	return p.inner.Embed(ctx, texts)
}

func memoryAttempts(t *testing.T, pool *pgxpool.Pool, id string) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT embed_attempts FROM memories WHERE id = $1`, id).Scan(&n))
	return n
}

func TestRetryPoisonBatchDoesNotStarve(t *testing.T) {
	pool := testdb.Pool(t)
	st := store.New(pool)
	const poison = "this poison row can never be embedded"
	pe := &poisonEmbedder{inner: embed.FakeEmbedder{Dims: 1536}, poison: poison}
	b := New(st, pe, slog.New(slog.DiscardHandler))
	ctx := context.Background()

	// The good memory embeds fine synchronously; reset it to pending so the
	// retry batch will include it alongside the poison row.
	good, err := b.CreateMemory(ctx, "a perfectly good memory", nil, nil)
	require.NoError(t, err)
	require.Equal(t, "ok", good.EmbedStatus)
	_, err = pool.Exec(ctx,
		`UPDATE memories SET embedding = NULL, embed_status = 'pending' WHERE id = $1`, good.ID)
	require.NoError(t, err)

	// The poison memory fails synchronously; clear its backoff so it is
	// eligible for the retry pass (keeping its incremented attempt count).
	bad, err := b.CreateMemory(ctx, poison, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "failed", bad.EmbedStatus)
	_, err = pool.Exec(ctx,
		`UPDATE memories SET embed_last_attempt_at = NULL WHERE id = $1`, bad.ID)
	require.NoError(t, err)
	attemptsBefore := memoryAttempts(t, pool, bad.ID)

	// One retry pass: the batched call fails on the poison row, but the
	// per-item fallback must still recover the good row.
	b.retryPendingEmbeds(ctx)

	gotGood, err := b.GetMemory(ctx, good.ID)
	require.NoError(t, err)
	require.Equal(t, "ok", gotGood.EmbedStatus, "good row must be re-embedded despite the poison row")

	gotBad, err := b.GetMemory(ctx, bad.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", gotBad.EmbedStatus)
	require.Greater(t, memoryAttempts(t, pool, bad.ID), attemptsBefore,
		"poison row's attempts must increment, pushing it into backoff")
}
