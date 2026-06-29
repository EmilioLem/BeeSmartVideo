package menu

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
)

const settingsFile = "settings.json"

type Options struct {
	Source          string `json:"source"`
	Method          int    `json:"method"`
	ThresholdMode   int    `json:"threshold_mode"`
	DeviceIndex     string `json:"device_index"`
	TrackingMethod  int    `json:"tracking_method"`
	ShowIDs         bool   `json:"show_ids"`
	Smoothness      int    `json:"smoothness"`
	SaveData        bool   `json:"save_data"`
	LoopVideo       bool   `json:"loop_video"`
	EnableTelemetry bool   `json:"enable_telemetry"`
}

func GetOptions() Options {
	opts := loadSettings()

	// Escanear videos disponibles
	files, _ := os.ReadDir("./videoSamples")
	var videoOptions []huh.Option[string]
	for _, f := range files {
		if !f.IsDir() {
			videoOptions = append(videoOptions, huh.NewOption(f.Name(), "./videoSamples/"+f.Name()))
		}
	}
	videoOptions = append(videoOptions, huh.NewOption("Live Webcam", "live"))

	// Use temporary strings for huh selection if needed, or bind directly if types match
	var methodStr string = strconv.Itoa(opts.Method)
	var thresholdStr string = strconv.Itoa(opts.ThresholdMode)
	var trackingStr string = strconv.Itoa(opts.TrackingMethod)
	var smoothnessStr string = strconv.Itoa(opts.Smoothness)

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
		),
	)

	err := form.Run()
	if err != nil {
		fmt.Printf("Error running form: %v\n", err)
		os.Exit(1)
	}

	// Convert back to int
	opts.Method, _ = strconv.Atoi(methodStr)
	opts.ThresholdMode, _ = strconv.Atoi(thresholdStr)
	opts.TrackingMethod, _ = strconv.Atoi(trackingStr)
	opts.Smoothness, _ = strconv.Atoi(smoothnessStr)

	saveSettings(opts)
	return opts
}

func loadSettings() Options {
	defaultOpts := Options{
		Source:          "live",
		Method:          1,
		ThresholdMode:   1,
		DeviceIndex:     "0",
		TrackingMethod:  0,
		ShowIDs:         true,
		Smoothness:      0,
		SaveData:        false,
		LoopVideo:       false,
		EnableTelemetry: false,
	}

	file, err := os.Open(settingsFile)
	if err != nil {
		return defaultOpts
	}
	defer file.Close()

	var opts Options
	if err := json.NewDecoder(file).Decode(&opts); err != nil {
		return defaultOpts
	}
	if opts.Source == "" {
		opts.Source = "live"
	}
	return opts
}

func saveSettings(opts Options) {
	file, err := os.Create(settingsFile)
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
