package api

import (
	"time"

	"bingo/internal/bingo"
)

// The types below define the JSON contract of the API. They are deliberately
// separate from the domain model, so that either can change without the other.

type bingoRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Size        int      `json:"size"`
	Cells       []string `json:"cells"` // labels, row by row
}

func (req bingoRequest) input() bingo.Input {
	return bingo.Input{Title: req.Title, Description: req.Description, Size: req.Size, Cells: req.Cells}
}

type duplicateRequest struct {
	Title   string `json:"title"`
	Shuffle bool   `json:"shuffle"`
}

type validationRequest struct {
	Note string `json:"note"`
}

type errorJSON struct {
	Error string `json:"error"`
}

type cellJSON struct {
	ID        int64      `json:"id"`
	Position  int        `json:"position"`
	Label     string     `json:"label"`
	CheckedAt *time.Time `json:"checkedAt"`
	Note      string     `json:"note"`
}

type lineJSON struct {
	Name  string `json:"name"`
	Cells []int  `json:"cells"`
}

type bingoJSON struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Size        int        `json:"size"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ArchivedAt  *time.Time `json:"archivedAt"`
	Checked     int        `json:"checked"`
	Total       int        `json:"total"`
	Pattern     []bool     `json:"pattern"`
	Lines       []lineJSON `json:"lines"`
	Cells       []cellJSON `json:"cells,omitempty"` // only in detail responses
}

type eventJSON struct {
	ID         int64     `json:"id"`
	BingoID    int64     `json:"bingoId"`
	BingoTitle string    `json:"bingoTitle"`
	CellID     *int64    `json:"cellId"`
	Kind       string    `json:"kind"`
	Label      string    `json:"label"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"createdAt"`
}

// toBingoJSON renders a bingo. List responses pass withCells=false: the
// pattern is enough to draw a preview.
func toBingoJSON(b *bingo.Bingo, withCells bool) bingoJSON {
	out := bingoJSON{
		ID:          b.ID,
		Title:       b.Title,
		Description: b.Description,
		Size:        b.Size,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		ArchivedAt:  b.ArchivedAt,
		Checked:     b.CheckedCount(),
		Total:       len(b.Cells),
		Pattern:     b.Pattern(),
		Lines:       make([]lineJSON, 0),
	}
	for _, l := range b.Lines() {
		out.Lines = append(out.Lines, lineJSON{Name: l.Name, Cells: l.Cells})
	}
	if withCells {
		out.Cells = make([]cellJSON, len(b.Cells))
		for i, c := range b.Cells {
			out.Cells[i] = cellJSON{ID: c.ID, Position: c.Position, Label: c.Label, CheckedAt: c.CheckedAt, Note: c.Note}
		}
	}
	return out
}

func toEventsJSON(events []bingo.Event) []eventJSON {
	out := make([]eventJSON, len(events))
	for i, e := range events {
		out[i] = eventJSON{
			ID:         e.ID,
			BingoID:    e.BingoID,
			BingoTitle: e.BingoTitle,
			CellID:     e.CellID,
			Kind:       e.Kind,
			Label:      e.Label,
			Note:       e.Note,
			CreatedAt:  e.CreatedAt,
		}
	}
	return out
}
