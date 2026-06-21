package main

import (
	"BeeSmartVideo/datasetgen"
	"BeeSmartVideo/in"
	"BeeSmartVideo/logic"
	"BeeSmartVideo/menu"
	"BeeSmartVideo/out"
	"BeeSmartVideo/out/webPageStats"
	"fmt"
	"math"
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
	
	var inputPath string
	if opts.Source == "live" {
		inputPath = fmt.Sprintf("/dev/video%s", opts.DeviceIndex)
	} else {
		inputPath = opts.Source
	}
	
	trackingMethod := opts.TrackingMethod
	showIDs := opts.ShowIDs
	smoothness := opts.Smoothness

	fmt.Println("\n=== Video Processing Started ===")
	fmt.Printf("Selected Method: %d | Threshold: %d | Tracker: %d | Source: %s | IDs: %v | Smoothness: %d\n",
		method, thresholdMode, trackingMethod, inputPath, showIDs, smoothness)
	fmt.Println("Dropping first 25 frames for light stabilization...")
	fmt.Println("Press Ctrl+C to exit")

	// Start Stats Server
	webPageStats.StartServer(8080)

	// Initialize input stream from webcam or file
	input, err := in.NewLiveInput(inputPath, inWidth, inHeight)
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

	var datagen *datasetgen.V1Generator
	if opts.SaveData {
		var errGen error
		datagen, errGen = datasetgen.NewV1Generator("dataset")
		if errGen != nil {
			fmt.Printf("Warning: failed to start dataset mapping: %v\n", errGen)
			datagen = nil // disable save data on error
		} else {
			defer datagen.Close()
		}
	}

	frameCount := 0
	fpsStart := time.Now()
	lastFPS := 0.0

	for {
		frame, err := input.ReadFrame()
		if err != nil {
			if opts.LoopVideo && opts.Source != "live" {
				// Restart video input for looping
				input.Close()
				input, err = in.NewLiveInput(inputPath, inWidth, inHeight)
				if err != nil {
					panic(fmt.Sprintf("Failed to restart input: %v", err))
				}
				continue
			}
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
			processedFrame, blobs = processor.ApplyEfficientFilteringV1(binaryFrame)
		default:
			processedFrame, blobs = processor.ApplyEfficientFilteringV1(binaryFrame)
		}

		// Step 3: Apply Persistent Tracking
		activeCount := 0
		if trackingMethod > 0 {
			activeCount = processor.ApplyTracking(blobs, trackingMethod)
			processor.OverlayTracks(processedFrame)
			
			if datagen != nil {
				datagen.ProcessFrameTracks(processor.Tracks, processor.FullResFrame, inWidth, inHeight, processor.FrameCount)
			}
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

			// Calculate average path length, speed, and distance
			avgPath := 0.0
			avgSpeed := 0.0
			avgDist := 0.0

			numTracks := len(processor.Tracks)
			if numTracks > 0 {
				totalPath := 0
				totalSpeed := 0.0
				for _, t := range processor.Tracks {
					totalPath += len(t.History)
					// Speed = sqrt(vx^2 + vy^2)
					speed := math.Sqrt(t.VX*t.VX + t.VY*t.VY)
					totalSpeed += speed
				}
				avgPath = float64(totalPath) / float64(numTracks)
				avgSpeed = totalSpeed / float64(numTracks)

				// Average distance between all pairs
				if numTracks > 1 {
					totalDist := 0.0
					count := 0
					for i := 0; i < numTracks; i++ {
						for j := i + 1; j < numTracks; j++ {
							dx := float64(processor.Tracks[i].Centroid.X - processor.Tracks[j].Centroid.X)
							dy := float64(processor.Tracks[i].Centroid.Y - processor.Tracks[j].Centroid.Y)
							totalDist += math.Sqrt(dx*dx + dy*dy)
							count++
						}
					}
					avgDist = totalDist / float64(count)
				}
			}

			// Update Web Dashboard Stats
			webPageStats.UpdateStats(activeCount, up, down, avgPath, avgSpeed, avgDist, lastFPS)
		}
	}
	fmt.Println("\n=== Video Processing Stopped ===")
}
