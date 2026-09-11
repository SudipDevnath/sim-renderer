package renderer

import (
	"context"
	"fmt"
	"math"
	"simulation-renderer/internal/inputs"
	"simulation-renderer/internal/shared"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1280
	screenHeight = 720
	title        = "Simulation-Renderer"

	// targetFPS caps the render loop; 0 = uncapped.
	targetFPS = 144

	// zoomSpeed controls how fast the scroll wheel zooms.
	zoomSpeed = 0.1

	// panButton is the mouse button held to pan the camera.
	panButton = rl.MouseButtonRight

	// pauseKey toggles the simulation pause.
	pauseKey = rl.KeySpace
)

// camera holds the current view transform applied when drawing primitives.
type camera struct {
	offset shared.Vec2 // screen-space pan offset
	zoom   float64
}

// worldToScreen maps a world-space Vec2 to a raylib screen Vector2.
func (c *camera) worldToScreen(p shared.Vec2) rl.Vector2 {
	return rl.Vector2{
		X: float32(p.X*c.zoom) + float32(c.offset.X) + float32(rl.GetScreenWidth())/2,
		Y: float32(p.Y*c.zoom) + float32(c.offset.Y) + float32(rl.GetScreenHeight())/2,
	}
}

// screenToWorld maps a screen-space point into the simulation's world space.
func (c *camera) screenToWorld(p rl.Vector2) shared.Vec2 {
	return shared.Vec2{
		X: (float64(p.X) - c.offset.X - float64(rl.GetScreenWidth())/2) / c.zoom,
		Y: (float64(p.Y) - c.offset.Y - float64(rl.GetScreenHeight())/2) / c.zoom,
	}
}

// toRl converts a shared.Color to a raylib Color.
func toRl(c shared.Color) rl.Color {
	return rl.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

// renderer drives the raylib window and render loop.
// It is the ONLY package in this codebase that imports raylib.
type renderer struct {
	frame  *atomic.Pointer[shared.Frame]
	inputs *inputs.InputManager
	cam    camera
	paused bool
}

// NewRenderer wires the renderer to the shared frame pointer and input manager.
func NewRenderer(
	frame *atomic.Pointer[shared.Frame],
	im *inputs.InputManager,
) *renderer {
	return &renderer{
		frame:  frame,
		inputs: im,
		cam:    camera{zoom: 1.0},
	}
}

// Run opens the window and blocks on the render loop until the window is
// closed or ctx is cancelled.  Must be called from the main goroutine
// (raylib requirement).
func (r *renderer) Run(ctx context.Context) {
	// Request a multisampled framebuffer before creating the window. This
	// smooths the diagonal lines and curved primitive edges produced at
	// fractional camera zoom levels.
	rl.SetConfigFlags(rl.FlagMsaa4xHint | rl.FlagWindowResizable)
	rl.InitWindow(screenWidth, screenHeight, title)
	defer rl.CloseWindow()
	// Maximizing after initialization ensures raylib has already created its
	// render target before the operating system emits the resize event.
	rl.MaximizeWindow()

	rl.SetTargetFPS(targetFPS)

	for !rl.WindowShouldClose() && ctx.Err() == nil {
		r.pollInput()
		r.draw()
	}
}

// pollInput translates raw raylib events into an InputState and pushes it to
// the input manager so the simulation goroutine can read it.
// Called once per frame on the main thread — raylib input is only valid here.
func (r *renderer) pollInput() {
	// ── Pause toggle ──────────────────────────────────────────────────────────
	if rl.IsKeyPressed(pauseKey) {
		r.paused = !r.paused
	}

	// ── Pan (right-mouse drag) ────────────────────────────────────────────────
	var panDelta shared.Vec2
	if rl.IsMouseButtonDown(panButton) {
		delta := rl.GetMouseDelta()
		r.cam.offset.X += float64(delta.X)
		r.cam.offset.Y += float64(delta.Y)
		panDelta = shared.Vec2{X: float64(delta.X), Y: float64(delta.Y)}
	}

	// ── Zoom (scroll wheel) ───────────────────────────────────────────────────
	wheel := rl.GetMouseWheelMove()
	var zoomDelta float64
	if wheel != 0 {
		zoomDelta = float64(wheel) * zoomSpeed
		r.cam.zoom = math.Max(0.05, r.cam.zoom+zoomDelta)
	}

	// ── Primary click (left mouse) ────────────────────────────────────────────
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		r.inputs.RecordPrimaryClick(r.cam.screenToWorld(rl.GetMousePosition()))
	}

	r.inputs.Update(shared.InputState{
		PanDelta:  panDelta,
		ZoomDelta: zoomDelta,
		Paused:    r.paused,
	})
}

