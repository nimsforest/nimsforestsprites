package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	sprites "github.com/nimsforest/nimsforestsprites"
)

func main() {
	// Parse flags
	width := flag.Int("width", 1920, "Frame width")
	height := flag.Int("height", 1080, "Frame height")
	fps := flag.Int("fps", 30, "Frames per second")
	duration := flag.Duration("duration", 5*time.Second, "Demo duration")
	outputDir := flag.String("output", "", "Output directory for frames (if empty, no files saved)")
	flag.Parse()

	fmt.Println("nimsforestsprites demo")
	fmt.Printf("Resolution: %dx%d @ %d fps\n", *width, *height, *fps)
	fmt.Printf("Duration: %v\n", *duration)

	// Create renderer
	renderer, err := sprites.New(sprites.Options{
		Width:     *width,
		Height:    *height,
		FrameRate: *fps,
		Scale:     1.0,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %v\n", err)
		os.Exit(1)
	}
	defer renderer.Close()

	// Create mock state
	mockState := sprites.NewMockState()
	renderer.Update(mockState)

	// Create output directory if specified
	if *outputDir != "" {
		if err := os.MkdirAll(*outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create output directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Saving frames to: %s\n", *outputDir)
	}

	// Set up context with timeout and signal handling
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nInterrupted, shutting down...")
		cancel()
	}()

	// Start frame generation
	frames := renderer.Frames(ctx)

	// Update state periodically
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
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
	frameCount := 0
	startTime := time.Now()

	fmt.Println("Generating frames...")

	for frame := range frames {
		frameCount++

		// Save frame if output directory is specified
		if *outputDir != "" {
			filename := filepath.Join(*outputDir, fmt.Sprintf("frame_%05d.png", frameCount))
			if err := saveFrame(filename, frame); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save frame %d: %v\n", frameCount, err)
			}
		}

		// Print progress every 30 frames
		if frameCount%30 == 0 {
			elapsed := time.Since(startTime)
			actualFPS := float64(frameCount) / elapsed.Seconds()
			fmt.Printf("Generated %d frames (%.1f fps actual)\n", frameCount, actualFPS)
		}
	}

	// Final stats
	elapsed := time.Since(startTime)
	actualFPS := float64(frameCount) / elapsed.Seconds()
	fmt.Printf("\nDemo complete!\n")
	fmt.Printf("Total frames: %d\n", frameCount)
	fmt.Printf("Duration: %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Average FPS: %.1f\n", actualFPS)

	if *outputDir != "" {
		fmt.Printf("Frames saved to: %s\n", *outputDir)
	}
}

func saveFrame(filename string, img image.Image) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(f, img)
}
