package menu

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
)

const SettingsFile = "settings.json"

// Options holds all runtime parameters for BeeSmartVideo detection.
// Hardcoded defaults are defined in DefaultOptions() for easy modification.
type Options struct {
	// Source: Input video source. Can be a video file path (e.g. "./videoSamples/40secBeesWithPolen.mp4") or "live" for camera input.
	Source string `json:"source"`

	// Method: Main processing algorithm ID (1 = Efficient filtering v1).
	Method int `json:"method"`

	// ThresholdMode: Thresholding strategy mode:
	//   11 = Static (78) [-50]
	//   12 = Static (88) [-40]
	//   7  = Static (98) [-30]
	//   6  = Static (108) [-20]
	//   5  = Static (118) [-10]
	//   1  = Static (128) [0] (Default static threshold)
	//   8  = Static (138) [+10]
	//   9  = Static (148) [+20]
	//   10 = Static (158) [+30]
	//   13 = Static (168) [+40]
	//   14 = Static (178) [+50]
	ThresholdMode int `json:"threshold_mode"`

	// DeviceIndex: Camera device index when Source == "live" (e.g., "0" for /dev/video0, "2" for /dev/video2).
	DeviceIndex string `json:"device_index"`

	// TrackingMethod: Honeybee persistent tracking algorithm:
	//   0 = None
	//   1 = Feature Consensus Tracking
	//   2 = Hungarian Assignment
	//   3 = Kalman Filter
	//   4 = Multi-Feature Matching
	//   5 = Motion-Gated Assignment
	//   6 = Deep Path Tracker
	TrackingMethod int `json:"tracking_method"`

	// ShowIDs: Overlay persistent track IDs on processed video frame output.
	ShowIDs bool `json:"show_ids"`

	// Smoothness: Gaussian blur level applied before binary conversion (0 = disabled).
	Smoothness int `json:"smoothness"`

	// SaveData: Export frame tracking data and cropped images to dataset/ folder.
	SaveData bool `json:"save_data"`

	// CropSize: Side of the square bee crop exported to dataset/, expressed in
	// PROCESSING pixels (the 426x240 scale). The actual full-res crop is
	// CropSize * 3, so the default of 64 equals a 192x192 px image.
	CropSize int `json:"crop_size"`

	// LoopVideo: Loop video playback continuously when using video file source.
	LoopVideo bool `json:"loop_video"`

	// EnableTelemetry: Publish live counting statistics over MQTT broker.
	EnableTelemetry bool `json:"enable_telemetry"`

	// RedChannel: Threshold on the red channel directly instead of luminance.
	// The hive entrance is illuminated with RED LEDs only, so the red channel
	// carries the bee signal. Default true; disable only for white light.
	RedChannel bool `json:"red_channel"`

	// EnableArUco: Decode ArUco markers on confirmed bee tracks through the
	// Python worker (ArUcoReader02.py), attaching the marker id to each track.
	// Only used by the v2 entry point. Default true.
	EnableArUco bool `json:"enable_aruco"`

	// IsInteractive is set at runtime depending on whether TUI mode was requested.
	IsInteractive bool `json:"-"`
}

// DefaultOptions returns hardcoded fallback settings for headless CLI & systemd operation.
func DefaultOptions() Options {
	return Options{
		Source:          "live",
		Method:          1,
		ThresholdMode:   1,
		DeviceIndex:     "0",
		TrackingMethod:  0,
		ShowIDs:         true,
		Smoothness:      0,
		SaveData:        false,
		CropSize:        64,
		LoopVideo:       false,
		EnableTelemetry: false,
		RedChannel:      true,
		EnableArUco:     true,
		IsInteractive:   false,
	}
}

// LoadSettings reads options from settings.json. If the file does not exist or is invalid, it returns DefaultOptions.
func LoadSettings() Options {
	opts := DefaultOptions()

	file, err := os.Open(SettingsFile)
	if err != nil {
		return opts
	}
	defer file.Close()

	// Decode over the defaults so fields missing from settings.json keep their
	// default value (important for booleans that default to true, e.g. red_channel).
	if err := json.NewDecoder(file).Decode(&opts); err != nil {
		return DefaultOptions()
	}
	if opts.Source == "" {
		opts.Source = DefaultOptions().Source
	}
	if opts.CropSize <= 0 {
		opts.CropSize = DefaultOptions().CropSize
	}
	return opts
}

// SaveSettings writes the given options to settings.json.
func SaveSettings(opts Options) {
	file, err := os.Create(SettingsFile)
	if err != nil {
		fmt.Printf("Warning: could not save settings: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(opts); err != nil {
		fmt.Printf("Warning: error encoding settings: %v\n", err)
	}
}

// GetOptions checks CLI arguments:
// - If ran with "huhForm", "--tui", "-tui", "--gui", "-gui", "-i", or "--interactive", it runs the interactive TUI menu.
// - Otherwise, it runs in headless mode loading settings from settings.json (systemd ready).
func GetOptions() Options {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "huhForm", "--tui", "-tui", "--gui", "-gui", "-i", "--interactive":
			return GetInteractiveOptions()
		}
	}

	return GetHeadlessOptions()
}

