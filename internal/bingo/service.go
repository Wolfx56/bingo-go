package bingo

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"
	"unicode/utf8"
)

// Service implements the app's use cases on top of a Repository.
type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		// Millisecond precision is all the history needs.
		now: func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) },
	}
}

// List returns every bingo, most recently active first.
func (s *Service) List(ctx context.Context) ([]Bingo, error) {
	return s.repo.Bingos(ctx)
}

// Get returns a bingo with its cells.
func (s *Service) Get(ctx context.Context, id int64) (*Bingo, error) {
	return s.repo.Bingo(ctx, id)
}

// Create stores a new bingo.
func (s *Service) Create(ctx context.Context, in Input) (*Bingo, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
	var out *Bingo
	err := s.repo.InTx(ctx, func(r Repository) error {
		var err error
		out, err = s.insert(ctx, r, in)
		return err
	})
	return out, err
}

// insert stores an already normalized bingo and opens its history.
func (s *Service) insert(ctx context.Context, r Repository, in Input) (*Bingo, error) {
	now := s.now()
	id, err := r.InsertBingo(ctx, in, now)
	if err != nil {
		return nil, err
	}
	if err := r.AddEvent(ctx, Event{BingoID: id, Kind: EventCreated, CreatedAt: now}); err != nil {
		return nil, err
	}
	return r.Bingo(ctx, id)
}

// Update changes a bingo's title, description and cell labels.
// The grid size and the cells' checked state are left untouched.
func (s *Service) Update(ctx context.Context, id int64, in Input) (*Bingo, error) {
	var out *Bingo
	err := s.repo.InTx(ctx, func(r Repository) error {
		b, err := r.Bingo(ctx, id)
		if err != nil {
			return err
		}
		if b.Archived() {
			return ErrArchived
		}
		in.Size = b.Size // the grid size is fixed at creation
		if err := in.normalize(); err != nil {
			return err
		}
		if err := r.UpdateBingo(ctx, id, in); err != nil {
			return err
		}
		out, err = r.Bingo(ctx, id)
		return err
	})
	return out, err
}

// Delete permanently removes a bingo, its cells and its history.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.DeleteBingo(ctx, id)
}

// SetArchived archives or unarchives a bingo. An archived bingo is read-only.
func (s *Service) SetArchived(ctx context.Context, id int64, archived bool) (*Bingo, error) {
	var out *Bingo
	err := s.repo.InTx(ctx, func(r Repository) error {
		b, err := r.Bingo(ctx, id)
		if err != nil {
			return err
		}
		if b.Archived() == archived {
			out = b
			return nil
		}
		now := s.now()
		at, kind := &now, EventArchived
		if !archived {
			at, kind = nil, EventUnarchived
		}
		if err := r.SetArchivedAt(ctx, id, at); err != nil {
			return err
		}
		if err := r.AddEvent(ctx, Event{BingoID: id, Kind: kind, CreatedAt: now}); err != nil {
			return err
		}
		out, err = r.Bingo(ctx, id)
		return err
	})
	return out, err
}

// Duplicate creates a fresh bingo with the cells of an existing one, optionally
// shuffled. An empty title falls back to the source title.
func (s *Service) Duplicate(ctx context.Context, id int64, title string, shuffle bool) (*Bingo, error) {
	var out *Bingo
	err := s.repo.InTx(ctx, func(r Repository) error {
		src, err := r.Bingo(ctx, id)
		if err != nil {
			return err
		}
		in := Input{Title: title, Description: src.Description, Size: src.Size}
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
		out, err = s.insert(ctx, r, in)
		return err
	})
	return out, err
}

// Validate checks a cell and persists it right away, with an optional note.
// Lines completed by this validation are recorded in the history.
// If the cell is already checked, only its note is updated.
func (s *Service) Validate(ctx context.Context, bingoID, cellID int64, note string) (*Bingo, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > maxNoteLen {
		return nil, invalid("La note ne doit pas dépasser %d caractères.", maxNoteLen)
	}
	var out *Bingo
	err := s.repo.InTx(ctx, func(r Repository) error {
		before, err := r.Bingo(ctx, bingoID)
		if err != nil {
			return err
		}
		if before.Archived() {
			return ErrArchived
		}
		cell := before.cell(cellID)
		if cell == nil {
			return ErrNotFound
		}
		if cell.Checked() {
			if err := r.SetCellState(ctx, cellID, cell.CheckedAt, note); err != nil {
				return err
			}
			out, err = r.Bingo(ctx, bingoID)
			return err
		}

		now := s.now()
		if err := r.SetCellState(ctx, cellID, &now, note); err != nil {
			return err
		}
		validated := Event{BingoID: bingoID, CellID: &cellID, Kind: EventValidated, Label: cell.Label, Note: note, CreatedAt: now}
		if err := r.AddEvent(ctx, validated); err != nil {
			return err
		}
		after, err := r.Bingo(ctx, bingoID)
		if err != nil {
			return err
		}
		known := make(map[string]bool)
		for _, l := range before.Lines() {
			known[l.Name] = true
		}
		for _, l := range after.Lines() {
			if known[l.Name] {
				continue
			}
			if err := r.AddEvent(ctx, Event{BingoID: bingoID, Kind: EventLine, Label: l.Name, CreatedAt: now}); err != nil {
				return err
			}
		}
		if after.Full() {
			if err := r.AddEvent(ctx, Event{BingoID: bingoID, Kind: EventFull, CreatedAt: now}); err != nil {
				return err
			}
		}
		out = after
		return nil
	})
	return out, err
}

// Unvalidate unchecks a cell and clears its note.
func (s *Service) Unvalidate(ctx context.Context, bingoID, cellID int64) (*Bingo, error) {
	var out *Bingo
	err := s.repo.InTx(ctx, func(r Repository) error {
		b, err := r.Bingo(ctx, bingoID)
		if err != nil {
			return err
		}
		if b.Archived() {
			return ErrArchived
		}
		cell := b.cell(cellID)
		if cell == nil {
			return ErrNotFound
		}
		if !cell.Checked() {
			out = b
			return nil
		}
		if err := r.SetCellState(ctx, cellID, nil, ""); err != nil {
			return err
		}
		unvalidated := Event{BingoID: bingoID, CellID: &cellID, Kind: EventUnvalidated, Label: cell.Label, CreatedAt: s.now()}
		if err := r.AddEvent(ctx, unvalidated); err != nil {
			return err
		}
		out, err = r.Bingo(ctx, bingoID)
		return err
	})
	return out, err
}

// History returns events, most recent first. A bingoID of 0 means all bingos;
// a limit outside 1..1000 means 1000.
func (s *Service) History(ctx context.Context, bingoID int64, limit int) ([]Event, error) {
	if limit <= 0 || limit > maxHistory {
		limit = maxHistory
	}
	return s.repo.Events(ctx, bingoID, limit)
}
