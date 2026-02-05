package main

import (
	"fmt"
	"time"
	"video-processor/in"
	"video-processor/logic"
	"video-processor/out"
)

const (
	width   = 640
	height  = 480
	bytesPP = 3 // RGB24
)

func main() {
	fmt.Println("=== Video Processing Started ===")
	fmt.Println("Press Ctrl+C to exit\n")

	// Initialize input stream from webcam
	input, err := in.NewLiveInput("/dev/video0", width, height)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize input: %v", err))
	}
	defer input.Close()

	// Initialize output display
	output, err := out.NewLiveOutput(width, height)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize output: %v", err))
	}
	defer output.Close()

	// Initialize processing logic
	processor := logic.NewProcessor(width, height, bytesPP)

	// Processing loop
	frameCount := 0
	fpsStart := time.Now()

	for {
		frameStart := time.Now()

		// Read frame from webcam
		frame, err := input.ReadFrame()
		if err != nil {
			break
		}

		// Process the frame
		// Step 1: Convert to binary grayscale and invert
		binaryFrame := processor.BinaryGrayscaleInverseWithThreshold(frame, 128)

		// Step 2: Apply k-means clustering on white pixels
		clusteredFrame := processor.ApplyKMeans(binaryFrame, 7)

		// Write to output
		if err := output.WriteFrame(clusteredFrame); err != nil {
			break
		}

		frameCount++
		frameTime := time.Since(frameStart)

		// Calculate FPS every 30 frames
		if frameCount%30 == 0 {
			elapsed := time.Since(fpsStart)
			fps := float64(30) / elapsed.Seconds()
			fpsStart = time.Now()

			whitePixels := processor.GetLastWhitePixelCount()
			fmt.Printf("FPS: %.1f | White pixels: %d | Clusters: 7 | Frame time: %v\n",
				fps, whitePixels, frameTime)
		}
	}

	fmt.Println("\n=== Video Processing Stopped ===")
}
