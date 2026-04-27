package logic

import (
	"math"
	"sort"
)

// ApplyHybridConvexPipeline (Method 15)
// Combines: Morphological Repair + Convex Hull Analysis + Dynamic Area Estimation
// Focuses on cleaning noise and then using both shape quality and area for count estimation.
func (p *Processor) ApplyHybridConvexPipeline(binaryFrame []byte) ([]byte, []Blob) {
	// 1. Morphological Repair (Closing: Dilation then Erosion)
	temp := p.dilate(binaryFrame)
	temp = p.erode(temp)

	// 2. Find and Extract Blobs
	points := p.FindBlobs(temp)
	rawBlobs := p.ExtractBlobs(points)

	if len(rawBlobs) == 0 {
		return binaryFrame, nil
	}

	// 3. Dynamic Median Area Calculation for baseline
	var areas []int
	for _, b := range rawBlobs {
		if b.Area > 50 {
			areas = append(areas, b.Area)
		}
	}
	if len(areas) == 0 {
		return p.colorBlobs(nil), nil
	}
	sort.Ints(areas)
	medianArea := float64(areas[len(areas)/2])

	var finalBlobs []Blob
	for _, b := range rawBlobs {
		if b.Area < 50 {
			continue
		}

		// 4. Hybrid Estimation: Adjust count based on Solidity (Convex Hull Analysis)
		beesInBlob := 1
		// If blob is much larger than median, it's definitely a cluster
		if float64(b.Area) > medianArea*1.2 {
			// If it has low solidity, it's likely high-count cluster
			if b.Solidity < 0.6 {
				beesInBlob = int(math.Max(1, math.Round(float64(b.Area)/(medianArea*0.85))))
			} else {
				beesInBlob = int(math.Max(1, math.Round(float64(b.Area)/medianArea)))
			}
		}

		for i := 0; i < beesInBlob; i++ {
			finalBlobs = append(finalBlobs, b)
		}
	}

	return p.colorBlobs(finalBlobs), finalBlobs
}
