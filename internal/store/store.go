// Package store handles SQLite persistence for bingos, their cells and the
// validation history.
//
// ValidationError messages and line names end up in front of the user, so they
// are written in French like the rest of the UI.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO)
)

const (
	MinSize = 3
	MaxSize = 6

	maxTitleLen = 120
	maxDescLen  = 500
	maxLabelLen = 200
	maxNoteLen  = 1000

	// Fixed width, so lexicographic order matches chronological order.
	timeLayout = "2006-01-02T15:04:05.000Z"
)

// History event kinds.
const (
	EventCreated     = "created"
	EventValidated   = "validated"
	EventUnvalidated = "unvalidated"
	EventLine        = "line"
	EventFull        = "full"
	EventArchived    = "archived"
	EventUnarchived  = "unarchived"
)

var (
	ErrNotFound = errors.New("not found")
	ErrArchived = errors.New("bingo is archived")
)

// ValidationError reports invalid input; its message is meant for the user.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

type Cell struct {
	ID        int64      `json:"id"`
	Position  int        `json:"position"`
	Label     string     `json:"label"`
	CheckedAt *time.Time `json:"checkedAt"`
	Note      string     `json:"note"`
}

type Bingo struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Size        int        `json:"size"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"` // last activity
	ArchivedAt  *time.Time `json:"archivedAt"`
	Checked     int        `json:"checked"`
	Total       int        `json:"total"`
	Pattern     []bool     `json:"pattern"` // checked state of each cell, by position
	Lines       []Line     `json:"lines"`   // completed lines
	Cells       []Cell     `json:"cells,omitempty"`
}

type Event struct {
	ID         int64     `json:"id"`
	BingoID    int64     `json:"bingoId"`
	BingoTitle string    `json:"bingoTitle"`
	CellID     *int64    `json:"cellId"`
	Kind       string    `json:"kind"`
	Label      string    `json:"label"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"createdAt"`
}

// BingoInput describes a bingo to create or update.
type BingoInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Size        int      `json:"size"`
	Cells       []string `json:"cells"` // labels, row by row
}

func (in *BingoInput) normalize() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.Title == "" {
		return invalid("Le titre est obligatoire.")
	}
	if utf8.RuneCountInString(in.Title) > maxTitleLen {
		return invalid("Le titre ne doit pas dépasser %d caractères.", maxTitleLen)
	}
	if utf8.RuneCountInString(in.Description) > maxDescLen {
		return invalid("La description ne doit pas dépasser %d caractères.", maxDescLen)
	}
	if in.Size < MinSize || in.Size > MaxSize {
		return invalid("La taille de la grille doit être comprise entre %d et %d.", MinSize, MaxSize)
	}
	if len(in.Cells) != in.Size*in.Size {
		return invalid("Une grille %d×%d doit contenir %d cases (%d reçues).", in.Size, in.Size, in.Size*in.Size, len(in.Cells))
	}
	for i, label := range in.Cells {
		label = strings.TrimSpace(label)
		if label == "" {
			return invalid("La case %d est vide.", i+1)
		}
		if utf8.RuneCountInString(label) > maxLabelLen {
			return invalid("La case %d dépasse %d caractères.", i+1, maxLabelLen)
		}
		in.Cells[i] = label
	}
	return nil
}

type Store struct{ db *sql.DB }

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
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating database: %w", err)
	}
	return s, nil
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

