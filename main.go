package main

import (
	"BeeSmartVideo/in"
	"BeeSmartVideo/logic"
	"BeeSmartVideo/menu"
	"BeeSmartVideo/out"
	"BeeSmartVideo/out/webPageStats"
	"fmt"
	"time"
)

const (
	inWidth  = 1280 // Capture resolution
	inHeight = 720
	width    = 426 // Processing resolution (240p)
	height   = 240
	bytesPP  = 3     // RGB24
	bgDelta  = 0.005 // Background adaptation speed (~1 min at 30fps)
)

func main() {
	opts := menu.GetOptions()

	method := opts.Method
	thresholdMode := opts.ThresholdMode
	device := fmt.Sprintf("/dev/video%s", opts.DeviceIndex)
	trackingMethod := opts.TrackingMethod
	showIDs := opts.ShowIDs
	smoothness := opts.Smoothness

	fmt.Println("\n=== Video Processing Started ===")
	fmt.Printf("Selected Method: %d | Threshold: %d | Tracker: %d | Camera: %s | IDs: %v | Smoothness: %d\n",
		method, thresholdMode, trackingMethod, device, showIDs, smoothness)
	fmt.Println("Dropping first 25 frames for light stabilization...")
	fmt.Println("Press Ctrl+C to exit")

	// Start Stats Server
	webPageStats.StartServer(8080)

	// Initialize input stream from webcam
	input, err := in.NewLiveInput(device, inWidth, inHeight)
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
	processor.ShowIDs = showIDs
	processor.Smoothness = smoothness

	frameCount := 0
	fpsStart := time.Now()
	lastFPS := 0.0

	for {
		frame, err := input.ReadFrame()
		if err != nil {
			break
		}

		// Store full res frame for future use
		processor.FullResFrame = frame

		// Downsample to processing resolution (240p)
		smallFrame := processor.Subsample3x(frame, inWidth, inHeight)

		// Step 0: Apply Smoothness (Blur) if enabled
		if processor.Smoothness > 0 {
			smallFrame = processor.ApplyBlur(smallFrame)
		}

		// Step 1: Binary conversion with selected threshold mode
		binaryFrame := processor.ProcessWithThresholdMode(smallFrame, thresholdMode)

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
			lastFPS = 10.0 / elapsed.Seconds()
			fpsStart = time.Now()
			fmt.Printf("\rFPS: %.1f | UP: %d | DOWN: %d | Active: %d | Blobs: %d   ", lastFPS, up, down, activeCount, len(blobs))

			// Calculate average path length
			avgPath := 0.0
			if len(processor.Tracks) > 0 {
				totalPath := 0
				for _, t := range processor.Tracks {
					totalPath += len(t.History)
				}
				avgPath = float64(totalPath) / float64(len(processor.Tracks))
			}

			// Update Web Dashboard Stats
			webPageStats.UpdateStats(activeCount, up, down, avgPath, lastFPS)
		}
	}
	fmt.Println("\n=== Video Processing Stopped ===")
}
