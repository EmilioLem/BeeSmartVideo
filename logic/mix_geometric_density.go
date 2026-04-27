package logic

import (
	"math"
	"sort"
)

// ApplyGeometricDensityFilter (Method 16)
// Combines: Shape Filtering + Neighbor Merge + Dynamic Area Estimation
// Ideal for high-density environments where bees are very close but not necessarily touching.
func (p *Processor) ApplyGeometricDensityFilter(binaryFrame []byte) ([]byte, []Blob) {
	// 1. Initial Blob Extraction
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)

	// 2. Shape Filtering: Remove tiny noise
	var filtered []Blob
	for _, b := range rawBlobs {
		if b.Area > 40 {
			filtered = append(filtered, b)
		}
	}

	// 3. Neighbor Merge: Join nearby blobs that are part of the same object
	var merged []Blob
	isMerged := make([]bool, len(filtered))
	for i := 0; i < len(filtered); i++ {
		if isMerged[i] {
			continue
		}
		current := filtered[i]
		for j := i + 1; j < len(filtered); j++ {
			if isMerged[j] {
				continue
			}
			dx := float64(current.Centroid.X - filtered[j].Centroid.X)
			dy := float64(current.Centroid.Y - filtered[j].Centroid.Y)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 25 { // Proximity threshold
				isMerged[j] = true
				// Note: Simplified merge, just keeping the first one for counting
			}
		}
		merged = append(merged, current)
	}

	if len(merged) == 0 {
		return p.colorBlobs(nil), nil
	}

	// 4. Dynamic Area Estimation on merged blobs
	var areas []int
	for _, b := range merged {
		areas = append(areas, b.Area)
	}
	sort.Ints(areas)
	medianArea := float64(areas[len(areas)/2])
	if medianArea < 300 {
		medianArea = 450
	}

	var finalBlobs []Blob
	for _, b := range merged {
		beesInBlob := int(math.Max(1, math.Round(float64(b.Area)/medianArea)))
		for i := 0; i < beesInBlob; i++ {
			finalBlobs = append(finalBlobs, b)
		}
	}

	return p.colorBlobs(finalBlobs), finalBlobs
}
