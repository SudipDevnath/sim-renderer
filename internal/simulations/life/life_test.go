package life

import (
	"simulation-renderer/internal/shared"
	"testing"
)

func TestStillLifeRemainsUnchanged(t *testing.T) {
	var l Life
	for _, cell := range [][2]int{{10, 10}, {10, 11}, {11, 10}, {11, 11}} {
		l.setAlive(cell[0], cell[1])
	}

	l.advance()
	for _, cell := range [][2]int{{10, 10}, {10, 11}, {11, 10}, {11, 11}} {
		if !l.cells[cell[0]][cell[1]] {
			t.Fatalf("cell %v should survive", cell)
		}
	}
}

func TestWrappingEdgeCanCreateCell(t *testing.T) {
	var l Life
	l.setAlive(rows-1, columns-1)
	l.setAlive(rows-1, 0)
	l.setAlive(0, columns-1)

	l.advance()
	if !l.cells[0][0] {
		t.Fatal("corner cell should be born from wrapped neighbours")
	}
}

func TestHandleInputTogglesCell(t *testing.T) {
	var l Life
	row, col := 12, 30
	boardW, boardH := float64(columns)*cellSize, float64(rows)*cellSize
	world := shared.Vec2{
		X: (float64(col)+0.5)*cellSize - boardW/2,
		Y: (float64(row)+0.5)*cellSize - boardH/2,
	}

	l.HandleInput(shared.InputState{PrimaryClicks: []shared.PrimaryClick{{World: world}}})
	if !l.cells[row][col] {
		t.Fatal("click should create a cell")
	}
	l.HandleInput(shared.InputState{PrimaryClicks: []shared.PrimaryClick{{World: world}}})
	if l.cells[row][col] {
		t.Fatal("second click should remove a cell")
	}
}
