package logic

import (
	"math"
	"sort"
)

// --- Helper Functions ---

// erode performs a 3x3 erosion
func (p *Processor) erode(binaryFrame []byte) []byte {
	eroded := make([]byte, len(binaryFrame))
	for y := 1; y < p.height-1; y++ {
		for x := 1; x < p.width-1; x++ {
			idx := (y*p.width + x) * p.bytesPP
			if binaryFrame[idx] == 255 {
				if binaryFrame[((y-1)*p.width+x)*p.bytesPP] == 0 ||
					binaryFrame[((y+1)*p.width+x)*p.bytesPP] == 0 ||
					binaryFrame[(y*p.width+(x-1))*p.bytesPP] == 0 ||
					binaryFrame[(y*p.width+(x+1))*p.bytesPP] == 0 {
					eroded[idx] = 0
					eroded[idx+1] = 0
					eroded[idx+2] = 0
				} else {
					eroded[idx] = 255
					eroded[idx+1] = 255
					eroded[idx+2] = 255
				}
			}
		}
	}
	return eroded
}

// dilate performs a 3x3 dilation
func (p *Processor) dilate(binaryFrame []byte) []byte {
	dilated := make([]byte, len(binaryFrame))
	for y := 1; y < p.height-1; y++ {
		for x := 1; x < p.width-1; x++ {
			idx := (y*p.width + x) * p.bytesPP
			if binaryFrame[((y-1)*p.width+x)*p.bytesPP] == 255 ||
				binaryFrame[((y+1)*p.width+x)*p.bytesPP] == 255 ||
				binaryFrame[(y*p.width+(x-1))*p.bytesPP] == 255 ||
				binaryFrame[(y*p.width+(x+1))*p.bytesPP] == 255 {
				dilated[idx] = 255
				dilated[idx+1] = 255
				dilated[idx+2] = 255
			}
		}
	}
	return dilated
}

// --- Implementation of the 10 Improved Methods ---

// 5: Morphological Repair (Close/Fill/Open)
func (p *Processor) ApplyMorphRepair(binaryFrame []byte) ([]byte, int) {
	// Closing: Dilation then Erosion (reconnects parts)
	temp := p.dilate(binaryFrame)
	temp = p.erode(temp)
	// Opening: Erosion then Dilation (removes noise)
	temp = p.erode(temp)
	temp = p.dilate(temp)

	blobs := p.FindBlobs(temp)
	return p.colorBlobs(blobs), len(blobs)
}

// 6: Distance Transform + Watershed (Simplified Heuristic)
func (p *Processor) ApplyWatershed(binaryFrame []byte) ([]byte, int) {
	blobs := p.FindBlobs(binaryFrame)
	count := 0
	var finalBlobs [][]Point

	for _, blob := range blobs {
		if len(blob) < 800 { // Normal bee
			count++
			finalBlobs = append(finalBlobs, blob)
		} else {
			// Cluster: split based on area as a proxy for watershed complexity
			clusterCount := int(math.Round(float64(len(blob)) / 450.0))
			count += clusterCount
			// Just add the blob once to visualization
			finalBlobs = append(finalBlobs, blob)
		}
	}
	return p.colorBlobs(finalBlobs), count
}

// 7: Convexity Defect Splitting (Heuristic)
func (p *Processor) ApplyDefectSplitting(binaryFrame []byte) ([]byte, int) {
	blobs := p.FindBlobs(binaryFrame)
	count := 0
	for _, blob := range blobs {
		// Calculate bounding box and area
		minX, minY, maxX, maxY := p.getBoundingBox(blob)
		bboxArea := (maxX - minX + 1) * (maxY - minY + 1)
		solidity := float64(len(blob)) / float64(bboxArea)

		if solidity < 0.45 && len(blob) > 600 {
			count += 2 // High probability of two bees touching at an angle
		} else {
			count += int(math.Max(1, math.Round(float64(len(blob))/450.0)))
		}
	}
	return p.colorBlobs(blobs), count
}

// 8: Skeleton-Based Splitting (Mock/Simple)
func (p *Processor) ApplySkeletonSplitting(binaryFrame []byte) ([]byte, int) {
	// Thinning/Skeletonization is very heavy, using a proxy:
	// A long, branched blob is counted as multiple
	blobs := p.FindBlobs(binaryFrame)
	count := 0
	for _, blob := range blobs {
		minX, minY, maxX, maxY := p.getBoundingBox(blob)
		w, h := maxX-minX+1, maxY-minY+1
		aspectRatio := float64(w) / float64(h)
		if aspectRatio > 2.5 || aspectRatio < 0.4 {
			count += int(math.Max(1, math.Round(float64(len(blob))/400.0)))
		} else {
			count += 1
		}
	}
	return p.colorBlobs(blobs), count
}

