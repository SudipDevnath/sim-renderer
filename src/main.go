package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"simulation-renderer/internal/inputs"
	"simulation-renderer/internal/renderer"
	"simulation-renderer/internal/scene"
	"simulation-renderer/internal/shared"
	"simulation-renderer/internal/simulations/boids"
	"simulation-renderer/internal/simulations/collision"
	"simulation-renderer/internal/simulations/compound"
	"simulation-renderer/internal/simulations/gravity"
	"simulation-renderer/internal/simulations/life"
	"simulation-renderer/internal/simulations/slime"
	"sync/atomic"
)

func main() {
	// ── CLI ───────────────────────────────────────────────────────────────────
	simFlag := flag.String("sim", "life", "simulation to run: life | gravity | collision | boids | slime")
	flag.Parse()

	// ── Pick simulation backend ───────────────────────────────────────────────
	simulations := map[string]func() shared.Simulation{
		"life":      func() shared.Simulation { return life.New() },
		"gravity":   func() shared.Simulation { return gravity.New() },
		"collision": func() shared.Simulation { return collision.New() },
		"boids":     func() shared.Simulation { return boids.New() },
		"slime":     func() shared.Simulation { return slime.New() },
		"comp":      func() shared.Simulation { return compound.New() },
	}
	newSimulation, ok := simulations[*simFlag]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown simulation %q; choose life, gravity, collision, boids, or slime\n", *simFlag)
		os.Exit(2)
	}
	sim := newSimulation()

	// ── Root context — cancelled when the render loop exits ───────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Shared atomic state ───────────────────────────────────────────────────
	// Scene writes; renderer reads.
	var frame atomic.Pointer[shared.Frame]

	// Renderer writes; scene reads.
	var inputState atomic.Pointer[shared.InputState]

	// ── Construct subsystems ──────────────────────────────────────────────────
	im := inputs.NewInputManager(&inputState)

	sc := scene.NewScene(sim, &frame, im)
	sc.Init()

	ren := renderer.NewRenderer(&frame, im)

	// ── Launch ────────────────────────────────────────────────────────────────
	go sc.Run(ctx) // sim goroutine — independent of render
	ren.Run(ctx)   // blocks on main thread (raylib requirement)
}
