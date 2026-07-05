// Package embed defines the embedding-provider abstraction.
package embed

import (
	"context"
	"hash/fnv"
	"math"
)

type Embedder interface {
	// Embed returns one vector per input text, in order.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
}

// FakeEmbedder produces deterministic pseudo-embeddings for tests: texts that
// share words produce vectors that are close in cosine space. Never touches
// the network.
type FakeEmbedder struct {
	Dims int
	// Err, when set, is returned by Embed to exercise failure paths.
	Err error
}

func (f *FakeEmbedder) Dimensions() int {
	if f.Dims == 0 {
		return 8
	}
	return f.Dims
}

func (f *FakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = f.vector(t)
	}
	return out, nil
}

func (f *FakeEmbedder) vector(text string) []float32 {
	dims := f.Dimensions()
	v := make([]float64, dims)
	start := 0
	for i := 0; i <= len(text); i++ {
		if i == len(text) || text[i] == ' ' || text[i] == '\n' {
			if i > start {
				h := fnv.New32a()
				h.Write([]byte(text[start:i]))
				v[int(h.Sum32())%dims] += 1
			}
			start = i + 1
		}
	}
	var norm float64
	for _, x := range v {
		norm += x * x
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		norm = 1
	}
	out := make([]float32, dims)
	for i, x := range v {
		out[i] = float32(x / norm)
	}
	return out
}
