package main

import (
	"BeeSmartVideo/in"
	"BeeSmartVideo/logic"
	"BeeSmartVideo/out"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	width   = 640
	height  = 480
	bytesPP = 3 // RGB24
)

func printMenu() {
	fmt.Println("\n=== BeeSmartVideo Usage ===")
	fmt.Println("Usage: go run main.go [segmentation_method] [mode] [camera_index]")

	fmt.Println("\n[1] Segmentation Methods (Counting):")
	fmt.Println("  1: K-Means Clustering (fixed K=7)")
	fmt.Println("  2: Morphological Erosion + CCL")
	fmt.Println("  3: Solidity Analysis (Convex Hull)")
	fmt.Println("  4: Isoperimetric Quotient (Perimeter/Area)")

	fmt.Println("\n[2] Processing Modes:")
	fmt.Println("  --- Category: Single Frame Thresholding ---")
	fmt.Println("  1: Static Threshold (128)")
	fmt.Println("  2: Adaptive Two-Peak Threshold")
	fmt.Println("  3: Otsu's Global Threshold")
	fmt.Println("  --- Category: Frame-over-Time ---")
	fmt.Println("  4: Basic Movement Layer (Background Subtraction)")

	fmt.Println("\n[3] Camera Index:")
	fmt.Println("  0: /dev/video0 (Internal)")
	fmt.Println("  2: /dev/video2 (External)")

	fmt.Println("\nExample:")
	fmt.Println("  go run main.go 2 4 0  (Erosion + Movement Layer on Cam 0)")
}

func main() {
	if len(os.Args) < 2 {
		printMenu()
		return
	}

	method, err := strconv.Atoi(os.Args[1])
	if err != nil || method < 1 || method > 4 {
		fmt.Printf("Invalid method: %s\n", os.Args[1])
		printMenu()
		return
	}

	thresholdMode := 1
	if len(os.Args) >= 3 {
		tm, err := strconv.Atoi(os.Args[2])
		if err == nil && tm >= 1 && tm <= 4 {
			thresholdMode = tm
		}
	}

	deviceIndex := "0"
	if len(os.Args) >= 4 {
		deviceIndex = os.Args[3]
	}
	device := fmt.Sprintf("/dev/video%s", deviceIndex)

	fmt.Println("=== Video Processing Started ===")
	fmt.Printf("Selected Method: %d | Threshold Mode: %d | Camera: %s\n", method, thresholdMode, device)
	fmt.Println("Press Ctrl+C to exit")

	// Initialize input stream from webcam
	// ... (rest of main remains similar but uses ProcessWithThresholdMode)
	input, err := in.NewLiveInput(device, width, height)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize input: %v", err))
	}
	defer input.Close()

	output, err := out.NewLiveOutput(width, height)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize output: %v", err))
	}
	defer output.Close()

	processor := logic.NewProcessor(width, height, bytesPP)

	frameCount := 0
	fpsStart := time.Now()

	for {
		frameStart := time.Now()

		frame, err := input.ReadFrame()
		if err != nil {
			break
		}

		// Step 1: Binary conversion with selected threshold mode
		binaryFrame := processor.ProcessWithThresholdMode(frame, thresholdMode)

		// Step 2: Apply selected counting method
		var processedFrame []byte
		var count int

		switch method {
		case 1:
			processedFrame = processor.ApplyKMeans(binaryFrame, 7)
			count = 7
		case 2:
			processedFrame, count = processor.ApplyErosion(binaryFrame)
		case 3:
			processedFrame, count = processor.ApplyConvexHull(binaryFrame)
		case 4:
			processedFrame, count = processor.ApplyPerimeterArea(binaryFrame)
		}

		if err := output.WriteFrame(processedFrame); err != nil {
			break
		}

		frameCount++
		if frameCount%30 == 0 {
			elapsed := time.Since(fpsStart)
			fps := float64(30) / elapsed.Seconds()
			fpsStart = time.Now()
			fmt.Printf("FPS: %.1f | White pixels: %d | Count: %d | Frame time: %v\n",
				fps, processor.GetLastWhitePixelCount(), count, time.Since(frameStart))
		}
	}
	fmt.Println("\n=== Video Processing Stopped ===")
}
