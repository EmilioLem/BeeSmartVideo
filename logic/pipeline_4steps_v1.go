package logic

import (
	"math"
)

// ApplyFourStepPipeline implements the "4 steps v1" method.
func (p *Processor) ApplyFourStepPipeline(binaryFrame []byte) ([]byte, []Blob) {
	params := p.FourStepParams

	// --- Step 1: Preprocessing ---
	// Morphological Repair (Custom Iterations)
	temp := binaryFrame
	for i := 0; i < params.MorphRepairIterations; i++ {
		temp = p.dilate(temp)
		temp = p.erode(temp)
		temp = p.erode(temp)
		temp = p.dilate(temp)
	}

	// K-Means Pre-processing (Mock: if K > 1, we could refine, but usually used for segmentation)
	// For now, we follow the user's description of using it to "define raw candidate areas".
	// Since K-Means usually outputs blobs, we'll use it if K > 1.
	var preBlobs []Blob
	if params.KMeansK > 1 {
		_, preBlobs = p.ApplyKMeans(temp, params.KMeansK)
	} else {
		points := p.FindBlobs(temp)
		preBlobs = p.ExtractBlobs(points)
	}

	// --- Step 2: Core Segmentation ---
	// Shape Filtering
	var filteredBlobs []Blob
	for _, b := range preBlobs {
		if b.Area >= params.ShapeFilterMinArea {
			filteredBlobs = append(filteredBlobs, b)
		}
	}

	// --- Step 3: Splitting ---
	// We combine multiple splitting techniques
	var splitBlobs []Blob
	for _, b := range filteredBlobs {
		beesInBlob := 1

		// 1. Watershed/Area-based check
		if float64(b.Area) > params.MaxNormalArea {
			beesInBlob = int(math.Max(float64(beesInBlob), math.Round(float64(b.Area)/params.TargetArea)))
		}

		// 2. Convexity Defect (Solidity check)
		if b.Solidity < params.SolidityThreshold {
			beesInBlob = int(math.Max(float64(beesInBlob), 2)) // At least 2 if low solidity
		}

		// 3. Skeleton-Based (Aspect Ratio check)
		w, h := b.BBox.MaxX-b.BBox.MinX+1, b.BBox.MaxY-b.BBox.MinY+1
		aspectRatio := float64(w) / float64(h)
		if aspectRatio > params.AspectRatioThreshold || aspectRatio < 1.0/params.AspectRatioThreshold {
			beesInBlob = int(math.Max(float64(beesInBlob), 2))
		}

		// Apply splitting by duplicating metadata (simplified heuristic)
		for i := 0; i < beesInBlob; i++ {
			splitBlobs = append(splitBlobs, b)
		}
	}

	// --- Step 4: Validation ---
	// Neighbor Merge
	var validBlobs []Blob
	merged := make([]bool, len(splitBlobs))
	for i := 0; i < len(splitBlobs); i++ {
		if merged[i] {
			continue
		}

		current := splitBlobs[i]
		for j := i + 1; j < len(splitBlobs); j++ {
			if merged[j] {
				continue
			}
			dx := float64(current.Centroid.X - splitBlobs[j].Centroid.X)
			dy := float64(current.Centroid.Y - splitBlobs[j].Centroid.Y)
			if math.Sqrt(dx*dx+dy*dy) < params.MergeDistance {
				merged[j] = true
			}
		}
		validBlobs = append(validBlobs, current)
	}

	// Motion and Temporal (Placeholder for now, could use params.TemporalStabilityWeight)
	
	return p.colorBlobs(validBlobs), validBlobs
}
