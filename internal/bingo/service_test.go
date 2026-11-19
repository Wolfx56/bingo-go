package bingo_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"bingo/internal/bingo"
	"bingo/internal/store"
)

// The service is tested against the real SQLite repository: the rules are only
// worth something if they hold once persisted.
func newService(t *testing.T) *bingo.Service {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return bingo.NewService(st)
}

func labels(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("Cell %d", i+1)
	}
	return out
}

func TestCreateValidation(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()
	cases := map[string]bingo.Input{
		"empty title":   {Title: " ", Size: 3, Cells: labels(9)},
		"invalid size":  {Title: "x", Size: 2, Cells: labels(4)},
		"missing cells": {Title: "x", Size: 3, Cells: labels(8)},
		"empty cell":    {Title: "x", Size: 3, Cells: append(labels(8), "  ")},
	}
	for name, in := range cases {
		var verr *bingo.ValidationError
		if _, err := svc.Create(ctx, in); !errors.As(err, &verr) {
			t.Errorf("%s: err = %v, want a ValidationError", name, err)
		}
	}
	if list, _ := svc.List(ctx); len(list) != 0 {
		t.Errorf("rejected inputs left %d bingos behind", len(list))
	}
}

func TestValidateFlow(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()
	b, err := svc.Create(ctx, bingo.Input{Title: "Test", Size: 3, Cells: labels(9)})
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Cells) != 9 || b.CheckedCount() != 0 {
		t.Fatalf("unexpected new bingo: %+v", b)
	}

	// Check the first row: the third cell completes a line.
	for i := range 3 {
		b, err = svc.Validate(ctx, b.ID, b.Cells[i].ID, "note")
		if err != nil {
			t.Fatal(err)
		}
	}
	if lines := b.Lines(); b.CheckedCount() != 3 || len(lines) != 1 || lines[0].Name != "Ligne 1" {
		t.Fatalf("after 3 validations: checked=%d lines=%v", b.CheckedCount(), lines)
	}
	if !b.Cells[0].Checked() || b.Cells[0].Note != "note" {
		t.Errorf("cell 0 not persisted: %+v", b.Cells[0])
	}

	// Read back from the database: the state really was saved.
	reread, err := svc.Get(ctx, b.ID)
	if err != nil || reread.CheckedCount() != 3 {
		t.Fatalf("reread: err=%v checked=%d", err, reread.CheckedCount())
	}

	// Validating an already checked cell only updates its note: no new event.
	b, err = svc.Validate(ctx, b.ID, b.Cells[0].ID, "edited")
	if err != nil || b.Cells[0].Note != "edited" {
		t.Fatalf("note update: err=%v note=%q", err, b.Cells[0].Note)
	}

	events, err := svc.History(ctx, b.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	kinds := make([]string, len(events))
	for i, e := range events {
		kinds[i] = e.Kind
	}
	want := []string{bingo.EventLine, bingo.EventValidated, bingo.EventValidated, bingo.EventValidated, bingo.EventCreated}
	if fmt.Sprint(kinds) != fmt.Sprint(want) {
		t.Errorf("history = %v, want %v", kinds, want)
	}

	// Unvalidating a cell breaks the line.
	b, err = svc.Unvalidate(ctx, b.ID, b.Cells[1].ID)
	if err != nil || b.CheckedCount() != 2 || len(b.Lines()) != 0 || b.Cells[1].Note != "" {
		t.Fatalf("after unvalidate: err=%v checked=%d lines=%v", err, b.CheckedCount(), b.Lines())
	}

	// An archived bingo is read-only.
	if _, err := svc.SetArchived(ctx, b.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Validate(ctx, b.ID, b.Cells[1].ID, ""); !errors.Is(err, bingo.ErrArchived) {
		t.Errorf("validating an archived bingo: err = %v, want ErrArchived", err)
	}
	if _, err := svc.Update(ctx, b.ID, bingo.Input{Title: "x", Cells: labels(9)}); !errors.Is(err, bingo.ErrArchived) {
		t.Errorf("updating an archived bingo: err = %v, want ErrArchived", err)
	}

	if _, err := svc.Validate(ctx, 999, 1, ""); !errors.Is(err, bingo.ErrNotFound) {
		t.Errorf("unknown bingo: err = %v, want ErrNotFound", err)
	}
}

func TestFullCard(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()
	b, _ := svc.Create(ctx, bingo.Input{Title: "Full", Size: 3, Cells: labels(9)})
	for _, c := range b.Cells {
		var err error
		if b, err = svc.Validate(ctx, b.ID, c.ID, ""); err != nil {
			t.Fatal(err)
		}
	}
	if !b.Full() || len(b.Lines()) != 8 {
		t.Fatalf("full=%v lines=%d, want true and 8", b.Full(), len(b.Lines()))
	}
	events, _ := svc.History(ctx, b.ID, 0)
	counts := map[string]int{}
	for _, e := range events {
		counts[e.Kind]++
	}
	if counts[bingo.EventFull] != 1 || counts[bingo.EventLine] != 8 || counts[bingo.EventValidated] != 9 {
		t.Errorf("event counts = %v, want 1 full, 8 line, 9 validated", counts)
	}
}

func TestUpdateDuplicateDelete(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()
	b, _ := svc.Create(ctx, bingo.Input{Title: "Original", Size: 3, Cells: labels(9)})
	b, _ = svc.Validate(ctx, b.ID, b.Cells[4].ID, "")

	renamed := labels(9)
	renamed[4] = "Center"
	b, err := svc.Update(ctx, b.ID, bingo.Input{Title: "Renamed", Cells: renamed})
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "Renamed" || b.Cells[4].Label != "Center" || !b.Cells[4].Checked() {
		t.Errorf("unexpected update: title=%q cell=%+v", b.Title, b.Cells[4])
	}

	copy, err := svc.Duplicate(ctx, b.ID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if copy.ID == b.ID || copy.Title != "Renamed" || copy.CheckedCount() != 0 || len(copy.Cells) != 9 {
		t.Errorf("unexpected copy: %+v", copy)
	}

	list, _ := svc.List(ctx)
	if len(list) != 2 || len(list[0].Cells) != 9 {
		t.Fatalf("list: got %d bingos, want 2 with their cells", len(list))
	}

	if err := svc.Delete(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, b.ID); !errors.Is(err, bingo.ErrNotFound) {
		t.Errorf("after delete: err = %v, want ErrNotFound", err)
	}
	if events, _ := svc.History(ctx, b.ID, 0); len(events) != 0 {
		t.Errorf("history not deleted: %d events left", len(events))
	}
	if err := svc.Delete(ctx, b.ID); !errors.Is(err, bingo.ErrNotFound) {
		t.Errorf("deleting twice: err = %v, want ErrNotFound", err)
	}
}
