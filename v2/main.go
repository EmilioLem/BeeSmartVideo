// BeeSmartVideo v2 entry point.
//
// v2 keeps the classic pipeline from main.go but unifies the interface into
// exactly TWO modes. There is nothing else to remember.
//
//	MODE 1 - CLI (default)
//	  go run .
//	  Headless. Reads v2/settings.json and processes video with no window.
//	  Ideal for systemd / servers. Override the source with --source:
//	    go run . --source ./videoSamples/40secBeesWithPolen.mp4
//	    go run . --source live
//	  A bare path is a shorthand: go run . ./videoSamples/other.mp4
//
//	MODE 2 - huh + GUI
//	  go run . --huh
//	  Opens the interactive huh form to choose settings (including the video
//	  source), then shows the processed video in an ffplay window.
//	  Aliases: --gui, -gui, huhForm, --tui, -tui, -i, --interactive.
//
// v2 is self-contained: it always runs relative to this folder, so
// settings.json, ./videoSamples and ./dataset/ sit next to this file no
// matter where the command is launched from.
package main

import (
	"BeeSmartVideo/aruco"
	"BeeSmartVideo/datasetgen"
	"BeeSmartVideo/in"
	"BeeSmartVideo/logic"
	"BeeSmartVideo/menu"
	"BeeSmartVideo/mqtt"
	"BeeSmartVideo/out"
	"BeeSmartVideo/out/webPageStats"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	inWidth  = 1280 // Capture resolution
	inHeight = 720
	width    = 426 // Processing resolution (240p)
	height   = 240
	bytesPP  = 3     // RGB24
	bgDelta  = 0.005 // Background adaptation speed (~1 min at 30fps)
	// Full-res scale used by Subsample3x, needed to map track centroids back.
	fullResScale = 3
	// How many frames a newly confirmed track is offered to the ArUco worker
	// before it is marked as "no marker". Bounds the cost for untagged bees.
	arucoMaxAttempts = 8
	// Directory (relative to v2/) where --debugImage writes bee crops.
	debugImageDir = "debugImages"
	// Keep only the last N debug crops: the filename index wraps at this value.
	debugImageRing = 100
)

func main() {
	anchorToV2Dir()
	args := parseArgs(os.Args[1:])
	opts := loadOptions(args)

	arucoClient := startArUco(args, opts)
	if arucoClient != nil {
		defer arucoClient.Close()
	}

	if args.debugImage {
		if err := os.MkdirAll(debugImageDir, 0755); err != nil {
			fmt.Printf("Warning: debug images disabled: %v\n", err)
			args.debugImage = false
		} else {
			fmt.Printf("Debug crops enabled: keeping the last %d bee crops in %s/\n", debugImageRing, debugImageDir)
		}
	}

	run(opts, arucoClient, args.debugImage)
}

// anchorToV2Dir changes the working directory to this file's folder so v2
// always reads v2/settings.json, scans v2/videoSamples and writes v2/dataset,
// regardless of where the process was started from.
func anchorToV2Dir() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return
	}
	dir := filepath.Dir(file)
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		return
	}
	_ = os.Chdir(dir)
}

// v2Args is the parsed command line of v2.
type v2Args struct {
	interactive bool   // true => MODE 2 (huh + GUI)
	source      string // source override for MODE 1
	sourceSet   bool
	noArUco     bool   // disable the ArUco worker for this run
	arucoScript string // path to ArUcoReader02.py
	python      string // python interpreter for the worker
	debugImage  bool   // save the last N detected bee crops for troubleshooting
}

// parseArgs understands the whole v2 interface:
//
//	--huh | --gui | huhForm | --tui | -i | --interactive   -> MODE 2
//	--source <path|live>                                    -> CLI source override
//	--no-aruco                                              -> disable ArUco worker
//	--aruco-script <path>                                   -> worker script path
//	--python <path>                                         -> python interpreter
//	<path>                                                  -> shorthand for --source
//
// Unknown flags are ignored, so stray arguments never crash the tool.
func parseArgs(argv []string) v2Args {
	var args v2Args

	for i := 0; i < len(argv); i++ {
		arg := argv[i]

		switch {
		case isInteractiveFlag(arg):
			args.interactive = true
		case arg == "--no-aruco" || arg == "-no-aruco":
			args.noArUco = true
		case arg == "--debugImage" || arg == "--debug-image" || arg == "-debugImage":
			args.debugImage = true
		case arg == "--aruco-script":
			if i+1 < len(argv) {
				i++
				args.arucoScript = argv[i]
			}
		case strings.HasPrefix(arg, "--aruco-script="):
			args.arucoScript = strings.TrimPrefix(arg, "--aruco-script=")
		case arg == "--python":
			if i+1 < len(argv) {
				i++
				args.python = argv[i]
			}
		case strings.HasPrefix(arg, "--python="):
			args.python = strings.TrimPrefix(arg, "--python=")
		case arg == "--source" || arg == "-source":
			if i+1 < len(argv) {
				i++
				args.source = argv[i]
				args.sourceSet = true
			}
		case strings.HasPrefix(arg, "--source="):
			args.source = strings.TrimPrefix(arg, "--source=")
			args.sourceSet = true
		case strings.HasPrefix(arg, "-source="):
			args.source = strings.TrimPrefix(arg, "-source=")
			args.sourceSet = true
		case arg != "" && !strings.HasPrefix(arg, "-"):
			// Bare positional argument is a shorthand for --source.
			args.source = arg
			args.sourceSet = true
		}
	}

	return args
}

