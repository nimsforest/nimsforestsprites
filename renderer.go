package nimsforestsprites

import (
	"context"
	"image"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Options configures the renderer
type Options struct {
	Width     int     // Frame width (default 1920)
	Height    int     // Frame height (default 1080)
	FrameRate int     // Target FPS (default 30)
	Scale     float64 // Sprite scale (default 1.0)
}

// DefaultOptions returns the default renderer options
func DefaultOptions() Options {
	return Options{
		Width:     1920,
		Height:    1080,
		FrameRate: 30,
		Scale:     1.0,
	}
}

// Renderer renders state to frames using Ebitengine
type Renderer struct {
	opts   Options
	scene  *Scene
	state  State
	mu     sync.RWMutex
	closed bool

	// Offscreen image for headless rendering
	offscreen *ebiten.Image
}

// New creates a new renderer with the given options
func New(opts Options) (*Renderer, error) {
	// Apply defaults
	if opts.Width == 0 {
		opts.Width = 1920
	}
	if opts.Height == 0 {
		opts.Height = 1080
	}
	if opts.FrameRate == 0 {
		opts.FrameRate = 30
	}
	if opts.Scale == 0 {
		opts.Scale = 1.0
	}

	// Create offscreen image for headless rendering
	offscreen := ebiten.NewImage(opts.Width, opts.Height)

	r := &Renderer{
		opts:      opts,
		scene:     NewScene(opts.Width, opts.Height, opts.Scale),
		offscreen: offscreen,
	}

	return r, nil
}

// Update updates the state for the next frame
func (r *Renderer) Update(state State) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.state = state
	if r.scene != nil {
		r.scene.UpdateFromState(state)
	}
}

// Render renders a single frame with the current state
func (r *Renderer) Render(state State) image.Image {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	// Update scene from state
	r.scene.UpdateFromState(state)

	// Clear offscreen
	r.offscreen.Clear()

	// Draw scene to offscreen
	r.scene.Draw(r.offscreen)

	// Return a copy of the offscreen image
	return r.captureFrame()
}

// Frames returns a channel that receives continuous frames
func (r *Renderer) Frames(ctx context.Context) <-chan image.Image {
	frames := make(chan image.Image, 2)

	go func() {
		defer close(frames)

		frameDuration := time.Second / time.Duration(r.opts.FrameRate)
		ticker := time.NewTicker(frameDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.mu.Lock()
				if r.closed {
					r.mu.Unlock()
					return
				}

				// Update scene animation
				r.scene.Update()

				// Clear and draw
				r.offscreen.Clear()
				r.scene.Draw(r.offscreen)

				// Capture frame
				frame := r.captureFrame()
				r.mu.Unlock()

				// Send frame (non-blocking if channel is full)
				select {
				case frames <- frame:
				default:
					// Drop frame if channel is full
				}
			}
		}
	}()

	return frames
}

// captureFrame captures the current offscreen image to an image.Image
// Must be called with lock held
func (r *Renderer) captureFrame() image.Image {
	bounds := image.Rect(0, 0, r.opts.Width, r.opts.Height)
	img := image.NewRGBA(bounds)

	// Read pixels from offscreen
	r.offscreen.ReadPixels(img.Pix)

	return img
}

// Close closes the renderer and releases resources
func (r *Renderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true
	r.offscreen = nil
	r.scene = nil

	return nil
}

// Size returns the current frame dimensions
func (r *Renderer) Size() (width, height int) {
	return r.opts.Width, r.opts.Height
}

// FrameRate returns the configured frame rate
func (r *Renderer) FrameRate() int {
	return r.opts.FrameRate
}
