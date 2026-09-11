// Package slime implements a colourful agent-based slime mould simulation.
package slime

import (
	"math"
	"math/rand"
	"simulation-renderer/internal/shared"
)

const (
	columns        = 120
	rows           = 68
	cellSize       = 50.0
	agentCount     = 900
	sensorDistance = 22.0
	sensorAngle    = math.Pi / 5
	turnAngle      = 2.3
	moveSpeed      = 78.0
	diffusion      = 0.38
	evaporation    = 0.975
	nutrientLife   = 4.5
)

type agent struct {
	pos   shared.Vec2
	angle float64
	color shared.Color
}
type pigment struct{ r, g, b float64 }
type nutrient struct {
	pos  shared.Vec2
	life float64
}

// Slime is a population of agents that follows its own evaporating pigment
// field. Each colour colony leaves a different luminous trail.
type Slime struct {
	agents           []agent
	field, nextField []pigment
	nutrients        []nutrient
	rng              *rand.Rand
}

func New() *Slime             { return &Slime{} }
func (s *Slime) Name() string { return "Chromatic Slime Mould" }

func (s *Slime) Init() {
	s.rng = rand.New(rand.NewSource(71))
	s.field, s.nextField = make([]pigment, columns*rows), make([]pigment, columns*rows)
	s.agents = make([]agent, agentCount)
	palette := []shared.Color{{R: 30, G: 238, B: 255, A: 255}, {R: 177, G: 80, B: 255, A: 255}, {R: 255, G: 46, B: 172, A: 255}, {R: 255, G: 173, B: 46, A: 255}}
	for i := range s.agents {
		angle := s.rng.Float64() * 2 * math.Pi
		radius := math.Sqrt(s.rng.Float64()) * 110
		s.agents[i] = agent{pos: shared.Vec2{X: math.Cos(angle) * radius, Y: math.Sin(angle) * radius}, angle: angle + math.Pi, color: palette[i%len(palette)]}
	}
	s.nutrients = []nutrient{{pos: shared.Vec2{}, life: nutrientLife}}
}

func (s *Slime) Tick(dt float64, input shared.InputState) {
	s.HandleInput(input)
	for i := range s.agents {
		a := &s.agents[i]
		forward, left, right := s.sample(a.pos, a.angle), s.sample(a.pos, a.angle-sensorAngle), s.sample(a.pos, a.angle+sensorAngle)
		switch {
		case left > forward && left > right:
			a.angle -= turnAngle * dt
		case right > forward && right > left:
			a.angle += turnAngle * dt
		case forward < left || forward < right:
			a.angle += (s.rng.Float64()*2 - 1) * turnAngle * dt
		}
		for _, n := range s.nutrients {
			if n.life > 0 {
				dx, dy := n.pos.X-a.pos.X, n.pos.Y-a.pos.Y
				if d := math.Hypot(dx, dy); d < 240 && d > 1 {
					a.angle += angleDelta(a.angle, math.Atan2(dy, dx)) * dt * .7
				}
			}
		}
		a.pos.X += math.Cos(a.angle) * moveSpeed * dt
		a.pos.Y += math.Sin(a.angle) * moveSpeed * dt
		if !inside(a.pos) {
			a.pos.X = clamp(a.pos.X, -worldW()/2+1, worldW()/2-1)
			a.pos.Y = clamp(a.pos.Y, -worldH()/2+1, worldH()/2-1)
			a.angle += math.Pi + (s.rng.Float64()-.5)*1.2
		}
		s.deposit(a.pos, a.color, 1)
	}
	for i := range s.nutrients {
		n := &s.nutrients[i]
		n.life -= dt
		if n.life > 0 {
			s.deposit(n.pos, shared.Color{R: 255, G: 174, B: 50, A: 255}, 2.2)
		}
	}
	s.diffuse()
}

// HandleInput drops an amber food source, drawing nearby mould toward it.
func (s *Slime) HandleInput(input shared.InputState) {
	for _, click := range input.PrimaryClicks {
		if inside(click.World) {
			s.nutrients = append(s.nutrients, nutrient{pos: click.World, life: nutrientLife})
		}
	}
}

