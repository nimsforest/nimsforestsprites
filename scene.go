package nimsforestsprites

import (
	"image/color"
	"sort"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Scene manages the scene graph and sprite rendering
type Scene struct {
	width, height int
	scale         float64

	mu       sync.RWMutex
	lands    []LandSprite
	sprites  []Sprite
	tick     int
	cameraX  float64
	cameraY  float64
	tileSize float64
}

// Sprite represents a renderable sprite in the scene
type Sprite struct {
	X, Y     float64 // World position
	Z        float64 // Z-order (higher = drawn later)
	Width    float64
	Height   float64
	Color    color.RGBA
	Type     string
	ID       string
	Progress float64 // Animation progress 0.0-1.0
}

// LandSprite represents a land tile sprite
type LandSprite struct {
	X, Y   float64 // Grid position
	Type   string
	ID     string
	Color  color.RGBA
	Width  float64
	Height float64
}

// NewScene creates a new scene with the given dimensions
func NewScene(width, height int, scale float64) *Scene {
	return &Scene{
		width:    width,
		height:   height,
		scale:    scale,
		tileSize: 80 * scale,
		cameraX:  float64(width) / 2,
		cameraY:  float64(height) / 3,
	}
}

// UpdateFromState updates the scene from a state object
func (s *Scene) UpdateFromState(state State) {
	if state == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update lands
	lands := state.Lands()
	s.lands = make([]LandSprite, len(lands))
	for i, land := range lands {
		s.lands[i] = LandSprite{
			X:      land.X,
			Y:      land.Y,
			Type:   land.Type,
			ID:     land.ID,
			Color:  s.getLandColor(land.Type),
			Width:  s.tileSize,
			Height: s.tileSize * 0.5, // Isometric height
		}
	}

	// Update process sprites
	processes := state.Processes()
	s.sprites = make([]Sprite, len(processes))
	for i, proc := range processes {
		s.sprites[i] = Sprite{
			X:        proc.X,
			Y:        proc.Y,
			Z:        proc.Y + 0.1, // Z-order based on Y position
			Width:    s.tileSize * 0.3,
			Height:   s.tileSize * 0.3,
			Color:    s.getProcessColor(proc.Type),
			Type:     proc.Type,
			ID:       proc.ID,
			Progress: proc.Progress,
		}
	}

	// Sort sprites by Z-order
	sort.Slice(s.sprites, func(i, j int) bool {
		return s.sprites[i].Z < s.sprites[j].Z
	})
}

// Update updates the scene animation state
func (s *Scene) Update() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tick++

	// Animate sprites
	for i := range s.sprites {
		// Add subtle bobbing animation
		s.sprites[i].Progress += 0.02
		if s.sprites[i].Progress > 1.0 {
			s.sprites[i].Progress = 0.0
		}
	}
}

// Draw renders the scene to the given image
func (s *Scene) Draw(screen *ebiten.Image) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Fill background
	screen.Fill(color.RGBA{30, 40, 50, 255})

	// Draw lands first (background layer)
	for _, land := range s.lands {
		s.drawLand(screen, land)
	}

	// Draw sprites (foreground layer)
	for _, sprite := range s.sprites {
		s.drawSprite(screen, sprite)
	}

	// Draw frame counter in corner (debug)
	s.drawDebugInfo(screen)
}

// gridToScreen converts grid coordinates to screen coordinates (isometric)
func (s *Scene) gridToScreen(gx, gy float64) (sx, sy float64) {
	// Isometric projection
	sx = (gx-gy)*s.tileSize*0.5 + s.cameraX
	sy = (gx+gy)*s.tileSize*0.25 + s.cameraY
	return sx, sy
}