// isInteractiveFlag reports whether arg selects MODE 2 (huh + GUI).
func isInteractiveFlag(arg string) bool {
	switch arg {
	case "--huh", "huhForm", "--gui", "-gui", "--tui", "-tui", "-i", "--interactive":
		return true
	}
	return false
}

// loadOptions resolves the two modes and applies the CLI source override.
func loadOptions(args v2Args) menu.Options {
	if args.interactive {
		opts := menu.GetInteractiveOptions()
		if args.sourceSet {
			fmt.Println("Note: --source is ignored in huh + GUI mode; pick the source in the form.")
		}
		return opts
	}

	opts := menu.GetHeadlessOptions()
	if args.sourceSet {
		opts.Source = args.source
		fmt.Printf("Source set from CLI: %s\n", opts.Source)
	}
	return opts
}

// startArUco launches the persistent Python ArUco worker unless it is disabled
// by settings or the --no-aruco flag. If it cannot start (e.g. python or cv2
// missing) the pipeline keeps running without marker decoding.
func startArUco(args v2Args, opts menu.Options) *aruco.Client {
	if !opts.EnableArUco || args.noArUco {
		return nil
	}

	script := args.arucoScript
	if script == "" {
		script = "./ArUcoReader02.py"
	}
	python := args.python
	if python == "" {
		python = "python3"
	}

	client, err := aruco.NewClient(python, script)
	if err != nil {
		fmt.Printf("Warning: ArUco worker disabled: %v\n", err)
		return nil
	}
	fmt.Printf("ArUco worker started: %s %s\n", python, script)
	return client
}

// decodeTrackMarkers sends the crops of newly confirmed tracks to the worker.
// Each track is offered to the decoder at most arucoMaxAttempts times; a decoded
// marker is stored on the track, and once the attempts run out the track is
// marked MarkerNone so untagged bees are not retried forever.
func decodeTrackMarkers(client *aruco.Client, p *logic.Processor, fullFrame []byte, cropSizeProcessing int) {
	cropSizeFull := cropSizeProcessing * fullResScale

	var images []aruco.Image
	for i := range p.Tracks {
		t := &p.Tracks[i]
		if t.Status != "confirmed" || t.MarkerID != logic.MarkerUnknown || t.MarkerAttempts >= arucoMaxAttempts {
			continue
		}
		b64, err := aruco.EncodeCropJPEG(
			fullFrame, inWidth, inHeight, t.Centroid.X, t.Centroid.Y, fullResScale, cropSizeFull,
		)
		if err != nil {
			continue
		}
		images = append(images, aruco.Image{TrackID: t.ID, JPEGB64: b64})
	}

	if len(images) == 0 {
		return
	}

	results, err := client.Decode(images)
	if err != nil {
		// stderr keeps the two-line live status region intact.
		fmt.Fprintf(os.Stderr, "ArUco decode error: %v\n", err)
		return
	}

	for _, r := range results {
		for i := range p.Tracks {
			t := &p.Tracks[i]
			if t.ID != r.TrackID {
				continue
			}
			t.MarkerAttempts++
			if r.Found && r.MarkerID != nil {
				t.MarkerID = *r.MarkerID
				t.MarkerLabel = r.Label
			} else if t.MarkerAttempts >= arucoMaxAttempts {
				t.MarkerID = logic.MarkerNone
			}
			break
		}
	}
}

// saveDebugCrops writes the crop of every confirmed track to debugImages/ for
// troubleshooting. Names cycle through bee_000.jpg .. bee_099.jpg so the folder
// always holds only the most recent debugImageRing crops (unordered by design).
func saveDebugCrops(p *logic.Processor, fullFrame []byte, cropSizeProcessing int, index *int) {
	cropSizeFull := cropSizeProcessing * fullResScale
	for i := range p.Tracks {
		t := &p.Tracks[i]
		if t.Status != "confirmed" {
			continue
		}
		jpegBytes, err := aruco.CropJPEG(
			fullFrame, inWidth, inHeight, t.Centroid.X, t.Centroid.Y, fullResScale, cropSizeFull,
		)
		if err != nil {
			continue
		}

		name := fmt.Sprintf("bee_%03d.jpg", *index)
		*index = (*index + 1) % debugImageRing
		if err := os.WriteFile(filepath.Join(debugImageDir, name), jpegBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "debug image write error: %v\n", err)
			return
		}
	}
}