// GetHeadlessOptions loads settings non-interactively without displaying a TUI menu.
func GetHeadlessOptions() Options {
	opts := LoadSettings()
	opts.IsInteractive = false
	fmt.Println("=== Mode: Headless CLI (systemd ready) ===")
	fmt.Println("Configuration loaded from 'settings.json'.")
	fmt.Println("Graphical output window (ffplay): Disabled")
	fmt.Println("Tip: Pass 'huhForm' or '--tui' flag to open the interactive TUI menu.")
	return opts
}

// GetInteractiveOptions displays the interactive TUI form (huh) to configure and save options.
func GetInteractiveOptions() Options {
	fmt.Println("=== Mode: Interactive TUI Menu ===")
	opts := LoadSettings()
	opts.IsInteractive = true

	// Scan available video samples
	files, _ := os.ReadDir("./videoSamples")
	var videoOptions []huh.Option[string]
	for _, f := range files {
		if !f.IsDir() {
			videoOptions = append(videoOptions, huh.NewOption(f.Name(), "./videoSamples/"+f.Name()))
		}
	}
	videoOptions = append(videoOptions, huh.NewOption("Live Webcam", "live"))

	var methodStr string = strconv.Itoa(opts.Method)
	var thresholdStr string = strconv.Itoa(opts.ThresholdMode)
	var trackingStr string = strconv.Itoa(opts.TrackingMethod)
	var smoothnessStr string = strconv.Itoa(opts.Smoothness)
	var cropStr string = strconv.Itoa(opts.CropSize)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Options(videoOptions...).
				Value(&opts.Source),

			huh.NewSelect[string]().
				Title("Main Processing Method").
				Description("Select the processing algorithm").
				Options(
					huh.NewOption("Efficient filtering v1", "1"),
				).
				Value(&methodStr),

			huh.NewSelect[string]().
				Title("Processing Mode").
				Description("Thresholding strategy").
				Options(
					huh.NewOption("Static (78) [-50]", "11"),
					huh.NewOption("Static (88) [-40]", "12"),
					huh.NewOption("Static (98) [-30]", "7"),
					huh.NewOption("Static (108) [-20]", "6"),
					huh.NewOption("Static (118) [-10]", "5"),
					huh.NewOption("Static (128) [0]", "1"),
					huh.NewOption("Static (138) [+10]", "8"),
					huh.NewOption("Static (148) [+20]", "9"),
					huh.NewOption("Static (158) [+30]", "10"),
					huh.NewOption("Static (168) [+40]", "13"),
					huh.NewOption("Static (178) [+50]", "14"),
				).
				Value(&thresholdStr),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Camera Index").
				Description("Device index (e.g. 0 for /dev/video0)").
				Value(&opts.DeviceIndex),
		).WithHideFunc(func() bool {
			return opts.Source != "live"
		}),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Persistent Tracking").
				Description("Long-term honeybee tracking").
				Options(
					huh.NewOption("None", "0"),
					huh.NewOption("Feature Consensus Tracking", "1"),
					huh.NewOption("Hungarian Assignment", "2"),
					huh.NewOption("Kalman Filter", "3"),
					huh.NewOption("Multi-Feature Matching", "4"),
					huh.NewOption("Motion-Gated Assignment", "5"),
					huh.NewOption("Deep Path Tracker", "6"),
				).
				Value(&trackingStr),

			huh.NewConfirm().
				Title("Show IDs").
				Description("Overlay track IDs on video").
				Value(&opts.ShowIDs),

			huh.NewSelect[string]().
				Title("Bee Crop Size").
				Description("Exported crop side, in processing pixels (full-res = x3)").
				Options(
					huh.NewOption("Small (32 proc px / 96 full-res)", "32"),
					huh.NewOption("Medium (48 proc px / 144 full-res)", "48"),
					huh.NewOption("Default (64 proc px / 192 full-res)", "64"),
					huh.NewOption("Large (96 proc px / 288 full-res)", "96"),
					huh.NewOption("X-Large (128 proc px / 384 full-res)", "128"),
				).
				Value(&cropStr),

			huh.NewConfirm().
				Title("Export AI Dataset").
				Description("Save CSV and crop images to dataset/").
				Value(&opts.SaveData),

			huh.NewConfirm().
				Title("Loop Video Playback").
				Description("Repeat video file in loop (does not affect live source)").
				Value(&opts.LoopVideo),

			huh.NewConfirm().
				Title("Enable MQTT Telemetry").
				Description("Publish live counting telemetry to an MQTT broker").
				Value(&opts.EnableTelemetry),

			huh.NewConfirm().
				Title("Red-LED Mode (red channel)").
				Description("Threshold on the red channel - hive entrance lit with red LEDs only").
				Value(&opts.RedChannel),

			huh.NewConfirm().
				Title("Enable ArUco Decoding").
				Description("Read marker IDs from confirmed bee crops (v2 only; needs Python + OpenCV)").
				Value(&opts.EnableArUco),
		),
	)

	err := form.Run()
	if err != nil {
		fmt.Printf("Error running form: %v\n", err)
		os.Exit(1)
	}

	opts.Method, _ = strconv.Atoi(methodStr)
	opts.ThresholdMode, _ = strconv.Atoi(thresholdStr)
	opts.TrackingMethod, _ = strconv.Atoi(trackingStr)
	opts.Smoothness, _ = strconv.Atoi(smoothnessStr)
	opts.CropSize, _ = strconv.Atoi(cropStr)

	SaveSettings(opts)
	return opts
}
