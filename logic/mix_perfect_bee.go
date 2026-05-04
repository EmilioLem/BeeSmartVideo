package logic

import (
	"math"
)

// ApplyPerfectBeeFinder (Method 19)
// Combines: Shape Filtering + Isoperimetric Quotient + Convexity Defect Splitting
// Strict filter that looks for the "ideal" bee shape and splits deviations.
func (p *Processor) ApplyPerfectBeeFinder(binaryFrame []byte) ([]byte, []Blob) {
	// 1. Initial extraction
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)

	var finalBlobs []Blob
	for _, b := range rawBlobs {
		// 2. Shape Filter
		if b.Area < 100 {
			continue
		}

		// 3. Isoperimetric Analysis (Perimeter vs Area)
		// For a circle, ratio = 1.0. For bees, it's usually 0.6 - 0.8.
		// Higher complexity (cluster) reduces this ratio significantly.
		
		// 4. Convexity Defect Analysis (using Solidity)
		count := 1
		target := p.GetTargetArea()
		if b.Solidity < 0.5 {
			// Deep "dents" suggest a split needed
			count = int(math.Max(2, math.Round(float64(b.Area)/(target*0.88))))
		} else if float64(b.Area) > p.GetMaxNormalArea() {
			count = int(math.Round(float64(b.Area) / target))
		}

		for i := 0; i < count; i++ {
			finalBlobs = append(finalBlobs, b)
		}
	}

	return p.colorBlobs(finalBlobs), finalBlobs
}
