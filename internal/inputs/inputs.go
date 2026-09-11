package inputs

import (
	"simulation-renderer/internal/shared"
	"sync"
	"sync/atomic"
)

// InputManager is a thin thread-safe bridge between the renderer (which
// translates raw raylib events into InputState) and the simulation goroutine
// (which reads InputState to adjust behaviour).
//
// Continuous input state is exchanged atomically. Discrete click events are
// protected by a small mutex-backed queue so every event reaches the scene.
type InputManager struct {
	state  *atomic.Pointer[shared.InputState]
	mu     sync.Mutex
	clicks []shared.PrimaryClick
}

// NewInputManager creates an InputManager backed by the provided atomic
// pointer.  main.go owns the pointer lifetime and passes it to both here and
// the scene so both sides share the same storage.
func NewInputManager(state *atomic.Pointer[shared.InputState]) *InputManager {
	// Store a valid zero-value so readers never see nil.
	zero := &shared.InputState{}
	state.Store(zero)
	return &InputManager{state: state}
}

// Update replaces the current InputState.  Called by the renderer once per
// frame after polling raylib events.
func (im *InputManager) Update(s shared.InputState) {
	// Clicks are events, not state: they are queued separately so multiple
	// clicks between simulation ticks are all delivered.
	s.PrimaryClicks = nil
	im.state.Store(&s)
}

// RecordPrimaryClick appends a left-click event for the scene to consume.
func (im *InputManager) RecordPrimaryClick(world shared.Vec2) {
	im.mu.Lock()
	im.clicks = append(im.clicks, shared.PrimaryClick{World: world})
	im.mu.Unlock()
}

// Load returns a snapshot of the current InputState.  Called by the
// simulation goroutine each tick.
func (im *InputManager) Load() shared.InputState {
	return *im.state.Load()
}

// DrainPrimaryClicks returns every click received since the previous drain.
func (im *InputManager) DrainPrimaryClicks() []shared.PrimaryClick {
	im.mu.Lock()
	clicks := im.clicks
	im.clicks = nil
	im.mu.Unlock()
	return clicks
}
