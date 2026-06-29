package main

import (
	"BeeSmartVideo/logic"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PointResult struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type FramePreAnnotation struct {
	FrameName string        `json:"frameName"`
	Points    []PointResult `json:"points"`
}

type Settings struct {
	ThresholdMode int `json:"threshold_mode"`
	Smoothness    int `json:"smoothness"`
}

func loadThresholdFromSettings(settingsPath string) int {
	file, err := os.Open(settingsPath)
	if err != nil {
		return 1 // default static threshold 128
	}
	defer file.Close()
	var s Settings
	if err := json.NewDecoder(file).Decode(&s); err != nil || s.ThresholdMode == 0 {
		return 1
	}
	return s.ThresholdMode
}

func resizeTo240p(srcRGB []byte, srcW, srcH, dstW, dstH int) []byte {
	dst := make([]byte, dstW*dstH*3)
	for y := 0; y < dstH; y++ {
		srcY := y * srcH / dstH
		for x := 0; x < dstW; x++ {
			srcX := x * srcW / dstW
			srcIdx := (srcY*srcW + srcX) * 3
			dstIdx := (y*dstW + x) * 3
			dst[dstIdx] = srcRGB[srcIdx]
			dst[dstIdx+1] = srcRGB[srcIdx+1]
			dst[dstIdx+2] = srcRGB[srcIdx+2]
		}
	}
	return dst
}

func main() {
	framesDir := flag.String("framesDir", "", "Path to directory containing frame images")
	thresholdModeFlag := flag.Int("threshold", 0, "Threshold mode for processing (0 to auto-detect from settings.json)")
	settingsFileFlag := flag.String("settings", "settings.json", "Path to settings.json file")
	flag.Parse()

	if *framesDir == "" {
		fmt.Fprintf(os.Stderr, "Error: -framesDir is required\n")
		os.Exit(1)
	}

	thresholdMode := *thresholdModeFlag
	if thresholdMode == 0 {
		thresholdMode = loadThresholdFromSettings(*settingsFileFlag)
	}

	files, err := os.ReadDir(*framesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading frames directory: %v\n", err)
		os.Exit(1)
	}

	var imageFiles []string
	for _, f := range files {
		if !f.IsDir() && (strings.HasSuffix(strings.ToLower(f.Name()), ".jpg") || strings.HasSuffix(strings.ToLower(f.Name()), ".jpeg")) {
			imageFiles = append(imageFiles, f.Name())
		}
	}

	sort.Strings(imageFiles)

	const procW = 426
	const procH = 240
	proc := logic.NewProcessor(procW, procH, 3, 0.005)

	var results []FramePreAnnotation

	for _, fileName := range imageFiles {
		filePath := filepath.Join(*framesDir, fileName)
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to open %s: %v\n", fileName, err)
			continue
		}

		img, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to decode %s: %v\n", fileName, err)
			continue
		}

		bounds := img.Bounds()
		origWidth := bounds.Dx()
		origHeight := bounds.Dy()

		// Convert image.Image to RGB24 byte slice
		rgbFrame := make([]byte, origWidth*origHeight*3)
		for y := 0; y < origHeight; y++ {
			for x := 0; x < origWidth; x++ {
				r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				idx := (y*origWidth + x) * 3
				rgbFrame[idx] = byte(r >> 8)
				rgbFrame[idx+1] = byte(g >> 8)
				rgbFrame[idx+2] = byte(b >> 8)
			}
		}

		// Downsample to 240p processing resolution matching main classifier program
		smallFrame := resizeTo240p(rgbFrame, origWidth, origHeight, procW, procH)

		binaryFrame := proc.ProcessWithThresholdMode(smallFrame, thresholdMode)
		_, blobs := proc.ApplyEfficientFilteringV1(binaryFrame)

		points := make([]PointResult, 0, len(blobs))
		scaleX := float64(origWidth) / float64(procW)
		scaleY := float64(origHeight) / float64(procH)

		for _, b := range blobs {
			points = append(points, PointResult{
				X: float64(b.Centroid.X) * scaleX,
				Y: float64(b.Centroid.Y) * scaleY,
			})
		}

		results = append(results, FramePreAnnotation{
			FrameName: fileName,
			Points:    points,
		})
	}

	if results == nil {
		results = []FramePreAnnotation{}
	}

	jsonBytes, err := json.Marshal(results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	os.Stdout.Write(jsonBytes)
}
