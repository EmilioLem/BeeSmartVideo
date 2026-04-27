package logic

import (
	"math"
)

// ApplyStructuralSkeletonWatershed (Method 17)
// Combines: Morphological Repair + Skeleton-Based Splitting + Watershed Heuristic
// Designed for complex clusters that can be both elongated and clumped.
func (p *Processor) ApplyStructuralSkeletonWatershed(binaryFrame []byte) ([]byte, []Blob) {
	// 1. Repair fragmented parts
	temp := p.dilate(binaryFrame)
	temp = p.erode(temp)

	// 2. Extract Blobs
	points := p.FindBlobs(temp)
	rawBlobs := p.ExtractBlobs(points)

	var finalBlobs []Blob
	for _, b := range rawBlobs {
		if b.Area < 40 {
			continue
		}

		// 3. Skeleton Aspect Analysis
		w, h := b.BBox.MaxX-b.BBox.MinX+1, b.BBox.MaxY-b.BBox.MinY+1
		aspectRatio := float64(w) / float64(h)
		
		beesBySkeleton := 1
		if aspectRatio > 2.2 || aspectRatio < 0.45 {
			beesBySkeleton = int(math.Max(1, math.Round(float64(b.Area)/400.0)))
		}

		// 4. Watershed-style Area Analysis (Heuristic override)
		beesByArea := int(math.Max(1, math.Round(float64(b.Area)/450.0)))

		// Take the maximum of both splitting heuristics
		count := beesBySkeleton
		if beesByArea > count {
			count = beesByArea
		}

		for i := 0; i < count; i++ {
			finalBlobs = append(finalBlobs, b)
		}
	}

	return p.colorBlobs(finalBlobs), finalBlobs
}
