package collision

import (
	"math"
	"math/rand"
	"simulation-renderer/internal/shared"
)

const (
	numBodies       = 60
	arenaW          = 580.0
	arenaH          = 320.0
	minMass         = 5.0
	maxMass         = 15.0
	massToRadius    = 3.5
	restitution     = 0.95
	wallRestitution = 0.97
)

// body is the collision simulation's private particle representation.
type body struct {
	pos, vel shared.Vec2
	mass     float64
	radius   float64
}

// Collision implements shared.Simulation with an O(n²) elastic collision
// detector inside a bounded rectangular arena.
type Collision struct {
	bodies []body
}

// New returns a ready-to-use Collision simulation.
func New() *Collision { return &Collision{} }

// Name satisfies shared.Simulation.
func (c *Collision) Name() string { return "N-Body Collision" }

// Init scatters non-overlapping bodies inside the arena with random velocities.
func (c *Collision) Init() {
	rng := rand.New(rand.NewSource(7))
	c.bodies = make([]body, 0, numBodies)

	maxAttempts := 10000
	attempts := 0

	for len(c.bodies) < numBodies && attempts < maxAttempts {
		attempts++

		mass := minMass + rng.Float64()*(maxMass-minMass)
		radius := math.Sqrt(mass) * massToRadius

		px := (rng.Float64()*2 - 1) * (arenaW - radius)
		py := (rng.Float64()*2 - 1) * (arenaH - radius)

		// Reject if overlapping any placed body.
		overlap := false
		for _, b := range c.bodies {
			dx := px - b.pos.X
			dy := py - b.pos.Y
			if math.Sqrt(dx*dx+dy*dy) < radius+b.radius {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}

		speed := 80.0 + rng.Float64()*120.0
		angle := rng.Float64() * 2 * math.Pi

		c.bodies = append(c.bodies, body{
			pos:    shared.Vec2{X: px, Y: py},
			vel:    shared.Vec2{X: math.Cos(angle) * speed, Y: math.Sin(angle) * speed},
			mass:   mass,
			radius: radius,
		})
	}
}

// Tick advances the collision simulation by dt seconds.
// Three passes: integrate → wall bounce → pairwise collision resolution.
func (c *Collision) Tick(dt float64, _ shared.InputState) {
	n := len(c.bodies)

	// ── 1. Integrate ──────────────────────────────────────────────────────────
	for i := range c.bodies {
		b := &c.bodies[i]
		b.pos.X += b.vel.X * dt
		b.pos.Y += b.vel.Y * dt
	}

	// ── 2. Wall bounce ────────────────────────────────────────────────────────
	for i := range c.bodies {
		b := &c.bodies[i]

		if b.pos.X-b.radius < -arenaW {
			b.pos.X = -arenaW + b.radius
			b.vel.X = math.Abs(b.vel.X) * wallRestitution
		} else if b.pos.X+b.radius > arenaW {
			b.pos.X = arenaW - b.radius
			b.vel.X = -math.Abs(b.vel.X) * wallRestitution
		}
		if b.pos.Y-b.radius < -arenaH {
			b.pos.Y = -arenaH + b.radius
			b.vel.Y = math.Abs(b.vel.Y) * wallRestitution
		} else if b.pos.Y+b.radius > arenaH {
			b.pos.Y = arenaH - b.radius
			b.vel.Y = -math.Abs(b.vel.Y) * wallRestitution
		}
	}

	// ── 3. Pairwise collision detection & resolution (O(n²)) ─────────────────
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			bi := &c.bodies[i]
			bj := &c.bodies[j]

			dx := bj.pos.X - bi.pos.X
			dy := bj.pos.Y - bi.pos.Y
			distSq := dx*dx + dy*dy
			minDist := bi.radius + bj.radius

			if distSq >= minDist*minDist {
				continue
			}

			dist := math.Sqrt(distSq)
			if dist < 1e-9 {
				bi.pos.X -= 0.5
				bj.pos.X += 0.5
				continue
			}

			nx := dx / dist
			ny := dy / dist

			// Positional correction.
			overlap := minDist - dist
			totalMass := bi.mass + bj.mass
			bi.pos.X -= nx * overlap * (bj.mass / totalMass)
			bi.pos.Y -= ny * overlap * (bj.mass / totalMass)
			bj.pos.X += nx * overlap * (bi.mass / totalMass)
			bj.pos.Y += ny * overlap * (bi.mass / totalMass)

			// Elastic impulse.
			rvx := bi.vel.X - bj.vel.X
			rvy := bi.vel.Y - bj.vel.Y
			relVelN := rvx*nx + rvy*ny

			if relVelN <= 0 {
				continue
			}

			impulse := (1 + restitution) * relVelN / totalMass
			bi.vel.X -= impulse * bj.mass * nx
			bi.vel.Y -= impulse * bj.mass * ny
			bj.vel.X += impulse * bi.mass * nx
			bj.vel.Y += impulse * bi.mass * ny
		}
	}
}

// Frame converts the current body state into draw primitives.
// It also emits the arena boundary as a DrawRectLines primitive.
func (c *Collision) Frame() shared.Frame {
	prims := make([]shared.Primitive, 0, len(c.bodies)+1)

	// Arena boundary.
	prims = append(prims, shared.Primitive{
		Kind:  shared.DrawRectLines,
		Pos:   shared.Vec2{},
		W:     arenaW * 2,
		H:     arenaH * 2,
		Color: shared.Gray,
	})

	// Bodies.
	for _, b := range c.bodies {
		prims = append(prims, shared.Primitive{
			Kind:   shared.DrawCircle,
			Pos:    b.pos,
			Radius: b.radius,
			Color:  radiusColor(b.radius),
		})
	}

	return shared.Frame{
		Primitives: prims,
		SimName:    c.Name(),
	}
}

// radiusColor maps a body radius to a color for visual variety.
func radiusColor(radius float64) shared.Color {
	// radius range roughly [sqrt(5)*3.5, sqrt(15)*3.5] ≈ [7.8, 13.5]
	lo, hi := math.Sqrt(minMass)*massToRadius, math.Sqrt(maxMass)*massToRadius
	t := float32((radius - lo) / (hi - lo))
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return shared.Color{
		R: uint8(50 + t*200),
		G: uint8(200 - t*100),
		B: uint8(255 - t*180),
		A: 255,
	}
}
