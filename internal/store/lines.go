package store

import "fmt"

// Line is a winning line (row, column or diagonal) whose cells are all checked.
type Line struct {
	Name  string `json:"name"`  // user-facing label, in French
	Cells []int  `json:"cells"` // cell positions, from 0 to size*size-1
}

// CompletedLines returns the fully checked lines of a size×size grid.
// checked is indexed by position (row by row).
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
