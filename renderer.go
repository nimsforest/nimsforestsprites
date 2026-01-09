package nimsforestsprites

import (
	"context"
	"image"
	"image/color"
	"image/draw"
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
	UseGPU    bool    // Use GPU rendering via ebiten (default true)
}

// DefaultOptions returns the default renderer options
func DefaultOptions() Options {
	return Options{
		Width:     1920,
		Height:    1080,
		FrameRate: 30,
		Scale:     1.0,
		UseGPU:    true,
	}
}

// Renderer renders state to frames
type Renderer struct {
	opts   Options
	state  State
	mu     sync.RWMutex
	closed bool
	tick   int

	// For GPU mode: ebiten game running in background
	game      *ebitenGame
	gameReady chan struct{}
	frameCh   chan image.Image
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

	r := &Renderer{
		opts:      opts,
		gameReady: make(chan struct{}),
		frameCh:   make(chan image.Image, 2),
	}

	if opts.UseGPU {
		r.startEbitenGame()
	}

	return r, nil
}

// startEbitenGame starts ebiten in a background goroutine
func (r *Renderer) startEbitenGame() {
	r.game = &ebitenGame{
		renderer:  r,
		offscreen: ebiten.NewImage(r.opts.Width, r.opts.Height),
	}

	go func() {
		// Set window size (even for headless, this sets the logical size)
		ebiten.SetWindowSize(r.opts.Width, r.opts.Height)
		ebiten.SetWindowTitle("nimsforestsprites")

		// Run game in background - this blocks until game exits
		ebiten.RunGame(r.game)
	}()

	// Wait a moment for ebiten to initialize
	time.Sleep(100 * time.Millisecond)
	close(r.gameReady)
}

// ebitenGame implements ebiten.Game interface
type ebitenGame struct {
	renderer  *Renderer
	offscreen *ebiten.Image
	ready     bool
}

func (g *ebitenGame) Update() error {
	g.ready = true
	return nil
}

func (g *ebitenGame) Draw(screen *ebiten.Image) {
	if !g.ready {
		return
	}

	g.renderer.mu.RLock()
	state := g.renderer.state
	tick := g.renderer.tick
	g.renderer.mu.RUnlock()

	// Clear
	g.offscreen.Clear()

	// Draw scene
	g.drawScene(g.offscreen, state, tick)

	// Copy to screen
	screen.DrawImage(g.offscreen, nil)

	// Capture frame for output
	select {
	case g.renderer.frameCh <- g.captureFrame():
	default:
	}
}

func (g *ebitenGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.renderer.opts.Width, g.renderer.opts.Height
}

func (g *ebitenGame) captureFrame() image.Image {
	bounds := image.Rect(0, 0, g.renderer.opts.Width, g.renderer.opts.Height)
	img := image.NewRGBA(bounds)
	g.offscreen.ReadPixels(img.Pix)
	return img
}

func (g *ebitenGame) drawScene(screen *ebiten.Image, state State, tick int) {
	// Draw dark background
	screen.Fill(color.RGBA{20, 25, 30, 255})

	if state == nil {
		return
	}

	// Draw lands as grid
	lands := state.Lands()
	tileSize := int(64 * g.renderer.opts.Scale)
	startX := 100
	startY := 100

	for _, land := range lands {
		x := float32(startX + int(land.X*float64(tileSize)))
		y := float32(startY + int(land.Y*float64(tileSize)))

		// Get land color with pulse animation
		landColor := getLandColor(land.Type)
		pulse := float64(tick%60) / 60.0
		if pulse > 0.5 {
			pulse = 1.0 - pulse
		}
		landColor.A = uint8(200 + pulse*55)

		// Draw land tile using vector
		drawFilledRect(screen, x, y, float32(tileSize-2), float32(tileSize-2), landColor)
	}

	// Draw processes
	processes := state.Processes()
	for _, proc := range processes {
		px := float32(startX+int(proc.X)*tileSize) + float32(tileSize/2)
		py := float32(startY+int(proc.Y)*tileSize) + float32(tileSize/2)

		// Bounce animation
		bounce := sin(float64(tick)/10.0+proc.X*0.5) * 3
		py += float32(bounce)

		procColor := getProcessColor(proc.Type)
		drawFilledCircle(screen, px, py, 8, procColor)
	}

	// Frame indicator
	frameX := float32(10 + (tick%60)*2)
	drawFilledRect(screen, frameX, 10, 4, 4, color.RGBA{100, 200, 100, 200})
}

// Update updates the state for the next frame
func (r *Renderer) Update(state State) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = state
}