func (s *Slime) Frame() shared.Frame {
	prims := make([]shared.Primitive, 0, columns*rows/2+len(s.agents)+len(s.nutrients)+1)
	prims = append(prims, shared.Primitive{Kind: shared.DrawRectLines, W: worldW(), H: worldH(), Color: shared.Color{R: 34, G: 49, B: 76, A: 255}})
	for y := 0; y < rows; y++ {
		for x := 0; x < columns; x++ {
			p := s.field[y*columns+x]
			if math.Max(p.r, math.Max(p.g, p.b)) < 8 {
				continue
			}
			prims = append(prims, shared.Primitive{Kind: shared.DrawRect, Pos: shared.Vec2{X: (float64(x)+.5)*cellSize - worldW()/2, Y: (float64(y)+.5)*cellSize - worldH()/2}, W: cellSize + .3, H: cellSize + .3, Color: shared.Color{R: byte(clamp(p.r, 0, 255)), G: byte(clamp(p.g, 0, 255)), B: byte(clamp(p.b, 0, 255)), A: 255}})
		}
	}
	for _, n := range s.nutrients {
		if n.life > 0 {
			prims = append(prims, shared.Primitive{Kind: shared.DrawCircle, Pos: n.pos, Radius: 5 + 8*n.life/nutrientLife, Color: shared.Color{R: 255, G: 220, B: 100, A: 255}})
		}
	}
	for _, a := range s.agents {
		prims = append(prims, shared.Primitive{Kind: shared.DrawCircle, Pos: a.pos, Radius: 1.7, Color: a.color})
	}
	return shared.Frame{Primitives: prims, SimName: s.Name(), ControlHint: "LMB: place nutrient"}
}

func (s *Slime) sample(pos shared.Vec2, angle float64) float64 {
	x, y := cellAt(shared.Vec2{X: pos.X + math.Cos(angle)*sensorDistance, Y: pos.Y + math.Sin(angle)*sensorDistance})
	if x < 0 || y < 0 || x >= columns || y >= rows {
		return -1
	}
	p := s.field[y*columns+x]
	return p.r + p.g + p.b
}
func (s *Slime) deposit(pos shared.Vec2, c shared.Color, amount float64) {
	x, y := cellAt(pos)
	if x < 0 || y < 0 || x >= columns || y >= rows {
		return
	}
	p := &s.field[y*columns+x]
	p.r = math.Min(255, p.r+float64(c.R)*amount*.12)
	p.g = math.Min(255, p.g+float64(c.G)*amount*.12)
	p.b = math.Min(255, p.b+float64(c.B)*amount*.12)
}
func (s *Slime) diffuse() {
	for y := 0; y < rows; y++ {
		for x := 0; x < columns; x++ {
			var total pigment
			for oy := -1; oy <= 1; oy++ {
				for ox := -1; ox <= 1; ox++ {
					nx, ny := x+ox, y+oy
					if nx < 0 || nx >= columns || ny < 0 || ny >= rows {
						continue
					}
					q := s.field[ny*columns+nx]
					total.r += q.r
					total.g += q.g
					total.b += q.b
				}
			}
			p, n := s.field[y*columns+x], &s.nextField[y*columns+x]
			n.r = (p.r*(1-diffusion) + total.r/9*diffusion) * evaporation
			n.g = (p.g*(1-diffusion) + total.g/9*diffusion) * evaporation
			n.b = (p.b*(1-diffusion) + total.b/9*diffusion) * evaporation
		}
	}
	s.field, s.nextField = s.nextField, s.field
}
func worldW() float64           { return float64(columns) * cellSize }
func worldH() float64           { return float64(rows) * cellSize }
func inside(p shared.Vec2) bool { return math.Abs(p.X) < worldW()/2 && math.Abs(p.Y) < worldH()/2 }
func cellAt(p shared.Vec2) (int, int) {
	return int((p.X + worldW()/2) / cellSize), int((p.Y + worldH()/2) / cellSize)
}
func clamp(v, low, high float64) float64  { return math.Max(low, math.Min(high, v)) }
func angleDelta(from, to float64) float64 { return math.Atan2(math.Sin(to-from), math.Cos(to-from)) }
