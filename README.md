# nimsforestsprites

Go package for rendering sprite-based visualizations using Ebitengine. Outputs `image.Image` frames that can be consumed by nimsforestencoder or any other frame consumer.

## Features

- Headless rendering support for server-side use
- Outputs frames via `chan image.Image`
- Scene graph with sprite positioning and Z-ordering
- MockState for testing and demos without nimsforest2
- Compatible with nimsforest2 ViewModel

## Installation

```bash
go get github.com/nimsforest/nimsforestsprites
```

## Quick Start

```go
package main

import (
    "context"
    "time"

    sprites "github.com/nimsforest/nimsforestsprites"
)

func main() {
    // Create renderer
    renderer, err := sprites.New(sprites.Options{
        Width:     1920,
        Height:    1080,
        FrameRate: 30,
    })
    if err != nil {
        panic(err)
    }
    defer renderer.Close()

    // Create mock state for demo
    mockState := sprites.NewMockState()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Get frame channel
    frames := renderer.Frames(ctx)

    // Update state periodically
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                mockState.Randomize()
                renderer.Update(mockState)
            }
        }
    }()

    // Consume frames
    for frame := range frames {
        // Send to encoder, save to file, etc.
        _ = frame
    }
}
```

## Demo

Run the demo to see animated sprites without needing nimsforest2:

```bash
go run ./demo
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

## License

MIT License - see [LICENSE](LICENSE) for details.