func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	for i := version; i < len(migrations); i++ {
		tx, err := s.db.Begin()
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

// querier is satisfied by both *sql.DB and *sql.Tx.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Store) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func now() string { return time.Now().UTC().Format(timeLayout) }

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

const bingoColumns = `b.id, b.title, b.description, b.size, b.created_at, b.archived_at,
	COALESCE((SELECT MAX(e.created_at) FROM events e WHERE e.bingo_id = b.id), b.created_at)`

type scanner interface{ Scan(dest ...any) error }

func scanBingo(row scanner) (*Bingo, error) {
	var (
		b                Bingo
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

// summarize computes the derived fields from the cells' checked state.
func (b *Bingo) summarize(pattern []bool) {
	b.Pattern = pattern
	b.Total = len(pattern)
	b.Checked = 0
	for _, c := range pattern {
		if c {
			b.Checked++
		}
	}
	b.Lines = CompletedLines(b.Size, pattern)
}

func getBingo(ctx context.Context, q querier, id int64) (*Bingo, error) {
	b, err := scanBingo(q.QueryRowContext(ctx, `SELECT `+bingoColumns+` FROM bingos b WHERE b.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.QueryContext(ctx,
		`SELECT id, position, label, checked_at, note FROM cells WHERE bingo_id = ? ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pattern := make([]bool, 0, b.Size*b.Size)
	for rows.Next() {
		var (
			c       Cell
			checked sql.NullString
		)
		if err := rows.Scan(&c.ID, &c.Position, &c.Label, &checked, &c.Note); err != nil {
			return nil, err
		}
		c.CheckedAt = parseNullTime(checked)
		b.Cells = append(b.Cells, c)
		pattern = append(pattern, c.CheckedAt != nil)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	b.summarize(pattern)
	return b, nil
}

func addEvent(ctx context.Context, q querier, bingoID int64, cellID *int64, kind, label, note, at string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO events (bingo_id, cell_id, kind, label, note, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		bingoID, cellID, kind, label, note, at)
	return err
}

// List returns every bingo (without cell details), most recently active first.
func (s *Store) List(ctx context.Context) ([]Bingo, error) {
	bingos := make([]Bingo, 0)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT `+bingoColumns+` FROM bingos b ORDER BY 7 DESC, b.id DESC`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			b, err := scanBingo(rows)
			if err != nil {
				return err
			}
			bingos = append(bingos, *b)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()

		cells, err := tx.QueryContext(ctx, `SELECT bingo_id, checked_at IS NOT NULL FROM cells ORDER BY bingo_id, position`)
		if err != nil {
			return err
		}
		defer cells.Close()
		patterns := make(map[int64][]bool, len(bingos))
		for cells.Next() {
			var (
				id      int64
				checked bool
			)
			if err := cells.Scan(&id, &checked); err != nil {
				return err
			}
			patterns[id] = append(patterns[id], checked)
		}
		if err := cells.Err(); err != nil {
			return err
		}
		for i := range bingos {
			bingos[i].summarize(patterns[bingos[i].ID])
		}
		return nil
	})
	return bingos, err
}

// Get returns a bingo with its cells.
func (s *Store) Get(ctx context.Context, id int64) (*Bingo, error) {
	return getBingo(ctx, s.db, id)
}

func insertBingo(ctx context.Context, tx *sql.Tx, in BingoInput) (int64, error) {
	ts := now()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO bingos (title, description, size, created_at) VALUES (?, ?, ?, ?)`,
		in.Title, in.Description, in.Size, ts)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for pos, label := range in.Cells {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO cells (bingo_id, position, label) VALUES (?, ?, ?)`, id, pos, label); err != nil {
			return 0, err
		}
	}
	return id, addEvent(ctx, tx, id, nil, EventCreated, "", "", ts)
}

// Create stores a new bingo.
func (s *Store) Create(ctx context.Context, in BingoInput) (*Bingo, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
	var out *Bingo
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		id, err := insertBingo(ctx, tx, in)
		if err != nil {
			return err
		}
		out, err = getBingo(ctx, tx, id)
		return err
	})
	return out, err
}

// Update changes a bingo's title, description and cell labels.
// The grid size and the cells' checked state are left untouched.
func (s *Store) Update(ctx context.Context, id int64, in BingoInput) (*Bingo, error) {
	var out *Bingo
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		b, err := getBingo(ctx, tx, id)
		if err != nil {
			return err
		}
		if b.ArchivedAt != nil {
			return ErrArchived
		}
		in.Size = b.Size
		if err := in.normalize(); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE bingos SET title = ?, description = ? WHERE id = ?`, in.Title, in.Description, id); err != nil {
			return err
		}
		for pos, label := range in.Cells {
			if _, err := tx.ExecContext(ctx,
				`UPDATE cells SET label = ? WHERE bingo_id = ? AND position = ?`, label, id, pos); err != nil {
				return err
			}
		}
		out, err = getBingo(ctx, tx, id)
		return err
	})
	return out, err
}

// Delete permanently removes a bingo, its cells and its history.
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM bingos WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetArchived archives or unarchives a bingo. An archived bingo is read-only.
func (s *Store) SetArchived(ctx context.Context, id int64, archived bool) (*Bingo, error) {
	var out *Bingo
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		b, err := getBingo(ctx, tx, id)
		if err != nil {
			return err
		}
		if (b.ArchivedAt != nil) != archived {
			ts := now()
			var value any
			kind := EventUnarchived
			if archived {
				value, kind = ts, EventArchived
			}
			if _, err := tx.ExecContext(ctx, `UPDATE bingos SET archived_at = ? WHERE id = ?`, value, id); err != nil {
				return err
			}
			if err := addEvent(ctx, tx, id, nil, kind, "", "", ts); err != nil {
				return err
			}
		}
		out, err = getBingo(ctx, tx, id)
		return err
	})
	return out, err
}

// Duplicate creates a fresh bingo with the cells of an existing one, optionally
// shuffled. An empty title falls back to the source title.
func (s *Store) Duplicate(ctx context.Context, id int64, title string, shuffle bool) (*Bingo, error) {
	var out *Bingo
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		src, err := getBingo(ctx, tx, id)
		if err != nil {
			return err
		}
		in := BingoInput{Title: title, Description: src.Description, Size: src.Size}
		if strings.TrimSpace(in.Title) == "" {
			in.Title = src.Title
		}
		for _, c := range src.Cells {
			in.Cells = append(in.Cells, c.Label)
		}
		if shuffle {
			rand.Shuffle(len(in.Cells), func(i, j int) { in.Cells[i], in.Cells[j] = in.Cells[j], in.Cells[i] })
		}
		if err := in.normalize(); err != nil {
			return err
		}
		newID, err := insertBingo(ctx, tx, in)
		if err != nil {
			return err
		}
		out, err = getBingo(ctx, tx, newID)
		return err
	})
	return out, err
}

