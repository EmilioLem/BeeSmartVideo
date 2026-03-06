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
	bytesPP = 3     // RGB24
	bgDelta = 0.005 // Background adaptation speed (~1 min at 30fps)
)

func printMenu() {
	fmt.Println("\n=== BeeSmartVideo Usage ===")
	fmt.Println("Usage: go run main.go [method] [mode] [cam] [tracking]")

	fmt.Println("\n[1] Segmentation Methods (Counting):")
	fmt.Println("  --- Category: Initial Methods ---")
	fmt.Println("  1: K-Means Clustering (fixed K=7)")
	fmt.Println("  2: Morphological Erosion + CCL")
	fmt.Println("  3: Solidity Analysis (Convex Hull)")
	fmt.Println("  4: Isoperimetric Quotient (Perimeter/Area)")
	fmt.Println("  --- Category: Improved Methods ---")
	fmt.Println("  5: Morphological Repair (Close/Fill/Open)")
	fmt.Println("  6: Distance Transform + Watershed")
	fmt.Println("  7: Convexity Defect Splitting")
	fmt.Println("  8: Skeleton-Based Splitting")
	fmt.Println("  9: Dynamic Area Estimation (Median)")
	fmt.Println("  10: Shape Filtering (Descriptors)")
	fmt.Println("  11: Neighbor Merge Pass")
	fmt.Println("  12: Motion Direction Consistency")
	fmt.Println("  13: Temporal Blob Stabilization")
	fmt.Println("  14: Multi-Stage Segmentation Pipeline")

	fmt.Println("\n[2] Processing Modes:")
	fmt.Println("  --- Category: Single Frame Thresholding ---")
	fmt.Println("  1: Static Threshold (128)")
	fmt.Println("  2: Adaptive Two-Peak Threshold")
	fmt.Println("  3: Otsu's Global Threshold")
	fmt.Println("  --- Category: Frame-over-Time ---")
	fmt.Println("  4: Basic Movement Layer (Running Average)")

	fmt.Println("\n[3] Persistent Tracking Methods:")
	fmt.Println("  0: No Tracking (Raw Detection Count)")
	fmt.Println("  1: Nearest-Centroid Tracker (Baseline)")
	fmt.Println("  2: Hungarian Assignment (Optimal Matching)")
	fmt.Println("  3: Kalman Filter (Prediction-Based)")
	fmt.Println("  4: Multi-Feature Matching (Shape/Area)")
	fmt.Println("  5: Motion-Gated Assignment (Plausibility)")

	fmt.Println("\n[4] Camera Index:")
	fmt.Println("  0: /dev/video0 (Internal)")
	fmt.Println("  2: /dev/video2 (External)")

	fmt.Println("\nExample:")
	fmt.Println("  go run main.go 14 4 0 3  (Pipeline + Movement + Kalman on Cam 0)")
}

func main() {
	if len(os.Args) < 2 {
		printMenu()
		return
	}

	method, err := strconv.Atoi(os.Args[1])
	if err != nil || method < 1 || method > 14 {
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

	trackingMethod := 0
	if len(os.Args) >= 5 {
		tr, err := strconv.Atoi(os.Args[4])
		if err == nil && tr >= 0 && tr <= 5 {
			trackingMethod = tr
		}
	}

	fmt.Println("=== Video Processing Started ===")
	fmt.Printf("Selected Method: %d | Threshold: %d | Tracker: %d | Camera: %s\n",
		method, thresholdMode, trackingMethod, device)
	fmt.Println("Press Ctrl+C to exit")

	// Initialize input stream from webcam
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

	processor := logic.NewProcessor(width, height, bytesPP, bgDelta)

	frameCount := 0
	fpsStart := time.Now()

	for {

		frame, err := input.ReadFrame()
		if err != nil {
			break
		}

		// Step 1: Binary conversion with selected threshold mode
		binaryFrame := processor.ProcessWithThresholdMode(frame, thresholdMode)

		// Step 2: Apply selected counting method
		var processedFrame []byte
		var blobs []logic.Blob

		switch method {
		case 1:
			processedFrame, blobs = processor.ApplyKMeans(binaryFrame, 7)
		case 2:
			processedFrame, blobs = processor.ApplyErosion(binaryFrame)
		case 3:
			processedFrame, blobs = processor.ApplyConvexHull(binaryFrame)
		case 4:
			processedFrame, blobs = processor.ApplyPerimeterArea(binaryFrame)
		case 5:
			processedFrame, blobs = processor.ApplyMorphRepair(binaryFrame)
		case 6:
			processedFrame, blobs = processor.ApplyWatershed(binaryFrame)
		case 7:
			processedFrame, blobs = processor.ApplyDefectSplitting(binaryFrame)
		case 8:
			processedFrame, blobs = processor.ApplySkeletonSplitting(binaryFrame)
		case 9:
			processedFrame, blobs = processor.ApplyDynamicAreaEstimation(binaryFrame)
		case 10:
			processedFrame, blobs = processor.ApplyShapeFiltering(binaryFrame)
		case 11:
			processedFrame, blobs = processor.ApplyNeighborMerge(binaryFrame)
		case 12:
			processedFrame, blobs = processor.ApplyMotionConsistency(binaryFrame)
		case 13:
			processedFrame, blobs = processor.ApplyTemporalStabilisation(binaryFrame)
		case 14:
			processedFrame, blobs = processor.ApplyAdvancedPipeline(binaryFrame)
		}

		// Step 3: Apply Persistent Tracking
		activeCount := 0
		if trackingMethod > 0 {
			activeCount = processor.ApplyTracking(blobs, trackingMethod)
			processor.OverlayTracks(processedFrame)
		} else {
			activeCount = len(blobs)
		}

		if err := output.WriteFrame(processedFrame); err != nil {
			break
		}

		frameCount++
		if frameCount%10 == 0 {
			up, down := processor.GetCounts()
			elapsed := time.Since(fpsStart)
			fps := 10.0 / elapsed.Seconds()
			fpsStart = time.Now()
			fmt.Printf("\rFPS: %.1f | UP: %d | DOWN: %d | Active: %d | Blobs: %d   ", fps, up, down, activeCount, len(blobs))
		}
	}
	fmt.Println("\n=== Video Processing Stopped ===")
}
