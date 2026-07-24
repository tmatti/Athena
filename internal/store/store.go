// Package store is the data-access layer. All SQL lives here.
package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// SubjectOwner is the single resource owner every credential resolves to.
// There is deliberately no user table yet; if multi-user ever lands, tokens
// and middleware keep their shape and only this value becomes per-user.
const SubjectOwner = "owner"

// ErrInvalidCursor is returned when a pagination cursor cannot be decoded.
var ErrInvalidCursor = errors.New("invalid cursor")

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