func (b *Bingo) cell(id int64) *Cell {
	for i := range b.Cells {
		if b.Cells[i].ID == id {
			return &b.Cells[i]
		}
	}
	return nil
}

// Validate checks a cell and persists it right away, with an optional note.
// Lines completed by this validation are recorded in the history.
// If the cell is already checked, only its note is updated.
func (s *Store) Validate(ctx context.Context, bingoID, cellID int64, note string) (*Bingo, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > maxNoteLen {
		return nil, invalid("La note ne doit pas dépasser %d caractères.", maxNoteLen)
	}
	var out *Bingo
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		before, err := getBingo(ctx, tx, bingoID)
		if err != nil {
			return err
		}
		if before.ArchivedAt != nil {
			return ErrArchived
		}
		cell := before.cell(cellID)
		if cell == nil {
			return ErrNotFound
		}
		if cell.CheckedAt != nil {
			if _, err := tx.ExecContext(ctx, `UPDATE cells SET note = ? WHERE id = ?`, note, cellID); err != nil {
				return err
			}
			out, err = getBingo(ctx, tx, bingoID)
			return err
		}

		ts := now()
		if _, err := tx.ExecContext(ctx,
			`UPDATE cells SET checked_at = ?, note = ? WHERE id = ?`, ts, note, cellID); err != nil {
			return err
		}
		if err := addEvent(ctx, tx, bingoID, &cellID, EventValidated, cell.Label, note, ts); err != nil {
			return err
		}
		after, err := getBingo(ctx, tx, bingoID)
		if err != nil {
			return err
		}
		known := make(map[string]bool, len(before.Lines))
		for _, l := range before.Lines {
			known[l.Name] = true
		}
		for _, l := range after.Lines {
			if !known[l.Name] {
				if err := addEvent(ctx, tx, bingoID, nil, EventLine, l.Name, "", ts); err != nil {
					return err
				}
			}
		}
		if after.Checked == after.Total {
			if err := addEvent(ctx, tx, bingoID, nil, EventFull, "", "", ts); err != nil {
				return err
			}
		}
		out = after
		return nil
	})
	return out, err
}

// Unvalidate unchecks a cell and clears its note.
func (s *Store) Unvalidate(ctx context.Context, bingoID, cellID int64) (*Bingo, error) {
	var out *Bingo
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		b, err := getBingo(ctx, tx, bingoID)
		if err != nil {
			return err
		}
		if b.ArchivedAt != nil {
			return ErrArchived
		}
		cell := b.cell(cellID)
		if cell == nil {
			return ErrNotFound
		}
		if cell.CheckedAt == nil {
			out = b
			return nil
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE cells SET checked_at = NULL, note = '' WHERE id = ?`, cellID); err != nil {
			return err
		}
		if err := addEvent(ctx, tx, bingoID, &cellID, EventUnvalidated, cell.Label, "", now()); err != nil {
			return err
		}
		out, err = getBingo(ctx, tx, bingoID)
		return err
	})
	return out, err
}

// History returns events, most recent first. A bingoID of 0 means all bingos.
func (s *Store) History(ctx context.Context, bingoID int64, limit int) ([]Event, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.bingo_id, b.title, e.cell_id, e.kind, e.label, e.note, e.created_at
		FROM events e JOIN bingos b ON b.id = e.bingo_id
		WHERE ?1 = 0 OR e.bingo_id = ?1
		ORDER BY e.id DESC LIMIT ?2`, bingoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]Event, 0)
	for rows.Next() {
		var (
			e       Event
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
