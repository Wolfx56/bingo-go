package bingo

import "testing"

func TestCompletedLines(t *testing.T) {
	checked := []bool{
		true, true, true,
		false, true, false,
		false, true, true,
	}
	got := CompletedLines(3, checked)
	want := []string{"Ligne 1", "Colonne 2", "Diagonale ↘"}
	if len(got) != len(want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}
	for i, l := range got {
		if l.Name != want[i] {
			t.Errorf("line %d = %q, want %q", i, l.Name, want[i])
		}
	}
	if n := len(CompletedLines(3, make([]bool, 9))); n != 0 {
		t.Errorf("empty grid: got %d lines, want 0", n)
	}
	if n := len(CompletedLines(3, make([]bool, 4))); n != 0 {
		t.Errorf("pattern of the wrong length: got %d lines, want 0", n)
	}
}
