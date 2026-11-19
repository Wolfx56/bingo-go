package bingo

import (
	"context"
	"time"
)

// Repository is the persistence the Service needs. Implementations only store
// and fetch: every business decision (validation, read-only archives, history
// entries) stays in the Service.
type Repository interface {
	// Bingos returns every bingo with its cells, most recently active first.
	Bingos(ctx context.Context) ([]Bingo, error)
	// Bingo returns one bingo with its cells, or ErrNotFound.
	Bingo(ctx context.Context, id int64) (*Bingo, error)
	// InsertBingo stores a new bingo and its cells and returns its ID.
	InsertBingo(ctx context.Context, in Input, at time.Time) (int64, error)
	// UpdateBingo replaces the title, description and cell labels (by position).
	UpdateBingo(ctx context.Context, id int64, in Input) error
	// DeleteBingo removes a bingo, its cells and its events, or returns ErrNotFound.
	DeleteBingo(ctx context.Context, id int64) error
	// SetArchivedAt sets or clears (nil) the archive date.
	SetArchivedAt(ctx context.Context, id int64, at *time.Time) error
	// SetCellState sets a cell's checked date (nil to uncheck) and note.
	SetCellState(ctx context.Context, cellID int64, checkedAt *time.Time, note string) error

	// AddEvent appends an entry to the history.
	AddEvent(ctx context.Context, e Event) error
	// Events returns history entries, most recent first. A bingoID of 0 means
	// all bingos.
	Events(ctx context.Context, bingoID int64, limit int) ([]Event, error)

	// InTx runs fn atomically: the Repository it receives is bound to a single
	// transaction, committed if fn returns nil and rolled back otherwise.
	InTx(ctx context.Context, fn func(Repository) error) error
}
