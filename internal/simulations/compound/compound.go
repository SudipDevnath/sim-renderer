// Package compound implements a compound cellular automaton simulation.
package compound

import (
	"math"
	"math/rand"
	"simulation-renderer/internal/shared"
)

const (
	columns       = 1000
	rows          = 600
	cellSize      = 10.0
	numCompounds  = 4 // number of chemical states
	diffusionRate = 0.2
	decayRate     = 0.98 - 0.9
	reactionRate  = 0.12
)

type cell struct {
	compounds [numCompounds]float64
}

type CompoundCA struct {
	grid, nextGrid []cell
	rng            *rand.Rand
}

func New() *CompoundCA             { return &CompoundCA{} }
func (c *CompoundCA) Name() string { return "Compound Cellular Automata" }

func (c *CompoundCA) Init() {
	c.rng = rand.New(rand.NewSource(99))
	c.grid, c.nextGrid = make([]cell, columns*rows), make([]cell, columns*rows)

	// Seed random compounds
	for i := range c.grid {
		for k := 0; k < numCompounds; k++ {
			if c.rng.Float64() < 0.1 {
				c.grid[i].compounds[k] = c.rng.Float64() * 255
			}
		}
	}
}

func (c *CompoundCA) Tick(dt float64, input shared.InputState) {
	c.HandleInput(input)
	if dt < 0 {
		dt = 0
	}

	for y := 0; y < rows; y++ {
		for x := 0; x < columns; x++ {
			idx := y*columns + x
			cur := c.grid[idx]

			// Diffusion with neighbors
			var next cell
			for k := 0; k < numCompounds; k++ {
				sum := cur.compounds[k]
				count := 1
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := (x+dx+columns)%columns, (y+dy+rows)%rows
						n := c.grid[ny*columns+nx]
						sum += n.compounds[k]
						count++
					}
				}
				avg := sum / float64(count)
				// Apply diffusion and decay
				next.compounds[k] = cur.compounds[k]*(1-diffusionRate) + avg*diffusionRate
				next.compounds[k] *= decayRate
			}

			// Example reaction rule: compound 0 + compound 1 → compound 2
			reaction := math.Min(
				next.compounds[0],
				math.Min(next.compounds[1], next.compounds[0]*next.compounds[1]*reactionRate*dt),
			)
			next.compounds[0] -= reaction
			next.compounds[1] -= reaction
			next.compounds[2] += reaction

			c.nextGrid[idx] = next
		}
	}
	c.grid, c.nextGrid = c.nextGrid, c.grid
}

func (c *CompoundCA) HandleInput(input shared.InputState) {
	for _, click := range input.PrimaryClicks {
		x := int((click.World.X + float64(columns)*cellSize/2) / cellSize)
		y := int((click.World.Y + float64(rows)*cellSize/2) / cellSize)
		if x >= 0 && x < columns && y >= 0 && y < rows {
			idx := y*columns + x
			// Inject compound 3 at click
			c.grid[idx].compounds[3] = 255
		}
	}
}

func (c *CompoundCA) Frame() shared.Frame {
	prims := make([]shared.Primitive, 0, columns*rows)
	for y := 0; y < rows; y++ {
		for x := 0; x < columns; x++ {
			idx := y*columns + x
			cell := c.grid[idx]
			// Compound 3 is the reagent added by clicks. Blend it into the
			// display so injections remain visible while they diffuse.
			r := colorByte(cell.compounds[0] + cell.compounds[3]*0.45)
			g := colorByte(cell.compounds[1] + cell.compounds[3]*0.15)
			b := colorByte(cell.compounds[2] + cell.compounds[3])
			a := colorByte(math.Max(cell.compounds[0], math.Max(cell.compounds[1], math.Max(cell.compounds[2], cell.compounds[3]))))
			if r < 10 && g < 10 && b < 10 {
				continue
			}
			prims = append(prims, shared.Primitive{
				Kind: shared.DrawRect,
				Pos: shared.Vec2{
					X: (float64(x)+0.5)*cellSize - float64(columns)*cellSize/2,
					Y: (float64(y)+0.5)*cellSize - float64(rows)*cellSize/2,
				},
				W: cellSize - 1, H: cellSize - 1,
				Color: shared.Color{R: r, G: g, B: b, A: a},
			})
		}
	}
	return shared.Frame{Primitives: prims, SimName: c.Name(), ControlHint: "LMB: inject compound"}
}

// colorByte clamps a concentration before converting it to a display colour.
func colorByte(v float64) uint8 {
	if v <= 0 || math.IsNaN(v) {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
}
