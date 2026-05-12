package logic

import (
	"math"
)

// ApplyEfficientFilteringV1 implements the "Efficient filtering v1" processing method.
func (p *Processor) ApplyEfficientFilteringV1(binaryFrame []byte) ([]byte, []Blob) {
	// --- Settings (Tweakable in code) ---
	
	// Morphological Repair Settings
	morphPasses := 2 // Number of passes for erosion/dilation
	// Note: Kernel size is currently fixed at 3x3 in the implementation below.
	// To change kernel size, the loop bounds would need adjustment.
	
	// Splitting/Merging Settings
	divergenceThreshold := 0.40 // 40% divergence from average size
	closenessThreshold := 30.0  // Distance threshold for merging close blobs
	
	// -------------------------------------

	// Step 1: Binary thresholding is already done and passed as binaryFrame.
	// We need a copy because we might modify it or need the original for cropping.
	workingFrame := make([]byte, len(binaryFrame))
	copy(workingFrame, binaryFrame)

	// Step 2: Morphological Repair (Erosion-Dilation cycles)
	// We do 'morphPasses' of dilation followed by erosion (Closing)
	// and then erosion followed by dilation (Opening) to clean noise.
	
	for i := 0; i < morphPasses; i++ {
		workingFrame = p.customDilate(workingFrame)
	}
	for i := 0; i < morphPasses; i++ {
		workingFrame = p.customErode(workingFrame)
	}
	for i := 0; i < morphPasses; i++ {
		workingFrame = p.customErode(workingFrame)
	}
	for i := 0; i < morphPasses; i++ {
		workingFrame = p.customDilate(workingFrame)
	}

	// Step 3: Connected Component Analysis
	pointGroups := p.FindBlobs(workingFrame)
	rawBlobs := p.ExtractBlobs(pointGroups)

	if len(rawBlobs) == 0 {
		return workingFrame, nil
	}

	// Calculate average size of detected blobs (amount of pixels per blob)
	// We filter out very small blobs to avoid skewing the average with noise.
	var totalArea int
	var count int
	for _, b := range rawBlobs {
		if b.Area > 20 { // Noise threshold
			totalArea += b.Area
			count++
		}
	}
	
	averageSize := 450.0 // Default fallback if no valid blobs
	if count > 0 {
		averageSize = float64(totalArea) / float64(count)
	}

	// Step 4: Splitting/Merging
	var finalBlobs []Blob
	mergedFlags := make([]bool, len(rawBlobs))

	for i := 0; i < len(rawBlobs); i++ {
		if mergedFlags[i] {
			continue
		}

		b := rawBlobs[i]
		
		// Ignore very small noise
		if b.Area < 10 {
			continue
		}

		// Check for merging (if too small)
		if float64(b.Area) < averageSize*(1.0-divergenceThreshold) {
			// Look for close neighbors to merge
			mergedPoints := b.Points
			mergedFlags[i] = true
			
			for j := i + 1; j < len(rawBlobs); j++ {
				if mergedFlags[j] {
					continue
				}
				nb := rawBlobs[j]
				
				// Calculate distance between centroids
				dx := float64(b.Centroid.X - nb.Centroid.X)
				dy := float64(b.Centroid.Y - nb.Centroid.Y)
				dist := math.Sqrt(dx*dx + dy*dy)
				
				if dist < closenessThreshold {
					mergedPoints = append(mergedPoints, nb.Points...)
					mergedFlags[j] = true
				}
			}
			
			// Create a new merged blob
			newBlobs := p.ExtractBlobs([][]Point{mergedPoints})
			if len(newBlobs) > 0 {
				finalBlobs = append(finalBlobs, newBlobs[0])
			}
			continue
		}

		// Check for splitting (if too big)
		if float64(b.Area) > averageSize*(1.0+divergenceThreshold) {
			// Calculate expected number of bees
			k := int(math.Round(float64(b.Area) / averageSize))
			if k <= 1 {
				k = 2 // At least split in 2 if it's large enough to trigger
			}
			
			// Run K-means on the points of this blob
			splitBlobs := p.splitBlobKMeans(b, k)
			finalBlobs = append(finalBlobs, splitBlobs...)
			mergedFlags[i] = true
			continue
		}

		// If it's within the normal range, keep it
		finalBlobs = append(finalBlobs, b)
		mergedFlags[i] = true
	}

	// Create output frame with colored blobs
	outputFrame := make([]byte, len(binaryFrame))
	for i, blob := range finalBlobs {
		color := p.GetVibrantColor(i)
		for _, point := range blob.Points {
			idx := (point.Y*p.width + point.X) * p.bytesPP
			outputFrame[idx] = color[0]
			outputFrame[idx+1] = color[1]
			outputFrame[idx+2] = color[2]
		}
	}

	return outputFrame, finalBlobs
}

// customErode performs a 3x3 erosion (specific to this method if needed, or reused)
func (p *Processor) customErode(frame []byte) []byte {
	out := make([]byte, len(frame))
	for y := 1; y < p.height-1; y++ {
		for x := 1; x < p.width-1; x++ {
			idx := (y*p.width + x) * p.bytesPP
			if frame[idx] == 255 {
				// Check 4-neighbors
				if frame[((y-1)*p.width+x)*p.bytesPP] == 0 ||
					frame[((y+1)*p.width+x)*p.bytesPP] == 0 ||
					frame[(y*p.width+(x-1))*p.bytesPP] == 0 ||
					frame[(y*p.width+(x+1))*p.bytesPP] == 0 {
					// Set to black
					out[idx] = 0
					out[idx+1] = 0
					out[idx+2] = 0
				} else {
					out[idx] = 255
					out[idx+1] = 255
					out[idx+2] = 255
				}
			}
		}
	}
	return out
}

// customDilate performs a 3x3 dilation
func (p *Processor) customDilate(frame []byte) []byte {
	out := make([]byte, len(frame))
	for y := 1; y < p.height-1; y++ {
		for x := 1; x < p.width-1; x++ {
			idx := (y*p.width + x) * p.bytesPP
			if frame[((y-1)*p.width+x)*p.bytesPP] == 255 ||
				frame[((y+1)*p.width+x)*p.bytesPP] == 255 ||
				frame[(y*p.width+(x-1))*p.bytesPP] == 255 ||
				frame[(y*p.width+(x+1))*p.bytesPP] == 255 {
				out[idx] = 255
				out[idx+1] = 255
				out[idx+2] = 255
			}
		}
	}
	return out
}

// splitBlobKMeans splits a blob into K smaller blobs using K-means on its pixels
func (p *Processor) splitBlobKMeans(blob Blob, k int) []Blob {
	if len(blob.Points) < k {
		return []Blob{blob}
	}

	// Perform k-means on the points
	assignments := p.kMeansClustering(blob.Points, k)

	// Group points by cluster
	clusterPoints := make([][]Point, k)
	for i, point := range blob.Points {
		clusterID := assignments[i]
		clusterPoints[clusterID] = append(clusterPoints[clusterID], point)
	}

	// Extract blobs from cluster points
	return p.ExtractBlobs(clusterPoints)
}