// 9: Dynamic Area Estimation (Median)
func (p *Processor) ApplyDynamicAreaEstimation(binaryFrame []byte) ([]byte, int) {
	blobs := p.FindBlobs(binaryFrame)
	if len(blobs) == 0 {
		return binaryFrame, 0
	}

	var areas []int
	for _, b := range blobs {
		areas = append(areas, len(b))
	}
	sort.Ints(areas)
	medianArea := float64(areas[len(areas)/2])
	if medianArea < 100 {
		medianArea = 450
	} // Fallback

	count := 0
	for _, b := range blobs {
		count += int(math.Max(1, math.Round(float64(len(b))/medianArea)))
	}
	return p.colorBlobs(blobs), count
}

// 10: Shape Filtering (Descriptors)
func (p *Processor) ApplyShapeFiltering(binaryFrame []byte) ([]byte, int) {
	blobs := p.FindBlobs(binaryFrame)
	var filteredBlobs [][]Point
	count := 0
	for _, b := range blobs {
		if len(b) > 50 { // Minimal area filter
			filteredBlobs = append(filteredBlobs, b)
			count += int(math.Max(1, math.Round(float64(len(b))/450.0)))
		}
	}
	return p.colorBlobs(filteredBlobs), count
}

// 11: Neighbor Merge Pass
func (p *Processor) ApplyNeighborMerge(binaryFrame []byte) ([]byte, int) {
	blobs := p.FindBlobs(binaryFrame)
	// Simplified: if two centroids are closer than 20px, count them once
	type Centroid struct {
		X, Y float64
		Area int
	}
	var centroids []Centroid
	for _, b := range blobs {
		var sx, sy int
		for _, pt := range b {
			sx += pt.X
			sy += pt.Y
		}
		centroids = append(centroids, Centroid{float64(sx) / float64(len(b)), float64(sy) / float64(len(b)), len(b)})
	}

	count := 0
	merged := make([]bool, len(centroids))
	for i := 0; i < len(centroids); i++ {
		if merged[i] {
			continue
		}
		count++
		for j := i + 1; j < len(centroids); j++ {
			dx, dy := centroids[i].X-centroids[j].X, centroids[i].Y-centroids[j].Y
			if math.Sqrt(dx*dx+dy*dy) < 30 {
				merged[j] = true
			}
		}
	}
	return p.colorBlobs(blobs), count
}

// 12-13: Temporal/Motion (Mocking as simple for now since tracking requires state across calls)
func (p *Processor) ApplyMotionConsistency(binaryFrame []byte) ([]byte, int) {
	return p.ApplyShapeFiltering(binaryFrame)
}
func (p *Processor) ApplyTemporalStabilisation(binaryFrame []byte) ([]byte, int) {
	return p.ApplyShapeFiltering(binaryFrame)
}

// 14: Multi-Stage Segmentation Pipeline
func (p *Processor) ApplyAdvancedPipeline(binaryFrame []byte) ([]byte, int) {
	// Repaired -> Filtered -> Dynamic Area
	temp := p.dilate(binaryFrame)
	temp = p.erode(temp)
	blobs := p.FindBlobs(temp)

	var areas []int
	for _, b := range blobs {
		if len(b) > 50 {
			areas = append(areas, len(b))
		}
	}
	if len(areas) == 0 {
		return binaryFrame, 0
	}
	sort.Ints(areas)
	medianArea := float64(areas[len(areas)/2])

	count := 0
	for _, b := range blobs {
		if len(b) > 50 {
			count += int(math.Max(1, math.Round(float64(len(b))/medianArea)))
		}
	}
	return p.colorBlobs(blobs), count
}

// --- Additional Utilities ---

func (p *Processor) colorBlobs(blobs [][]Point) []byte {
	out := make([]byte, p.width*p.height*p.bytesPP)
	for i, b := range blobs {
		c := p.GetVibrantColor(i)
		for _, pt := range b {
			idx := (pt.Y*p.width + pt.X) * p.bytesPP
			out[idx], out[idx+1], out[idx+2] = c[0], c[1], c[2]
		}
	}
	return out
}

func (p *Processor) getBoundingBox(blob []Point) (int, int, int, int) {
	minX, minY, maxX, maxY := p.width, p.height, 0, 0
	for _, pt := range blob {
		if pt.X < minX {
			minX = pt.X
		}
		if pt.X > maxX {
			maxX = pt.X
		}
		if pt.Y < minY {
			minY = pt.Y
		}
		if pt.Y > maxY {
			maxY = pt.Y
		}
	}
	return minX, minY, maxX, maxY
}
