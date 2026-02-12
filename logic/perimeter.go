package logic

import (
	"math"
)

// ApplyPerimeterArea analyzes blobs by comparing their perimeter to their area.
func (p *Processor) ApplyPerimeterArea(binaryFrame []byte) ([]byte, int) {
	blobs := p.FindBlobs(binaryFrame)
	count := 0

	outputFrame := make([]byte, len(binaryFrame))

	for i, blob := range blobs {
		area := float64(len(blob))
		perimeter := p.calculatePerimeter(blob)

		// Isoperimetric quotient (circularity) proxy: Perimeter^2 / Area
		// For a circle, it's 4*PI (~12.57).
		// For an oval bee, it might be 15-20.
		ratio := (perimeter * perimeter) / area

		beesInBlob := 1
		if area > 500 {
			if ratio > 25 {
				// High perimeter relative to area => complex shape, likely multiple bees
				beesInBlob = int(math.Round(area / 400.0))
			} else {
				beesInBlob = int(math.Round(area / 450.0))
			}
		}
		if beesInBlob < 1 {
			beesInBlob = 1
		}
		count += beesInBlob

		// Color with a unique vibrant color
		color := p.GetVibrantColor(i)

		for _, pt := range blob {
			pIdx := (pt.Y*p.width + pt.X) * p.bytesPP
			outputFrame[pIdx] = color[0]
			outputFrame[pIdx+1] = color[1]
			outputFrame[pIdx+2] = color[2]
		}
	}

	return outputFrame, count
}

// calculatePerimeter estimates the perimeter of a blob by counting boundary pixels
func (p *Processor) calculatePerimeter(blob []Point) float64 {
	// Simple approximation: a pixel is on the boundary if it has < 4 neighbors in the blob
	pixelSet := make(map[Point]bool)
	for _, pt := range blob {
		pixelSet[pt] = true
	}

	perimeter := 0.0
	for _, pt := range blob {
		neighbors := []Point{
			{X: pt.X, Y: pt.Y - 1},
			{X: pt.X, Y: pt.Y + 1},
			{X: pt.X - 1, Y: pt.Y},
			{X: pt.X + 1, Y: pt.Y},
		}

		isBoundary := false
		for _, n := range neighbors {
			if !pixelSet[n] {
				isBoundary = true
				break
			}
		}
		if isBoundary {
			perimeter += 1.0
		}
	}
	return perimeter
}
