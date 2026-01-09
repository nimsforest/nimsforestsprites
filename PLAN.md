# Plan: nimsforestsprites

## Overview
Go package for rendering sprite-based visualizations. Uses Ebitengine for 2D rendering. Outputs `image.Image` frames that can be consumed by nimsforestencoder or any other frame consumer.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ State Source                                                │
│   - nimsforest2 ViewModel (production)                     │
│   - Mock state (demo/testing)                              │
└─────────────────────┬───────────────────────────────────────┘
                      │ state updates
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ nimsforestsprites                                          │
│   ┌─────────────┐  ┌─────────────┐  ┌────────────────┐     │
│   │ Scene Graph │→ │ Ebitengine  │→ │ Frame Output   │     │
│   │ (sprites)   │  │ (render)    │  │ (image.Image)  │     │
│   └─────────────┘  └─────────────┘  └────────────────┘     │
│                                                             │
│   Assets: sprite sheets, animations, tiles                 │
└─────────────────────────────────────────────────────────────┘
                      │ chan image.Image
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ Frame Consumer                                              │
│   - nimsforestencoder (→ HLS → TV)                        │
│   - Direct window (dev mode)                               │
│   - Image file output (screenshots)                        │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
nimsforestsprites/
├── renderer.go       # Main Renderer struct and API
├── scene.go          # Scene graph, sprite management
├── sprites.go        # Sprite loading, animation
├── assets.go         # Asset loading (embedded or external)
├── state.go          # State interface (ViewModel abstraction)
├── demo/
│   └── main.go       # Demo: animated scene without nimsforest2
├── assets/
│   └── (embedded sprites for demo)
├── go.mod
├── go.sum
├── LICENSE
├── PLAN.md
└── README.md
```

## Core API Design

```go
package nimsforestsprites

// State is the interface for renderable state (implemented by ViewModel)
type State interface {
    // Minimal interface - actual ViewModel has more
    Lands() []Land
    // ... other accessors
}

// Land represents a renderable land/node
type Land struct {
    ID       string
    Name     string
    X, Y     float64  // Grid position
    Type     string   // "normal", "mana", etc.
    // ... processes, resources
}

// Renderer renders state to frames
type Renderer struct {
    // ...
}

// Options configures the renderer
type Options struct {
    Width      int     // Frame width (default 1920)
    Height     int     // Frame height (default 1080)
    FrameRate  int     // Target FPS (default 30)
    Scale      float64 // Sprite scale (default 1.0)
}

// Key methods:
func New(opts Options) (*Renderer, error)
func (r *Renderer) Render(state State) image.Image           // Single frame
func (r *Renderer) Frames(ctx context.Context) <-chan image.Image  // Continuous
func (r *Renderer) Update(state State)                       // Update state for next frame
func (r *Renderer) Close() error
```

## Implementation Steps

### 1. Initialize repository
- **First commit**: This PLAN.md
- Initialize Go module: `github.com/nimsforest/nimsforestsprites`
- Add LICENSE (MIT) and README

### 2. State interface (`state.go`)
- Define minimal State interface
- Create MockState for demo/testing
- Keep compatible with nimsforest2 ViewModel

### 3. Sprite management (`sprites.go`)
- Load sprite sheets (PNG)
- Animation frame management
- Sprite types: land tiles, processes, effects

### 4. Scene graph (`scene.go`)
- Manage sprite positions
- Isometric grid layout (like webview)
- Z-ordering for proper layering
- Camera/viewport

### 5. Renderer (`renderer.go`)
- Ebitengine headless rendering
- State → Scene → Frame pipeline
- Frame channel for continuous output

### 6. Demo (`demo/main.go`)
- Animated demo without nimsforest2
- MockState with fake lands/processes
- Outputs frames to encoder or window

## Dependencies
- `github.com/hajimehoshi/ebiten/v2` - Ebitengine for rendering
- Standard library for image output

## Validation (without nimsforest2)

### Demo mode
```bash
go run ./demo
```

Shows animated scene with:
- Grid of land tiles
- Moving "process" sprites
- State changes over time

### Integration test
```go
// Create renderer with mock state
renderer, _ := nimsforestsprites.New(nimsforestsprites.Options{
    Width: 1920, Height: 1080, FrameRate: 30,
})

mockState := nimsforestsprites.NewMockState()
frames := renderer.Frames(ctx)

// Update state periodically
go func() {
    for {
        mockState.Randomize()  // Simulate state changes
        renderer.Update(mockState)
        time.Sleep(time.Second)
    }
}()

// Consume frames
for frame := range frames {
    // Send to encoder, save to file, etc.
}
```

## Integration with nimsforest2

```go
import (
    sprites "github.com/nimsforest/nimsforestsprites"
    "github.com/nimsforest/nimsforest2/internal/viewmodel"
)

// Adapter to implement sprites.State from viewmodel.World
type ViewModelAdapter struct {
    world *viewmodel.World
}

func (a *ViewModelAdapter) Lands() []sprites.Land {
    // Convert viewmodel.Land to sprites.Land
}

// Usage
renderer, _ := sprites.New(sprites.Options{...})
adapter := &ViewModelAdapter{world: vm.GetWorld()}
renderer.Update(adapter)
frames := renderer.Frames(ctx)
```

## Sprite Assets

For demo, embed simple placeholder sprites:
- Green square: normal land
- Purple square: mana land
- Small colored circles: processes (trees, nims, etc.)

Production sprites can be loaded from external files.
