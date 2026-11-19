// Package store is the SQLite implementation of bingo.Repository. It only
// stores and fetches data: business rules live in package bingo.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO)

	"bingo/internal/bingo"
)

// Fixed width, so lexicographic order matches chronological order.
const timeLayout = "2006-01-02T15:04:05.000Z"

// querier is satisfied by both *sql.DB and *sql.Tx.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store runs its queries either directly on the database or, for the value
// handed out by InTx, inside one transaction.
type Store struct {
	db *sql.DB // nil when bound to a transaction
	q  querier
}

var _ bingo.Repository = (*Store)(nil)

// Open opens the SQLite database at path, creating it if needed.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating database directory: %w", err)
		}
	}
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	db, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	// A single connection: SQLite allows one writer at a time and the load is tiny.
	// Consequence: a result set must be closed before the next query starts.
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating database: %w", err)
	}
	return &Store{db: db, q: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// migrations are applied in order; PRAGMA user_version tracks progress.
// Never edit an existing entry: append a new one.
var migrations = []string{
	`CREATE TABLE bingos (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		title       TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		size        INTEGER NOT NULL,
		created_at  TEXT NOT NULL,
		archived_at TEXT
	);
	CREATE TABLE cells (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		bingo_id   INTEGER NOT NULL REFERENCES bingos(id) ON DELETE CASCADE,
		position   INTEGER NOT NULL,
		label      TEXT NOT NULL,
		checked_at TEXT,
		note       TEXT NOT NULL DEFAULT '',
		UNIQUE (bingo_id, position)
	);
	CREATE TABLE events (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		bingo_id   INTEGER NOT NULL REFERENCES bingos(id) ON DELETE CASCADE,
		cell_id    INTEGER REFERENCES cells(id) ON DELETE SET NULL,
		kind       TEXT NOT NULL,
		label      TEXT NOT NULL DEFAULT '',
		note       TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	CREATE INDEX events_bingo ON events (bingo_id, id);`,
}

func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	for i := version; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// InTx implements bingo.Repository. Nested calls join the current transaction.
func (s *Store) InTx(ctx context.Context, fn func(bingo.Repository) error) error {
	if s.db == nil {
		return fn(s)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(&Store{q: tx}); err != nil {
		return err
	}
	return tx.Commit()
}

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func formatNullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(timeLayout, s)
	return t
}

func parseNullTime(s sql.NullString) *time.Time {
	if !s.Valid {
		return nil
	}
	t := parseTime(s.String)
	return &t
}

// The last column is the date of the most recent event, i.e. the last activity.
const bingoColumns = `b.id, b.title, b.description, b.size, b.created_at, b.archived_at,
	COALESCE((SELECT MAX(e.created_at) FROM events e WHERE e.bingo_id = b.id), b.created_at)`

type scanner interface{ Scan(dest ...any) error }

func scanBingo(row scanner) (*bingo.Bingo, error) {
	var (
		b                bingo.Bingo
		created, updated string
		archived         sql.NullString
	)
	if err := row.Scan(&b.ID, &b.Title, &b.Description, &b.Size, &created, &archived, &updated); err != nil {
		return nil, err
	}
	b.CreatedAt = parseTime(created)
	b.UpdatedAt = parseTime(updated)
	b.ArchivedAt = parseNullTime(archived)
	return &b, nil
}

// cells returns cells grouped by bingo ID, ordered by position.
// A bingoID of 0 means all bingos.
func (s *Store) cells(ctx context.Context, bingoID int64) (map[int64][]bingo.Cell, error) {
	rows, err := s.q.QueryContext(ctx, `
		SELECT bingo_id, id, position, label, checked_at, note FROM cells
		WHERE ?1 = 0 OR bingo_id = ?1
		ORDER BY bingo_id, position`, bingoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cells := make(map[int64][]bingo.Cell)
	for rows.Next() {
		var (
			owner   int64
			c       bingo.Cell
			checked sql.NullString
		)
		if err := rows.Scan(&owner, &c.ID, &c.Position, &c.Label, &checked, &c.Note); err != nil {
			return nil, err
		}
		c.CheckedAt = parseNullTime(checked)
		cells[owner] = append(cells[owner], c)
	}
	return cells, rows.Err()
}

func (s *Store) Bingos(ctx context.Context) ([]bingo.Bingo, error) {
	bingos := make([]bingo.Bingo, 0)
	rows, err := s.q.QueryContext(ctx, `SELECT `+bingoColumns+` FROM bingos b ORDER BY 7 DESC, b.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		b, err := scanBingo(rows)
		if err != nil {
			return nil, err
		}
		bingos = append(bingos, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close() // frees the connection for the next query

	cells, err := s.cells(ctx, 0)
	if err != nil {
		return nil, err
	}
	for i := range bingos {
		bingos[i].Cells = cells[bingos[i].ID]
	}
	return bingos, nil
}

func (s *Store) Bingo(ctx context.Context, id int64) (*bingo.Bingo, error) {
	b, err := scanBingo(s.q.QueryRowContext(ctx, `SELECT `+bingoColumns+` FROM bingos b WHERE b.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, bingo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	cells, err := s.cells(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Cells = cells[id]
	return b, nil
}

func (s *Store) InsertBingo(ctx context.Context, in bingo.Input, at time.Time) (int64, error) {
	res, err := s.q.ExecContext(ctx,
		`INSERT INTO bingos (title, description, size, created_at) VALUES (?, ?, ?, ?)`,
		in.Title, in.Description, in.Size, formatTime(at))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for pos, label := range in.Cells {
		if _, err := s.q.ExecContext(ctx,
			`INSERT INTO cells (bingo_id, position, label) VALUES (?, ?, ?)`, id, pos, label); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (s *Store) UpdateBingo(ctx context.Context, id int64, in bingo.Input) error {
	if _, err := s.q.ExecContext(ctx,
		`UPDATE bingos SET title = ?, description = ? WHERE id = ?`, in.Title, in.Description, id); err != nil {
		return err
	}
	for pos, label := range in.Cells {
		if _, err := s.q.ExecContext(ctx,
			`UPDATE cells SET label = ? WHERE bingo_id = ? AND position = ?`, label, id, pos); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteBingo(ctx context.Context, id int64) error {
	res, err := s.q.ExecContext(ctx, `DELETE FROM bingos WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return bingo.ErrNotFound
	}
	return nil
}

func (s *Store) SetArchivedAt(ctx context.Context, id int64, at *time.Time) error {
	_, err := s.q.ExecContext(ctx, `UPDATE bingos SET archived_at = ? WHERE id = ?`, formatNullTime(at), id)
	return err
}

func (s *Store) SetCellState(ctx context.Context, cellID int64, checkedAt *time.Time, note string) error {
	_, err := s.q.ExecContext(ctx,
		`UPDATE cells SET checked_at = ?, note = ? WHERE id = ?`, formatNullTime(checkedAt), note, cellID)
	return err
}

func (s *Store) AddEvent(ctx context.Context, e bingo.Event) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO events (bingo_id, cell_id, kind, label, note, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		e.BingoID, e.CellID, e.Kind, e.Label, e.Note, formatTime(e.CreatedAt))
	return err
}

func (s *Store) Events(ctx context.Context, bingoID int64, limit int) ([]bingo.Event, error) {
	rows, err := s.q.QueryContext(ctx, `
		SELECT e.id, e.bingo_id, b.title, e.cell_id, e.kind, e.label, e.note, e.created_at
		FROM events e JOIN bingos b ON b.id = e.bingo_id
		WHERE ?1 = 0 OR e.bingo_id = ?1
		ORDER BY e.id DESC LIMIT ?2`, bingoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]bingo.Event, 0)
	for rows.Next() {
		var (
			e       bingo.Event
			cellID  sql.NullInt64
			created string
		)
		if err := rows.Scan(&e.ID, &e.BingoID, &e.BingoTitle, &cellID, &e.Kind, &e.Label, &e.Note, &created); err != nil {
			return nil, err
		}
		if cellID.Valid {
			e.CellID = &cellID.Int64
		}
		e.CreatedAt = parseTime(created)
		events = append(events, e)
	}
	return events, rows.Err()
}
