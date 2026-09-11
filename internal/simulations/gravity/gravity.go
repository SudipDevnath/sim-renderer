package gravity

import (
	"math"
	"math/rand"
	"simulation-renderer/internal/shared"
)

const (
	numBodies   = 128
	G           = 1000.0
	softening   = 50.0
	spawnRadius = 400.0
	baseRadius  = 3.0
)

// body is the gravity simulation's private particle representation.
// Nothing outside this package knows this type exists.
type body struct {
	pos, vel, acc shared.Vec2
	mass          float64
}

// Gravity implements shared.Simulation with an O(n²) Newtonian gravity
// solver and Euler integration.
type Gravity struct {
	bodies []body
}

// New returns a ready-to-use Gravity simulation.
func New() *Gravity { return &Gravity{} }

// Name satisfies shared.Simulation.
func (g *Gravity) Name() string { return "N-Body Gravity" }

// Init sets up the starting body configuration.
func (g *Gravity) Init() {
	rng := rand.New(rand.NewSource(42))
	g.bodies = make([]body, numBodies)

	for i := range g.bodies {
		angle := rng.Float64() * 2 * math.Pi
		r := rng.Float64() * spawnRadius

		px := math.Cos(angle) * r
		py := math.Sin(angle) * r

		speed := math.Sqrt(G*numBodies/spawnRadius) * (0.5 + rng.Float64()*0.5)
		vx := -math.Sin(angle) * speed
		vy := math.Cos(angle) * speed

		g.bodies[i] = body{
			pos:  shared.Vec2{X: px, Y: py},
			vel:  shared.Vec2{X: vx, Y: vy},
			mass: 1.0 + rng.Float64()*4.0,
		}
	}
}

// Tick advances the gravity simulation by dt seconds.
func (g *Gravity) Tick(dt float64, _ shared.InputState) {
	n := len(g.bodies)

	// Reset accelerations.
	for i := range g.bodies {
		g.bodies[i].acc = shared.Vec2{}
	}

	// O(n²) pairwise gravity accumulation.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			bi := &g.bodies[i]
			bj := &g.bodies[j]

			dx := bj.pos.X - bi.pos.X
			dy := bj.pos.Y - bi.pos.Y

			distSq := dx*dx + dy*dy + softening*softening
			dist := math.Sqrt(distSq)
			invDist3 := 1.0 / (distSq * dist)

			fi := G * bj.mass * invDist3
			fj := G * bi.mass * invDist3

			bi.acc.X += fi * dx
			bi.acc.Y += fi * dy
			bj.acc.X -= fj * dx
			bj.acc.Y -= fj * dy
		}
	}

	// Euler integration.
	for i := range g.bodies {
		b := &g.bodies[i]
		b.vel.X += b.acc.X * dt
		b.vel.Y += b.acc.Y * dt
		b.pos.X += b.vel.X * dt
		b.pos.Y += b.vel.Y * dt
	}
}

// Frame converts the current body state into draw primitives.
func (g *Gravity) Frame() shared.Frame {
	prims := make([]shared.Primitive, len(g.bodies))
	for i, b := range g.bodies {
		prims[i] = shared.Primitive{
			Kind:   shared.DrawCircle,
			Pos:    b.pos,
			Radius: baseRadius,
			Color:  massColor(b.mass, 1.0, 5.0),
		}
	}
	return shared.Frame{
		Primitives: prims,
		SimName:    g.Name(),
	}
}

// massColor maps a mass value in [lo, hi] to a colour gradient
// blue/white → orange/red.
func massColor(mass, lo, hi float64) shared.Color {
	t := float32((mass - lo) / (hi - lo))
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return shared.Color{
		R: uint8(100 + t*155),
		G: uint8(180 - t*130),
		B: uint8(255 - t*200),
		A: 255,
	}
}
