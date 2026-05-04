package logic

import (
	"math"
)

// ApplyPerimeterArea analyzes blobs by comparing their perimeter to their area.
func (p *Processor) ApplyPerimeterArea(binaryFrame []byte) ([]byte, []Blob) {
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)
	var finalBlobs []Blob

	outputFrame := make([]byte, len(binaryFrame))

	for i, b := range rawBlobs {
		perimeter := p.calculatePerimeter(b.Points)
		// Isoperimetric quotient (circularity) proxy: Perimeter^2 / Area
		ratio := (perimeter * perimeter) / float64(b.Area)
		b.Ratio = ratio

		beesInBlob := 1
		target := p.GetTargetArea()
		if float64(b.Area) > target*1.1 {
			if ratio > 25 {
				beesInBlob = int(math.Round(float64(b.Area) / (target * 0.88)))
			} else {
				beesInBlob = int(math.Round(float64(b.Area) / target))
			}
		}
		if beesInBlob < 1 {
			beesInBlob = 1
		}

		for j := 0; j < beesInBlob; j++ {
			finalBlobs = append(finalBlobs, b)
		}

		// Color with a unique vibrant color
		color := p.GetVibrantColor(i)

		for _, pt := range b.Points {
			pIdx := (pt.Y*p.width + pt.X) * p.bytesPP
			outputFrame[pIdx] = color[0]
			outputFrame[pIdx+1] = color[1]
			outputFrame[pIdx+2] = color[2]
		}
	}

	return outputFrame, finalBlobs
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