func (s *Scene) drawLand(screen *ebiten.Image, land LandSprite) {
	sx, sy := s.gridToScreen(land.X, land.Y)

	// Draw isometric diamond shape
	halfW := land.Width * 0.5
	halfH := land.Height

	// Draw filled diamond
	path := &vector.Path{}
	path.MoveTo(float32(sx), float32(sy-halfH))
	path.LineTo(float32(sx+halfW), float32(sy))
	path.LineTo(float32(sx), float32(sy+halfH))
	path.LineTo(float32(sx-halfW), float32(sy))
	path.Close()

	// Fill with land color
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(land.Color.R) / 255
		vs[i].ColorG = float32(land.Color.G) / 255
		vs[i].ColorB = float32(land.Color.B) / 255
		vs[i].ColorA = float32(land.Color.A) / 255
	}

	// Create a 1x1 white image for solid color rendering
	whiteImg := ebiten.NewImage(3, 3)
	whiteImg.Fill(color.White)

	screen.DrawTriangles(vs, is, whiteImg, &ebiten.DrawTrianglesOptions{})

	// Draw border
	vector.StrokeLine(screen, float32(sx), float32(sy-halfH), float32(sx+halfW), float32(sy), 2, color.RGBA{60, 80, 100, 255}, true)
	vector.StrokeLine(screen, float32(sx+halfW), float32(sy), float32(sx), float32(sy+halfH), 2, color.RGBA{60, 80, 100, 255}, true)
	vector.StrokeLine(screen, float32(sx), float32(sy+halfH), float32(sx-halfW), float32(sy), 2, color.RGBA{60, 80, 100, 255}, true)
	vector.StrokeLine(screen, float32(sx-halfW), float32(sy), float32(sx), float32(sy-halfH), 2, color.RGBA{60, 80, 100, 255}, true)
}

func (s *Scene) drawSprite(screen *ebiten.Image, sprite Sprite) {
	sx, sy := s.gridToScreen(sprite.X, sprite.Y)

	// Add bobbing animation
	bobOffset := float64(s.tick%60) / 60.0 * 3.14159 * 2
	yOffset := sin(bobOffset+sprite.Progress*6.28) * 3

	// Draw sprite as a circle
	radius := float32(sprite.Width * 0.5)
	vector.DrawFilledCircle(screen, float32(sx), float32(sy+yOffset), radius, sprite.Color, true)

	// Draw progress indicator around sprite
	if sprite.Progress > 0 {
		progressAngle := float32(sprite.Progress * 6.28318)
		vector.StrokeArc(screen, float32(sx), float32(sy+yOffset), radius+4, 0, progressAngle, 2, color.RGBA{255, 255, 255, 128}, true)
	}

	// Draw outline
	vector.StrokeCircle(screen, float32(sx), float32(sy+yOffset), radius, 2, color.RGBA{255, 255, 255, 100}, true)
}

func (s *Scene) drawDebugInfo(screen *ebiten.Image) {
	// Draw small indicator in corner
	frameInSecond := s.tick % 60
	x := float32(10 + frameInSecond*2)
	vector.DrawFilledRect(screen, x, 10, 4, 4, color.RGBA{100, 200, 100, 200}, true)
}

func (s *Scene) getLandColor(landType string) color.RGBA {
	switch landType {
	case "mana":
		return color.RGBA{80, 60, 120, 255} // Purple for mana lands
	case "normal":
		return color.RGBA{50, 100, 60, 255} // Green for normal lands
	default:
		return color.RGBA{60, 80, 70, 255} // Default gray-green
	}
}

func (s *Scene) getProcessColor(procType string) color.RGBA {
	switch procType {
	case "tree":
		return color.RGBA{80, 180, 80, 255} // Green
	case "nim":
		return color.RGBA{200, 180, 80, 255} // Gold
	case "mana":
		return color.RGBA{150, 100, 200, 255} // Purple
	case "harvest":
		return color.RGBA{200, 100, 80, 255} // Orange-red
	default:
		return color.RGBA{150, 150, 150, 255} // Gray
	}
}

// Simple sin function using Taylor series approximation
func sin(x float64) float64 {
	// Normalize to -pi to pi
	for x > 3.14159 {
		x -= 6.28318
	}
	for x < -3.14159 {
		x += 6.28318
	}

	// Taylor series: sin(x) = x - x^3/3! + x^5/5! - ...
	x2 := x * x
	x3 := x2 * x
	x5 := x3 * x2
	x7 := x5 * x2

	return x - x3/6 + x5/120 - x7/5040
}
