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
	Source         string `json:"source"`
	Method         int    `json:"method"`
	ThresholdMode  int    `json:"threshold_mode"`
	DeviceIndex    string `json:"device_index"`
	TrackingMethod int    `json:"tracking_method"`
	ShowIDs        bool   `json:"show_ids"`
	Smoothness     int    `json:"smoothness"`
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
				Title("Input Source").
				Options(videoOptions...).
				Value(&opts.Source),

			huh.NewSelect[string]().
				Title("Segmentation Method").
				Description("Select the counting algorithm").
				Options(
					huh.NewOption("K-Means Clustering", "1"),
					huh.NewOption("Morphological Erosion", "2"),
					huh.NewOption("Convex Hull Analysis", "3"),
					huh.NewOption("Isoperimetric Quotient", "4"),
					huh.NewOption("Morphological Repair", "5"),
					huh.NewOption("Distance Transform + Watershed", "6"),
					huh.NewOption("Convexity Defect Splitting", "7"),
					huh.NewOption("Skeleton-Based Splitting", "8"),
					huh.NewOption("Dynamic Area Estimation", "9"),
					huh.NewOption("Shape Filtering", "10"),
					huh.NewOption("Neighbor Merge Pass", "11"),
					huh.NewOption("Motion Direction Consistency", "12"),
					huh.NewOption("Temporal Blob Stabilization", "13"),
					huh.NewOption("Multi-Stage Pipeline", "14"),
				).
				Value(&methodStr),

			huh.NewSelect[string]().
				Title("Processing Mode").
				Description("Thresholding strategy").
				Options(
					huh.NewOption("Static (128)", "1"),
					huh.NewOption("Adaptive Two-Peak", "2"),
					huh.NewOption("Otsu's Global", "3"),
					huh.NewOption("Basic Movement Layer", "4"),
				).
				Value(&thresholdStr),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Camera Index").
				Description("Device index (e.g. 0 for /dev/video0)").
				Value(&opts.DeviceIndex).
				HideIf(func() bool { return opts.Source != "live" }),

			huh.NewSelect[string]().
				Title("Persistent Tracking").
				Description("Long-term honeybee tracking").
				Options(
					huh.NewOption("None", "0"),
					huh.NewOption("Nearest-Centroid (Baseline)", "1"),
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
				Title("Smoothness (Blur)").
				Description("Image pre-processing blur level").
				Options(
					huh.NewOption("None", "0"),
					huh.NewOption("Light (3x3)", "1"),
					huh.NewOption("Medium (5x5)", "2"),
					huh.NewOption("High (7x7)", "3"),
					huh.NewOption("Aggressive (9x9)", "4"),
				).
				Value(&smoothnessStr),
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
		Source:         "live",
		Method:         14,
		ThresholdMode:  1,
		DeviceIndex:    "0",
		TrackingMethod: 0,
		ShowIDs:        true,
		Smoothness:     0,
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
