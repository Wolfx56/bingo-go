// Package bingo holds the domain model and the business rules of the app: what
// a bingo is, how cells get validated, when a line is won, what goes into the
// history. It knows nothing about HTTP or SQL: callers go through Service, and
// persistence goes through the Repository interface.
//
// ValidationError messages and line names end up in front of the user, so they
// are written in French like the rest of the UI.
package bingo

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MinSize = 3
	MaxSize = 6

	maxTitleLen = 120
	maxDescLen  = 500
	maxLabelLen = 200
	maxNoteLen  = 1000

	maxHistory = 1000
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
	ID        int64
	Position  int // 0 to size*size-1, row by row
	Label     string
	CheckedAt *time.Time
	Note      string
}

func (c Cell) Checked() bool { return c.CheckedAt != nil }

type Bingo struct {
	ID          int64
	Title       string
	Description string
	Size        int // the grid is Size×Size
	CreatedAt   time.Time
	UpdatedAt   time.Time // last activity
	ArchivedAt  *time.Time
	Cells       []Cell // ordered by position
}

// Archived reports whether the bingo is read-only.
func (b *Bingo) Archived() bool { return b.ArchivedAt != nil }

// Pattern returns the checked state of each cell, by position.
func (b *Bingo) Pattern() []bool {
	pattern := make([]bool, len(b.Cells))
	for i, c := range b.Cells {
		pattern[i] = c.Checked()
	}
	return pattern
}

// CheckedCount returns how many cells are checked.
func (b *Bingo) CheckedCount() int {
	n := 0
	for _, c := range b.Cells {
		if c.Checked() {
			n++
		}
	}
	return n
}

// Full reports whether every cell is checked.
func (b *Bingo) Full() bool { return b.CheckedCount() == len(b.Cells) }

// Lines returns the completed rows, columns and diagonals.
func (b *Bingo) Lines() []Line { return CompletedLines(b.Size, b.Pattern()) }

func (b *Bingo) cell(id int64) *Cell {
	for i := range b.Cells {
		if b.Cells[i].ID == id {
			return &b.Cells[i]
		}
	}
	return nil
}

// Event is one entry of a bingo's history.
type Event struct {
	ID         int64
	BingoID    int64
	BingoTitle string
	CellID     *int64 // nil for events that are not about a single cell
	Kind       string
	Label      string // cell label or line name at the time of the event
	Note       string
	CreatedAt  time.Time
}

// Input describes a bingo to create or update.
type Input struct {
	Title       string
	Description string
	Size        int
	Cells       []string // labels, row by row
}

// normalize trims the input in place and checks it against the business rules.
func (in *Input) normalize() error {
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
