# 🎮 Simulation Renderer

A high-performance, modular 2D simulation framework in Go, powered by [raylib-go](https://github.com/gen2brain/raylib-go).

⚡ The simulation runs in its own background goroutine at a fixed **120 Hz** tick rate, completely decoupled from the rendering loop. Even when simulation calculations get intensive, rendering and camera input stay smooth and responsive.

---

## 🏗️ Architecture

```text
main.go
  ├── go sc.Run(ctx)  ← simulation goroutine (120 Hz fixed tick)
  └── ren.Run(ctx)    ← render loop on main thread (raylib requirement)
```

🔄 Communication between the simulation and render loops keeps continuous state lock-free and queues discrete input events safely:

| 🔗 Channel | ✍️ Writer | 📖 Reader | 🧩 Type |
|---|---|---|---|
| Render primitives | `scene` goroutine | `renderer` main thread | `atomic.Pointer[shared.Frame]` |
| Continuous input state | `renderer` main thread | `scene` goroutine | `atomic.Pointer[shared.InputState]` |
| Click events | `renderer` main thread | `scene` goroutine | mutex-backed queue |

### 🧩 Generic Primitive Pipeline

Neither `scene` nor `renderer` knows about domain models such as `Body`, `Particle`, or `Cell`.

🛠️ Simulations implement a simple interface:

```go
type Simulation interface {
    Name() string
    Init()
    Tick(dt float64, input InputState)
    Frame() Frame
}
```

A `Frame` is composed of generic 2D primitives:

- 🔵 `DrawCircle` — position, radius, color
- 📏 `DrawLine` — start, end, color
- 🟦 `DrawRect` — position, width, height, color
- ▫️ `DrawRectLines` — position, width, height, color
- 💬 `DrawText` — position, label, color

✨ Simulations can represent particles, fluids, cellular automata, rigid bodies, or vector fields without changing the engine or renderer.

### 📁 Package Layout

```text
simulation-renderer/
├── src/
│   └── main.go                     — wires subsystems, CLI flags, atomic pointers
└── internal/
    ├── shared/
    │   └── shared.go               — Vec2, Color, Primitive, Frame, Simulation interface
    ├── scene/
    │   └── scene.go                — generic simulation runner loop (physics-agnostic)
    ├── renderer/
    │   └── renderer.go             — raylib window, camera pan/zoom, primitive rasterizer
    ├── inputs/
    │   └── inputs.go               — atomic InputManager bridge
    └── simulations/
        ├── life/life.go            — Conway's Game of Life with a wrapping board
        ├── gravity/gravity.go      — O(n²) N-body gravity simulation
        ├── collision/collision.go  — O(n²) elastic collision simulation with bounded arena
        ├── boids/boids.go          — interactive flocking simulation
        └── slime/slime.go          — colourful agent-based slime mould
```

> 🔒 **Raylib isolation:** Only `internal/renderer/renderer.go` imports raylib. Simulations, the scene runner, and inputs use pure Go types.

---

## 🧪 Available Simulations

### 🧬 1. Conway's Game of Life (`-sim life`, default)

- 🧱 80 × 44 wrapping cellular-automaton board
- 📜 Classic Conway rules: cells survive with 2–3 neighbours; births occur at 3
- 🚀 Starts with a Gosper glider gun and a glider; cell colour brightens with age

### 🌌 2. N-Body Gravity (`-sim gravity`)

- 🪐 128 bodies arranged in a rotating disc
- 🧮 O(n²) all-pairs gravitational attraction with softening factor `ε²` to prevent singularities
- 🎨 Bodies are colour-coded by mass

### 💥 3. N-Body Elastic Collision (`-sim collision`)

- ⚪ 60 bodies spawned inside an arena without initial overlaps
- 🧮 O(n²) pairwise collision detection, positional separation, and impulse-based elastic velocity exchange
- 🧱 Restitution on particle and wall collisions
- 🖼️ Arena boundaries rendered with `DrawRectLines`

### 🐦 4. Boids Flocking (`-sim boids`)

- 👥 110 agents using separation, alignment, and cohesion steering rules
- 🔁 A wrapping world keeps the flock flowing continuously across arena edges
- 🧲 Left-click places a temporary attractor to guide the flock

### 🧫 5. Chromatic Slime Mould (`-sim slime`) ❗NOT WORKING

- 900 sensing agents follow and reinforce an evaporating pigment field
- Electric cyan, ultraviolet, neon magenta, and amber trails are tuned for the black canvas
- Left-click places a temporary amber nutrient and pulls nearby colonies into branching paths

---

## 🎛️ Controls

| 🖱️ Input | 🎬 Action |
|---|---|
| `Right Mouse` drag | Pan the camera in world space |
| `Left Mouse` click | Add or remove a Game of Life cell |
| Scroll Wheel | Zoom in / zoom out |
| `Space` | Pause / resume the simulation |

📊 The HUD reports measured simulation throughput as `Steps/s`, alongside render FPS and total simulation tick count.

---

## ▶️ Running

```sh
# 🧬 Run the default Game of Life simulation
go run ./src

# 🧬 Run Game of Life explicitly
go run ./src -sim life

# 💥 Run the collision simulation
go run ./src -sim collision

# 🐦 Run the interactive boids flock
go run ./src -sim boids

# 🧫 Run the colourful slime mould
go run ./src -sim slime
```

---

## ➕ Adding a Custom Simulation

To create a new simulation:

1. 📁 Create a package in `internal/simulations/<your_sim>/`.
2. 🧩 Implement `shared.Simulation`: `Name()`, `Init()`, `Tick(dt, input)`, and `Frame()`.
3. 🎨 In `Frame()`, generate the `shared.Primitive` items that represent your simulation state.
4. 🔀 Add the simulation to the `-sim` switch in [`src/main.go`](src/main.go).

✅ No changes are needed in `scene`, `renderer`, or `inputs`.

---

## 📜 License

Licensed under the Apache License 2.0 — see [LICENSE](LICENSE) for details.
