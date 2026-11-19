package store

import "fmt"

// Line est une ligne gagnante (rangée, colonne ou diagonale) entièrement validée.
type Line struct {
	Name  string `json:"name"`
	Cells []int  `json:"cells"` // positions des cases, de 0 à size*size-1
}

// CompletedLines renvoie les lignes entièrement validées d'une grille size×size.
// checked est indexé par position (rangée par rangée).
func CompletedLines(size int, checked []bool) []Line {
	lines := make([]Line, 0)
	if size <= 0 || len(checked) != size*size {
		return lines
	}
	add := func(name string, pos func(i int) int) {
		cells := make([]int, size)
		for i := range size {
			p := pos(i)
			if !checked[p] {
				return
			}
			cells[i] = p
		}
		lines = append(lines, Line{Name: name, Cells: cells})
	}
	for r := range size {
		add(fmt.Sprintf("Ligne %d", r+1), func(i int) int { return r*size + i })
	}
	for c := range size {
		add(fmt.Sprintf("Colonne %d", c+1), func(i int) int { return i*size + c })
	}
	add("Diagonale ↘", func(i int) int { return i*size + i })
	add("Diagonale ↙", func(i int) int { return i*size + (size - 1 - i) })
	return lines
}
