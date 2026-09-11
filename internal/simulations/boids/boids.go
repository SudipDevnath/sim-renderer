// Package boids implements Craig Reynolds-style flocking as a shared.Simulation.
package boids

import (
	"math"
	"math/rand"
	"simulation-renderer/internal/shared"
)

const (
	count            = 110
	worldW           = 900.0
	worldH           = 560.0
	neighborRadius   = 78.0
	separationRadius = 28.0
	maxSpeed         = 150.0
	maxForce         = 105.0
	attractorLife    = 3.0
)

type boid struct{ pos, vel shared.Vec2 }

// Boids is a flocking simulation. A click creates a short-lived attractor.
type Boids struct {
	boids          []boid
	attractor      shared.Vec2
	attractorTimer float64
}

func New() *Boids             { return &Boids{} }
func (b *Boids) Name() string { return "Boids Flocking" }

// Init distributes a reproducible flock throughout the wrapping world.
func (b *Boids) Init() {
	rng := rand.New(rand.NewSource(23))
	b.boids = make([]boid, count)
	for i := range b.boids {
		angle := rng.Float64() * 2 * math.Pi
		speed := 65 + rng.Float64()*55
		b.boids[i] = boid{
			pos: shared.Vec2{X: (rng.Float64() - .5) * worldW, Y: (rng.Float64() - .5) * worldH},
			vel: shared.Vec2{X: math.Cos(angle) * speed, Y: math.Sin(angle) * speed},
		}
	}
	b.attractorTimer = 0
}

// Tick applies separation, alignment, and cohesion before integrating motion.
func (b *Boids) Tick(dt float64, input shared.InputState) {
	b.HandleInput(input)
	b.attractorTimer = math.Max(0, b.attractorTimer-dt)
	forces := make([]shared.Vec2, len(b.boids))
	for i, self := range b.boids {
		var separation, alignment, center shared.Vec2
		neighbors := 0
		for j, other := range b.boids {
			if i == j {
				continue
			}
			dx, dy := wrappedDelta(self.pos, other.pos)
			distSq := dx*dx + dy*dy
			if distSq >= neighborRadius*neighborRadius {
				continue
			}
			neighbors++
			alignment = add(alignment, other.vel)
			center = add(center, shared.Vec2{X: dx, Y: dy})
			if distSq < separationRadius*separationRadius && distSq > .001 {
				separation = add(separation, shared.Vec2{X: -dx / distSq, Y: -dy / distSq})
			}
		}
		force := shared.Vec2{}
		if neighbors > 0 {
			inv := 1 / float64(neighbors)
			force = add(force, steer(self.vel, scale(separation, 1), maxForce*1.25))
			force = add(force, steer(self.vel, scale(alignment, inv), maxForce*.55))
			force = add(force, steer(self.vel, scale(center, inv), maxForce*.32))
		}
		if b.attractorTimer > 0 {
			dx, dy := wrappedDelta(self.pos, b.attractor)
			force = add(force, steer(self.vel, shared.Vec2{X: dx, Y: dy}, maxForce*.8))
		}
		forces[i] = limit(force, maxForce*2)
	}
	for i := range b.boids {
		b.boids[i].vel = limit(add(b.boids[i].vel, scale(forces[i], dt)), maxSpeed)
		b.boids[i].pos = add(b.boids[i].pos, scale(b.boids[i].vel, dt))
		b.boids[i].pos.X = wrap(b.boids[i].pos.X, worldW)
		b.boids[i].pos.Y = wrap(b.boids[i].pos.Y, worldH)
	}
}

// HandleInput allows a target to be placed even while the scene is paused.
func (b *Boids) HandleInput(input shared.InputState) {
	for _, click := range input.PrimaryClicks {
		b.attractor, b.attractorTimer = click.World, attractorLife
	}
}

func (b *Boids) Frame() shared.Frame {
	prims := make([]shared.Primitive, 0, len(b.boids)*2+2)
	prims = append(prims, shared.Primitive{Kind: shared.DrawRectLines, W: worldW, H: worldH, Color: shared.Color{R: 48, G: 74, B: 96, A: 255}})
	if b.attractorTimer > 0 {
		prims = append(prims, shared.Primitive{Kind: shared.DrawCircle, Pos: b.attractor, Radius: 12 + 9*(b.attractorTimer/attractorLife), Color: shared.Color{R: 255, G: 190, B: 65, A: 255}})
	}
	for _, bird := range b.boids {
		unit := scale(bird.vel, 1/math.Max(length(bird.vel), 1))
		prims = append(prims,
			shared.Primitive{Kind: shared.DrawLine, Pos: add(bird.pos, scale(unit, -10)), Pos2: add(bird.pos, scale(unit, 14)), Color: shared.Color{R: 85, G: 190, B: 255, A: 255}},
			shared.Primitive{Kind: shared.DrawCircle, Pos: bird.pos, Radius: 3.5, Color: shared.Color{R: 220, G: 245, B: 255, A: 255}},
		)
	}
	return shared.Frame{Primitives: prims, SimName: b.Name(), ControlHint: "LMB: place attractor"}
}

func wrappedDelta(from, to shared.Vec2) (float64, float64) {
	dx, dy := to.X-from.X, to.Y-from.Y
	if dx > worldW/2 {
		dx -= worldW
	} else if dx < -worldW/2 {
		dx += worldW
	}
	if dy > worldH/2 {
		dy -= worldH
	} else if dy < -worldH/2 {
		dy += worldH
	}
	return dx, dy
}
func wrap(v, size float64) float64 {
	if v > size/2 {
		return v - size
	}
	if v < -size/2 {
		return v + size
	}
	return v
}
func add(a, b shared.Vec2) shared.Vec2           { return shared.Vec2{X: a.X + b.X, Y: a.Y + b.Y} }
func scale(v shared.Vec2, n float64) shared.Vec2 { return shared.Vec2{X: v.X * n, Y: v.Y * n} }
func length(v shared.Vec2) float64               { return math.Hypot(v.X, v.Y) }
func limit(v shared.Vec2, maximum float64) shared.Vec2 {
	if l := length(v); l > maximum {
		return scale(v, maximum/l)
	}
	return v
}
func steer(current, desired shared.Vec2, force float64) shared.Vec2 {
	if length(desired) < .001 {
		return shared.Vec2{}
	}
	desired = scale(desired, maxSpeed/length(desired))
	return limit(shared.Vec2{X: desired.X - current.X, Y: desired.Y - current.Y}, force)
}
