package datasetgen

import (
	"BeeSmartVideo/logic"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
)

// V1Generator handles dataset exporting of tracks into CSVs and JPG crops
type V1Generator struct {
	basePath string
	csvFile  *os.File
}

// NewV1Generator initializes a new export instance and writes standard headers
func NewV1Generator(basePath string) (*V1Generator, error) {
	// Ensure directories exist
	err := os.MkdirAll(filepath.Join(basePath, "images"), 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create dataset directories: %w", err)
	}

	csvPath := filepath.Join(basePath, "bee_data.csv")
	
	// Open or create CSV file
	fileExist := false
	if _, err := os.Stat(csvPath); err == nil {
		fileExist = true
	}

	f, err := os.OpenFile(csvPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open csv file: %w", err)
	}

	// Write header if it's a new file
	if !fileExist {
		_, err = f.WriteString("Frame,ID,X,Y,Area,Solidity,Ratio,VX,VY\n")
		if err != nil {
			return nil, fmt.Errorf("failed to write csv header: %w", err)
		}
	}

	return &V1Generator{
		basePath: basePath,
		csvFile:  f,
	}, nil
}

func (g *V1Generator) Close() {
	if g.csvFile != nil {
		g.csvFile.Close()
	}
}

func (g *V1Generator) ProcessFrameTracks(tracks []logic.Track, fullFrame []byte, inWidth, inHeight int, frameCount int) {
	for _, track := range tracks {
		if track.Status == "confirmed" {
			g.LogToCSV(track, frameCount)
			g.SaveCrop(track, frameCount, fullFrame, inWidth, inHeight)
		}
	}
}

func (g *V1Generator) LogToCSV(track logic.Track, frame int) {
	line := fmt.Sprintf("%d,%d,%d,%d,%d,%.4f,%.4f,%.4f,%.4f\n",
		frame, track.ID, track.Centroid.X, track.Centroid.Y, track.Area, track.Solidity, track.Ratio, track.VX, track.VY)
	g.csvFile.WriteString(line)
}

func (g *V1Generator) SaveCrop(track logic.Track, frame int, fullFrame []byte, inWidth, inHeight int) {
	if fullFrame == nil {
		return
	}

	bytesPP := 3 // RGB24
	cropSize := 64
	halfSize := cropSize / 2

	// Scale coordinates back to original resolution (assumes 3x downsampling)
	origX := track.Centroid.X * 3
	origY := track.Centroid.Y * 3

	// Create a new image for the crop
	img := image.NewRGBA(image.Rect(0, 0, cropSize, cropSize))

	for cy := 0; cy < cropSize; cy++ {
		for cx := 0; cx < cropSize; cx++ {
			srcX := origX - halfSize + cx
			srcY := origY - halfSize + cy

			if srcX >= 0 && srcX < inWidth && srcY >= 0 && srcY < inHeight {
				srcIdx := (srcY*inWidth + srcX) * bytesPP
				if srcIdx+2 < len(fullFrame) {
					img.SetRGBA(cx, cy, color.RGBA{
						R: fullFrame[srcIdx],
						G: fullFrame[srcIdx+1],
						B: fullFrame[srcIdx+2],
						A: 255,
					})
				}
			} else {
				// Pad out-of-bounds with black
				img.SetRGBA(cx, cy, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	// Save the crop
	filename := fmt.Sprintf("%d_%d.jpg", track.ID, frame)
	targetPath := filepath.Join(g.basePath, "images", filename)

	out, err := os.Create(targetPath)
	if err == nil {
		jpeg.Encode(out, img, &jpeg.Options{Quality: 90})
		out.Close()
	}
}