// draw loads the latest frame and renders all its primitives.
// A single atomic load at the top keeps the frame consistent for the
// entire draw call.
func (r *renderer) draw() {
	frame := r.frame.Load()

	rl.BeginDrawing()
	rl.ClearBackground(rl.Black)

	if frame != nil {
		for _, p := range frame.Primitives {
			r.drawPrimitive(p)
		}
	}

	r.drawHUD(frame)
	rl.EndDrawing()
}

// drawPrimitive dispatches a single Primitive to the appropriate raylib call.
func (r *renderer) drawPrimitive(p shared.Primitive) {
	col := toRl(p.Color)

	switch p.Kind {
	case shared.DrawCircle:
		pos := r.cam.worldToScreen(p.Pos)
		radius := float32(math.Max(1, p.Radius*r.cam.zoom))
		rl.DrawCircleV(pos, radius, col)

	case shared.DrawLine:
		start := r.cam.worldToScreen(p.Pos)
		end := r.cam.worldToScreen(p.Pos2)
		rl.DrawLineV(start, end, col)

	case shared.DrawRect:
		pos := r.cam.worldToScreen(p.Pos)
		w := float32(p.W * r.cam.zoom)
		h := float32(p.H * r.cam.zoom)
		rl.DrawRectangleV(
			rl.Vector2{X: pos.X - w/2, Y: pos.Y - h/2},
			rl.Vector2{X: w, Y: h},
			col,
		)

	case shared.DrawRectLines:
		pos := r.cam.worldToScreen(p.Pos)
		w := float32(p.W * r.cam.zoom)
		h := float32(p.H * r.cam.zoom)
		rl.DrawRectangleLinesEx(
			rl.Rectangle{X: pos.X - w/2, Y: pos.Y - h/2, Width: w, Height: h},
			1,
			col,
		)

	case shared.DrawText:
		pos := r.cam.worldToScreen(p.Pos)
		rl.DrawText(p.Label, int32(pos.X), int32(pos.Y), 16, col)

	default:
		panic(fmt.Sprintf("unknown DrawKind %d", p.Kind))
	}
}

// drawHUD renders the overlay (sim name, FPS, tick counter, controls).
func (r *renderer) drawHUD(frame *shared.Frame) {
	fps := rl.GetFPS()

	var simName string
	var tick uint64
	var stepsPerSec float64
	if frame != nil {
		simName = frame.SimName
		tick = frame.Tick
		stepsPerSec = frame.StepsPerSec
	}

	pausedStr := ""
	if r.paused {
		pausedStr = "  [PAUSED]"
	}

	line1 := fmt.Sprintf("%s%s", simName, pausedStr)
	line2 := fmt.Sprintf("FPS: %d   Steps/s: %.0f   Tick: %d   Zoom: %.2fx", fps, stepsPerSec, tick, r.cam.zoom)
	line3 := "RMB drag: pan   Scroll: zoom   Space: pause"
	if frame != nil && frame.ControlHint != "" {
		line3 = frame.ControlHint + "   " + line3
	}

	rl.DrawText(line1, 10, 10, 20, rl.RayWhite)
	rl.DrawText(line2, 10, 34, 16, rl.LightGray)
	rl.DrawText(line3, 10, 54, 14, rl.Gray)
}