// printStatus renders the two live status lines in place:
//
//	FPS: ... | UP: ... | Active: ... | Blobs: ...
//	ArUco IDs: track 7=H1-0042, track 12=H3-0242
//
// It uses ANSI cursor movement (like the '\r' it replaces), so both lines
// update every refresh. The first call skips the cursor-up because no status
// has been printed yet.
func printStatus(first *bool, fps float64, up, down, active, blobs int, tracks []logic.Track) {
	line1 := fmt.Sprintf("FPS: %.1f | UP: %d | DOWN: %d | Active: %d | Blobs: %d", fps, up, down, active, blobs)
	line2 := "ArUco IDs: " + formatMarkerIDs(tracks)

	if *first {
		fmt.Printf("\r%s\x1b[K\n", line1)
		fmt.Printf("%s\x1b[K", line2)
		*first = false
		return
	}
	fmt.Printf("\x1b[1A\r%s\x1b[K\n", line1)
	fmt.Printf("\r%s\x1b[K", line2)
}

// formatMarkerIDs lists the decoded ArUco labels with their track ids, e.g.
// "track 7=H1-0042, track 12=H3-0242". It caps the line length so terminal
// wrapping cannot break the two-line in-place update.
func formatMarkerIDs(tracks []logic.Track) string {
	const maxLen = 100

	var b strings.Builder
	shown := 0
	total := 0
	for _, t := range tracks {
		if t.MarkerID < 0 {
			continue
		}
		total++

		label := t.MarkerLabel
		if label == "" {
			label = fmt.Sprintf("%d", t.MarkerID)
		}
		entry := fmt.Sprintf("track %d=%s", t.ID, label)
		if b.Len() > 0 {
			entry = ", " + entry
		}
		if b.Len()+len(entry) > maxLen {
			continue
		}
		b.WriteString(entry)
		shown++
	}

	if shown == 0 {
		return "(none)"
	}
	if total > shown {
		b.WriteString(fmt.Sprintf(" (+%d more)", total-shown))
	}
	return b.String()
}

// run executes the video pipeline for the resolved options. It is identical
// to the original main.go loop and works for both CLI and huh + GUI modes.
func run(opts menu.Options, arucoClient *aruco.Client, debugImage bool) {
	var mqttClient *mqtt.Client
	if opts.EnableTelemetry {
		var err error
		mqttClient, err = mqtt.NewClient()
		if err != nil {
			fmt.Printf("Warning: failed to initialize MQTT client: %v\n", err)
		}
	}

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

	var output *out.LiveOutput
	if opts.IsInteractive {
		var err error
		output, err = out.NewLiveOutput(width, height)
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize output: %v", err))
		}
		defer output.Close()
	}

	processor := logic.NewProcessor(width, height, bytesPP, bgDelta)
	processor.ShowIDs = showIDs
	processor.Smoothness = smoothness
	processor.UseRedChannel = opts.RedChannel

	var datagen *datasetgen.V1Generator
	if opts.SaveData {
		var errGen error
		datagen, errGen = datasetgen.NewV1Generator("dataset", opts.CropSize*3)
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
	firstStatus := true
	debugIndex := 0

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

			// Step 4: Decode ArUco markers on newly confirmed tracks, then draw
			// both the track id and the decoded marker id.
			if arucoClient != nil {
				decodeTrackMarkers(arucoClient, processor, frame, opts.CropSize)
			}
			if debugImage {
				saveDebugCrops(processor, frame, opts.CropSize, &debugIndex)
			}
			processor.OverlayTracks(processedFrame)

			if datagen != nil {
				datagen.ProcessFrameTracks(processor.Tracks, processor.FullResFrame, inWidth, inHeight, processor.FrameCount)
			}
		} else {
			activeCount = len(blobs)
		}

		if output != nil {
			if err := output.WriteFrame(processedFrame); err != nil {
				break
			}
		}

		frameCount++
		if frameCount%10 == 0 {
			up, down := processor.GetCounts()
			elapsed := time.Since(fpsStart)
			lastFPS = 10.0 / elapsed.Seconds()
			fpsStart = time.Now()
			printStatus(&firstStatus, lastFPS, up, down, activeCount, len(blobs), processor.Tracks)

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

			// Update MQTT Telemetry
			if mqttClient != nil {
				mqttClient.PublishStats(activeCount, up, down, avgPath, avgSpeed, avgDist, lastFPS)
			}
		}
	}
	fmt.Println("\n=== Video Processing Stopped ===")
}
