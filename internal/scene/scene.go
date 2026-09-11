package scene

import (
	"context"
	"simulation-renderer/internal/inputs"
	"simulation-renderer/internal/shared"
	"sync/atomic"
	"time"
)

const (
	// SimHz is the target simulation update rate in Hz.
	SimHz = 120
)

// scene is a generic simulation runner.
// It owns the tick loop, pause logic, and frame publishing.
// All physics and state are fully encapsulated inside the Simulation it holds.
type scene struct {
	sim      shared.Simulation
	frame    *atomic.Pointer[shared.Frame]
	inputs   *inputs.InputManager
	tick     uint64
	stepMark time.Time
	steps    uint64
	stepsSec float64
}

// NewScene wires the runner to a Simulation backend, the shared frame
// pointer, and the input manager.  main.go owns all three lifetimes.
func NewScene(
	sim shared.Simulation,
	frame *atomic.Pointer[shared.Frame],
	im *inputs.InputManager,
) *scene {
	return &scene{
		sim:    sim,
		frame:  frame,
		inputs: im,
	}
}

// Init delegates to the Simulation to set up its internal state, then
// publishes the first frame so the renderer has something to draw immediately.
func (s *scene) Init() {
	s.sim.Init()
	s.stepMark = time.Now()
	s.publishFrame()
}

// Run is the simulation goroutine entry point.  It blocks until ctx is
// cancelled (i.e. the window closes).
func (s *scene) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second / SimHz)
	defer ticker.Stop()

	dt := 1.0 / float64(SimHz)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			in := s.inputs.Load()
			in.PrimaryClicks = s.inputs.DrainPrimaryClicks()

			if in.Paused {
				if handler, ok := s.sim.(shared.InputHandler); ok {
					handler.HandleInput(in)
				}
				// Re-publish so the renderer HUD reflects the paused state,
				// while optional input handlers can still update their state.
				s.publishFrame()
				continue
			}

			s.sim.Tick(dt, in)
			s.tick++
			s.recordStep(time.Now())
			s.publishFrame()
		}
	}
}

// recordStep maintains a one-second measurement window for the HUD. It uses
// wall time rather than SimHz so the displayed value reflects real throughput.
func (s *scene) recordStep(now time.Time) {
	s.steps++
	elapsed := now.Sub(s.stepMark)
	if elapsed < time.Second {
		return
	}
	s.stepsSec = float64(s.steps) / elapsed.Seconds()
	s.steps = 0
	s.stepMark = now
}

// publishFrame asks the simulation for its current frame, stamps the tick
// counter, and stores it atomically for the renderer.
func (s *scene) publishFrame() {
	f := s.sim.Frame()
	f.Tick = s.tick
	f.StepsPerSec = s.stepsSec
	s.frame.Store(&f)
}
