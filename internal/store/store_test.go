package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func labels(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("Case %d", i+1)
	}
	return out
}

func TestCompletedLines(t *testing.T) {
	checked := []bool{
		true, true, true,
		false, true, false,
		false, true, true,
	}
	got := CompletedLines(3, checked)
	want := []string{"Ligne 1", "Colonne 2", "Diagonale ↘"}
	if len(got) != len(want) {
		t.Fatalf("lignes = %v, attendu %v", got, want)
	}
	for i, l := range got {
		if l.Name != want[i] {
			t.Errorf("ligne %d = %q, attendu %q", i, l.Name, want[i])
		}
	}
	if n := len(CompletedLines(3, make([]bool, 9))); n != 0 {
		t.Errorf("grille vide : %d lignes", n)
	}
}

func TestCreateValidation(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	cases := map[string]BingoInput{
		"titre vide":       {Title: " ", Size: 3, Cells: labels(9)},
		"taille invalide":  {Title: "x", Size: 2, Cells: labels(4)},
		"cases manquantes": {Title: "x", Size: 3, Cells: labels(8)},
		"case vide":        {Title: "x", Size: 3, Cells: append(labels(8), "  ")},
	}
	for name, in := range cases {
		var verr *ValidationError
		if _, err := s.Create(ctx, in); !errors.As(err, &verr) {
			t.Errorf("%s : erreur = %v, attendu ValidationError", name, err)
		}
	}
}

func TestValidateFlow(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	b, err := s.Create(ctx, BingoInput{Title: "Test", Size: 3, Cells: labels(9)})
	if err != nil {
		t.Fatal(err)
	}
	if b.Total != 9 || b.Checked != 0 || len(b.Cells) != 9 {
		t.Fatalf("bingo créé inattendu : %+v", b)
	}

	// Valide la première rangée : la troisième case complète une ligne.
	for i := range 3 {
		b, err = s.Validate(ctx, b.ID, b.Cells[i].ID, "note")
		if err != nil {
			t.Fatal(err)
		}
	}
	if b.Checked != 3 || len(b.Lines) != 1 || b.Lines[0].Name != "Ligne 1" {
		t.Fatalf("après 3 validations : checked=%d lines=%v", b.Checked, b.Lines)
	}
	if b.Cells[0].CheckedAt == nil || b.Cells[0].Note != "note" {
		t.Errorf("case 0 non persistée : %+v", b.Cells[0])
	}

	// Relecture depuis la base : l'état est bien sauvegardé.
	reread, err := s.Get(ctx, b.ID)
	if err != nil || reread.Checked != 3 {
		t.Fatalf("relecture : %v, checked=%d", err, reread.Checked)
	}

	events, err := s.History(ctx, b.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	kinds := make([]string, len(events))
	for i, e := range events {
		kinds[i] = e.Kind
	}
	want := []string{EventLine, EventValidated, EventValidated, EventValidated, EventCreated}
	if fmt.Sprint(kinds) != fmt.Sprint(want) {
		t.Errorf("historique = %v, attendu %v", kinds, want)
	}

	// Annulation : la ligne disparaît.
	b, err = s.Unvalidate(ctx, b.ID, b.Cells[1].ID)
	if err != nil || b.Checked != 2 || len(b.Lines) != 0 || b.Cells[1].Note != "" {
		t.Fatalf("après annulation : err=%v checked=%d lines=%v", err, b.Checked, b.Lines)
	}

	// Un bingo archivé est en lecture seule.
	if _, err := s.SetArchived(ctx, b.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Validate(ctx, b.ID, b.Cells[1].ID, ""); !errors.Is(err, ErrArchived) {
		t.Errorf("validation d'un bingo archivé : %v", err)
	}

	if _, err := s.Validate(ctx, 999, 1, ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("bingo inexistant : %v", err)
	}
}

func TestUpdateDuplicateDelete(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	b, _ := s.Create(ctx, BingoInput{Title: "Original", Size: 3, Cells: labels(9)})
	b, _ = s.Validate(ctx, b.ID, b.Cells[4].ID, "")

	renamed := labels(9)
	renamed[4] = "Centre"
	b, err := s.Update(ctx, b.ID, BingoInput{Title: "Renommé", Cells: renamed})
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "Renommé" || b.Cells[4].Label != "Centre" || b.Cells[4].CheckedAt == nil {
		t.Errorf("mise à jour inattendue : %+v", b.Cells[4])
	}

	copy, err := s.Duplicate(ctx, b.ID, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if copy.ID == b.ID || copy.Title != "Renommé" || copy.Checked != 0 || copy.Total != 9 {
		t.Errorf("copie inattendue : %+v", copy)
	}

	list, _ := s.List(ctx)
	if len(list) != 2 {
		t.Fatalf("liste : %d bingos", len(list))
	}

	if err := s.Delete(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, b.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("après suppression : %v", err)
	}
	if events, _ := s.History(ctx, b.ID, 0); len(events) != 0 {
		t.Errorf("historique non supprimé : %d événements", len(events))
	}
}
