package shared

// Vec2 is a 2D vector used for positions and sizes.
type Vec2 struct {
	X, Y float64
}

// Color is a simple RGBA color value.
// Simulations set colors per-primitive; the renderer maps this to raylib.
type Color struct {
	R, G, B, A uint8
}

// Common colors simulations can use without pulling in raylib.
var (
	White   = Color{255, 255, 255, 255}
	Black   = Color{0, 0, 0, 255}
	Gray    = Color{130, 130, 130, 255}
	Red     = Color{230, 41, 55, 255}
	Green   = Color{0, 228, 48, 255}
	Blue    = Color{0, 121, 241, 255}
	Yellow  = Color{253, 249, 0, 255}
	Orange  = Color{255, 161, 0, 255}
	Purple  = Color{200, 122, 255, 255}
	SkyBlue = Color{102, 191, 255, 255}
)

// DrawKind identifies the type of a Primitive.
type DrawKind uint8

const (
	DrawCircle    DrawKind = iota // filled circle at Pos with Radius
	DrawLine                      // line from Pos to Pos2
	DrawRect                      // filled rectangle centred at Pos, W×H
	DrawRectLines                 // rectangle outline centred at Pos, W×H
	DrawText                      // Label string drawn at Pos
)

// Primitive is a single draw command produced by a simulation.
// The renderer switches on Kind and uses whichever fields are relevant.
type Primitive struct {
	Kind   DrawKind
	Pos    Vec2    // circle centre / line start / rect centre / text origin
	Pos2   Vec2    // line end (DrawLine only)
	Radius float64 // circle radius (DrawCircle only)
	W, H   float64 // dimensions (DrawRect / DrawRectLines only)
	Label  string  // text content (DrawText only)
	Color  Color
}

// Frame is the complete render output for one simulation tick.
// The scene goroutine publishes a new Frame each tick via atomic.Pointer;
// the renderer reads it lock-free every frame.
type Frame struct {
	Primitives  []Primitive
	SimName     string
	ControlHint string // simulation-specific controls appended by the renderer
	Tick        uint64
	StepsPerSec float64 // measured completed simulation updates per second
}

// InputState carries the input commands that the renderer collects each frame
// and makes available to the simulation via Tick.
type InputState struct {
	PanDelta      Vec2    // screen-space delta from panning this frame
	ZoomDelta     float64 // positive = zoom in, negative = zoom out
	Paused        bool
	PrimaryClicks []PrimaryClick // queued clicks since the previous simulation tick
}

// PrimaryClick is a left-click position expressed in world space.
type PrimaryClick struct {
	World Vec2
}

// Simulation is the interface every physics backend must implement.
//
// The scene runner calls Init once at startup, then on every tick (unless
// paused) calls Tick followed by Frame to get the latest render output.
//
// Simulations own their internal state entirely; no domain types leak into
// the runner or renderer.
type Simulation interface {
	// Name returns a human-readable label shown in the renderer HUD.
	Name() string

	// Init sets up the simulation's internal state from scratch.
	Init()

	// Tick advances the simulation by dt seconds.
	// input carries the latest player/UI state.
	Tick(dt float64, input InputState)

	// Frame converts the current internal state into a slice of draw
	// commands for the renderer.  Called once per published tick,
	// sequentially after Tick in the same goroutine.
	Frame() Frame
}

// InputHandler is an optional simulation capability for processing input
// without advancing time. The scene uses it while a simulation is paused.
type InputHandler interface {
	HandleInput(input InputState)
}