// Render renders a single frame with the current state
func (r *Renderer) Render(state State) image.Image {
	r.mu.Lock()
	r.state = state
	r.tick++
	r.mu.Unlock()

	if r.closed {
		return nil
	}

	if r.opts.UseGPU {
		// Wait for ebiten to be ready
		<-r.gameReady

		// Wait for next frame from ebiten
		select {
		case frame := <-r.frameCh:
			return frame
		case <-time.After(100 * time.Millisecond):
			// Timeout - return software rendered frame
			return r.renderFrameSoftware(state)
		}
	}

	return r.renderFrameSoftware(state)
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
				r.tick++
				state := r.state
				r.mu.Unlock()

				var frame image.Image
				if r.opts.UseGPU {
					select {
					case frame = <-r.frameCh:
					default:
						frame = r.renderFrameSoftware(state)
					}
				} else {
					frame = r.renderFrameSoftware(state)
				}

				select {
				case frames <- frame:
				default:
				}
			}
		}
	}()

	return frames
}

// renderFrameSoftware renders a frame using pure Go (no GPU)
func (r *Renderer) renderFrameSoftware(state State) image.Image {
	r.mu.RLock()
	tick := r.tick
	r.mu.RUnlock()

	img := image.NewRGBA(image.Rect(0, 0, r.opts.Width, r.opts.Height))

	// Draw dark background
	bg := color.RGBA{20, 25, 30, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	if state == nil {
		return img
	}

	// Draw lands as grid
	lands := state.Lands()
	tileSize := int(64 * r.opts.Scale)
	startX := 100
	startY := 100

	for _, land := range lands {
		x := startX + int(land.X)*tileSize
		y := startY + int(land.Y)*tileSize

		landColor := getLandColor(land.Type)
		pulse := float64(tick%60) / 60.0
		if pulse > 0.5 {
			pulse = 1.0 - pulse
		}
		landColor.A = uint8(200 + pulse*55)

		fillRectSW(img, x, y, tileSize-2, tileSize-2, landColor, r.opts.Width, r.opts.Height)
	}

	// Draw processes
	processes := state.Processes()
	for _, proc := range processes {
		px := startX + int(proc.X)*tileSize + tileSize/2
		py := startY + int(proc.Y)*tileSize + tileSize/2

		bounce := sin(float64(tick)/10.0+proc.X*0.5) * 3
		py += int(bounce)

		procColor := getProcessColor(proc.Type)
		fillCircleSW(img, px, py, 8, procColor, r.opts.Width, r.opts.Height)
	}

	// Frame indicator
	frameX := 10 + (tick%60)*2
	fillRectSW(img, frameX, 10, 4, 4, color.RGBA{100, 200, 100, 200}, r.opts.Width, r.opts.Height)

	return img
}

func getLandColor(landType string) color.RGBA {
	switch landType {
	case "mana":
		return color.RGBA{80, 60, 120, 255}
	case "forest":
		return color.RGBA{40, 80, 50, 255}
	case "water":
		return color.RGBA{40, 60, 100, 255}
	default:
		return color.RGBA{60, 70, 60, 255}
	}
}

func getProcessColor(procType string) color.RGBA {
	switch procType {
	case "tree":
		return color.RGBA{60, 150, 60, 255}
	case "nim":
		return color.RGBA{200, 180, 100, 255}
	case "mana":
		return color.RGBA{150, 100, 200, 255}
	default:
		return color.RGBA{150, 150, 150, 255}
	}
}

// Software rendering helpers
func fillRectSW(img *image.RGBA, x, y, w, h int, c color.RGBA, maxW, maxH int) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < maxW && py >= 0 && py < maxH {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func fillCircleSW(img *image.RGBA, cx, cy, radius int, c color.RGBA, maxW, maxH int) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				px, py := cx+x, cy+y
				if px >= 0 && px < maxW && py >= 0 && py < maxH {
					img.SetRGBA(px, py, c)
				}
			}
		}
	}
}

// Ebiten drawing helpers
func drawFilledRect(img *ebiten.Image, x, y, w, h float32, c color.RGBA) {
	rect := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	draw.Draw(rect, rect.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)

	ebitenRect := ebiten.NewImageFromImage(rect)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	img.DrawImage(ebitenRect, op)
}

func drawFilledCircle(img *ebiten.Image, cx, cy, radius float32, c color.RGBA) {
	r := int(radius)
	size := r*2 + 1
	circle := image.NewRGBA(image.Rect(0, 0, size, size))

	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				circle.SetRGBA(x+r, y+r, c)
			}
		}
	}

	ebitenCircle := ebiten.NewImageFromImage(circle)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(cx)-float64(r), float64(cy)-float64(r))
	img.DrawImage(ebitenCircle, op)
}

// sin returns sine approximation
func sin(x float64) float64 {
	x = x - float64(int(x/(2*3.14159)))*2*3.14159
	if x > 3.14159 {
		x -= 2 * 3.14159
	}
	x2 := x * x
	return x * (1 - x2/6 + x2*x2/120)
}

// Close closes the renderer
func (r *Renderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
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
