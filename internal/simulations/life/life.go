// Package life implements Conway's Game of Life as a shared.Simulation.
package life

import "simulation-renderer/internal/shared"

const (
	columns       = 80
	rows          = 44
	cellSize      = 14.0
	generationGap = 0.12
)

// Life is a finite, wrapping Conway's Game of Life board.
type Life struct {
	cells      [rows][columns]bool
	age        [rows][columns]uint8
	elapsed    float64
	generation uint64
	dirty      bool
	cached     []shared.Primitive
}

// New returns a ready-to-use Game of Life simulation.
func New() *Life { return &Life{} }

// Name satisfies shared.Simulation.
func (l *Life) Name() string { return "Conway's Game of Life" }

// Init clears the board and seeds a Gosper glider gun plus a small glider.
func (l *Life) Init() {
	*l = Life{}

	// The gun creates a continuous stream of gliders, which makes a good
	// starting pattern without relying on random state.
	gun := [][2]int{
		{5, 1}, {5, 2}, {6, 1}, {6, 2},
		{5, 11}, {6, 11}, {7, 11}, {4, 12}, {8, 12}, {3, 13}, {9, 13},
		{3, 14}, {9, 14}, {6, 15}, {4, 16}, {8, 16}, {5, 17}, {6, 17}, {7, 17}, {6, 18},
		{3, 21}, {4, 21}, {5, 21}, {3, 22}, {4, 22}, {5, 22}, {2, 23}, {6, 23},
		{1, 25}, {2, 25}, {6, 25}, {7, 25}, {3, 35}, {4, 35}, {3, 36}, {4, 36},
	}
	for _, p := range gun {
		l.setAlive(p[0]+16, p[1]+20)
	}

	// A glider gives the scene activity while the gun begins its cycle.
	for _, p := range [][2]int{{0, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}} {
		l.setAlive(p[0]+15, p[1]+3)
	}
	l.dirty = true
}

func (l *Life) setAlive(row, col int) {
	if row >= 0 && row < rows && col >= 0 && col < columns {
		l.cells[row][col] = true
		l.age[row][col] = 1
	}
}

// Tick advances at a deliberate visual cadence; dt still comes from the
// generic scene runner, so Life remains independent of the render rate.
func (l *Life) Tick(dt float64, input shared.InputState) {
	l.HandleInput(input)
	l.elapsed += dt
	for l.elapsed >= generationGap {
		l.elapsed -= generationGap
		l.advance()
	}
}

// HandleInput applies every queued click without advancing a generation.
func (l *Life) HandleInput(input shared.InputState) {
	for _, click := range input.PrimaryClicks {
		l.toggleCell(click.World)
	}
}

func (l *Life) toggleCell(world shared.Vec2) {
	boardW, boardH := float64(columns)*cellSize, float64(rows)*cellSize
	x, y := world.X+boardW/2, world.Y+boardH/2
	if x < 0 || x >= boardW || y < 0 || y >= boardH {
		return
	}

	col, row := int(x/cellSize), int(y/cellSize)
	l.cells[row][col] = !l.cells[row][col]
	if l.cells[row][col] {
		l.age[row][col] = 1
	} else {
		l.age[row][col] = 0
	}
	l.dirty = true
}

func (l *Life) advance() {
	var next [rows][columns]bool
	var nextAge [rows][columns]uint8
	for row := 0; row < rows; row++ {
		for col := 0; col < columns; col++ {
			neighbors := l.neighbors(row, col)
			next[row][col] = neighbors == 3 || (l.cells[row][col] && neighbors == 2)
			if next[row][col] {
				if l.cells[row][col] && l.age[row][col] < 255 {
					nextAge[row][col] = l.age[row][col] + 1
				} else {
					nextAge[row][col] = 1
				}
			}
		}
	}
	l.cells, l.age = next, nextAge
	l.generation++
	l.dirty = true
}

func (l *Life) neighbors(row, col int) int {
	count := 0
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			if dr == 0 && dc == 0 {
				continue
			}
			r := (row + dr + rows) % rows
			c := (col + dc + columns) % columns
			if l.cells[r][c] {
				count++
			}
		}
	}
	return count
}

// Frame draws the board and its living cells. Older cells become brighter,
// making stable structures and newly born cells easy to distinguish.
func (l *Life) Frame() shared.Frame {
	if l.dirty || l.cached == nil {
		l.cached = l.buildPrimitives()
		l.dirty = false
	}
	return shared.Frame{
		Primitives:  l.cached,
		SimName:     l.Name() + "  ·  Generation " + itoa(l.generation),
		ControlHint: "LMB: toggle cell",
	}
}

func (l *Life) buildPrimitives() []shared.Primitive {
	boardW, boardH := float64(columns)*cellSize, float64(rows)*cellSize
	prims := make([]shared.Primitive, 0, columns*rows/3+1)
	prims = append(prims, shared.Primitive{
		Kind: shared.DrawRectLines, Pos: shared.Vec2{}, W: boardW, H: boardH,
		Color: shared.Gray,
	})

	for row := 0; row < rows; row++ {
		for col := 0; col < columns; col++ {
			if !l.cells[row][col] {
				continue
			}
			age := l.age[row][col]
			prims = append(prims, shared.Primitive{
				Kind: shared.DrawRect,
				Pos: shared.Vec2{
					X: (float64(col)+0.5)*cellSize - boardW/2,
					Y: (float64(row)+0.5)*cellSize - boardH/2,
				},
				W: cellSize - 1, H: cellSize - 1,
				Color: cellColor(age),
			})
		}
	}

	return prims
}

func cellColor(age uint8) shared.Color {
	brightness := uint8(90)
	if age > 12 {
		brightness = 190
	} else {
		brightness += age * 8
	}
	return shared.Color{R: 45, G: brightness, B: 140 + brightness/2, A: 255}
}

// itoa avoids pulling formatting into the hot frame-building path.
func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte(n%10) + '0'
		n /= 10
	}
	return string(digits[i:])
}
