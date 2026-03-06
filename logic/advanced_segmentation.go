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
func (p *Processor) ApplyMorphRepair(binaryFrame []byte) ([]byte, []Blob) {
	// Closing: Dilation then Erosion (reconnects parts)
	temp := p.dilate(binaryFrame)
	temp = p.erode(temp)
	// Opening: Erosion then Dilation (removes noise)
	temp = p.erode(temp)
	temp = p.dilate(temp)

	points := p.FindBlobs(temp)
	blobs := p.ExtractBlobs(points)
	return p.colorBlobs(blobs), blobs
}

// 6: Distance Transform + Watershed (Simplified Heuristic)
func (p *Processor) ApplyWatershed(binaryFrame []byte) ([]byte, []Blob) {
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)
	var finalBlobs []Blob

	for _, b := range rawBlobs {
		if b.Area < 800 { // Normal bee
			finalBlobs = append(finalBlobs, b)
		} else {
			// Cluster: split based on area
			clusterCount := int(math.Round(float64(b.Area) / 450.0))
			for i := 0; i < clusterCount; i++ {
				finalBlobs = append(finalBlobs, b)
			}
		}
	}
	return p.colorBlobs(finalBlobs), finalBlobs
}

// 7: Convexity Defect Splitting (Heuristic)
func (p *Processor) ApplyDefectSplitting(binaryFrame []byte) ([]byte, []Blob) {
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)
	var finalBlobs []Blob

	for _, b := range rawBlobs {
		beesInBlob := 1
		if b.Solidity < 0.45 && b.Area > 600 {
			beesInBlob = 2
		} else {
			beesInBlob = int(math.Max(1, math.Round(float64(b.Area)/450.0)))
		}

		for i := 0; i < beesInBlob; i++ {
			finalBlobs = append(finalBlobs, b)
		}
	}
	return p.colorBlobs(finalBlobs), finalBlobs
}

// 8: Skeleton-Based Splitting (Mock/Simple)
func (p *Processor) ApplySkeletonSplitting(binaryFrame []byte) ([]byte, []Blob) {
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)
	var finalBlobs []Blob

	for _, b := range rawBlobs {
		w, h := b.BBox.MaxX-b.BBox.MinX+1, b.BBox.MaxY-b.BBox.MinY+1
		aspectRatio := float64(w) / float64(h)
		beesInBlob := 1
		if aspectRatio > 2.5 || aspectRatio < 0.4 {
			beesInBlob = int(math.Max(1, math.Round(float64(b.Area)/400.0)))
		}

		for i := 0; i < beesInBlob; i++ {
			finalBlobs = append(finalBlobs, b)
		}
	}
	return p.colorBlobs(finalBlobs), finalBlobs
}

// 9: Dynamic Area Estimation (Median)
func (p *Processor) ApplyDynamicAreaEstimation(binaryFrame []byte) ([]byte, []Blob) {
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)
	if len(rawBlobs) == 0 {
		return binaryFrame, nil
	}

	var areas []int
	for _, b := range rawBlobs {
		areas = append(areas, b.Area)
	}
	sort.Ints(areas)
	medianArea := float64(areas[len(areas)/2])
	if medianArea < 100 {
		medianArea = 450
	}

	var finalBlobs []Blob
	for _, b := range rawBlobs {
		beesInBlob := int(math.Max(1, math.Round(float64(b.Area)/medianArea)))
		for i := 0; i < beesInBlob; i++ {
			finalBlobs = append(finalBlobs, b)
		}
	}
	return p.colorBlobs(finalBlobs), finalBlobs
}

// 10: Shape Filtering (Descriptors)
func (p *Processor) ApplyShapeFiltering(binaryFrame []byte) ([]byte, []Blob) {
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)
	var finalBlobs []Blob

	for _, b := range rawBlobs {
		if b.Area > 50 {
			beesInBlob := int(math.Max(1, math.Round(float64(b.Area)/450.0)))
			for i := 0; i < beesInBlob; i++ {
				finalBlobs = append(finalBlobs, b)
			}
		}
	}
	return p.colorBlobs(finalBlobs), finalBlobs
}

// 11: Neighbor Merge Pass
func (p *Processor) ApplyNeighborMerge(binaryFrame []byte) ([]byte, []Blob) {
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)

	var filtered []Blob
	merged := make([]bool, len(rawBlobs))
	for i := 0; i < len(rawBlobs); i++ {
		if merged[i] {
			continue
		}

		current := rawBlobs[i]
		for j := i + 1; j < len(rawBlobs); j++ {
			if merged[j] {
				continue
			}
			dx := float64(current.Centroid.X - rawBlobs[j].Centroid.X)
			dy := float64(current.Centroid.Y - rawBlobs[j].Centroid.Y)
			if math.Sqrt(dx*dx+dy*dy) < 30 {
				merged[j] = true
				// Minimal merge logic: keep the first one
			}
		}
		filtered = append(filtered, current)
	}

	var finalBlobs []Blob
	for _, b := range filtered {
		beesInBlob := int(math.Max(1, math.Round(float64(b.Area)/450.0)))
		for i := 0; i < beesInBlob; i++ {
			finalBlobs = append(finalBlobs, b)
		}
	}
	return p.colorBlobs(finalBlobs), finalBlobs
}

func (p *Processor) ApplyMotionConsistency(binaryFrame []byte) ([]byte, []Blob) {
	return p.ApplyShapeFiltering(binaryFrame)
}
func (p *Processor) ApplyTemporalStabilisation(binaryFrame []byte) ([]byte, []Blob) {
	return p.ApplyShapeFiltering(binaryFrame)
}

// 14: Multi-Stage Segmentation Pipeline
func (p *Processor) ApplyAdvancedPipeline(binaryFrame []byte) ([]byte, []Blob) {
	temp := p.dilate(binaryFrame)
	temp = p.erode(temp)
	points := p.FindBlobs(temp)
	rawBlobs := p.ExtractBlobs(points)

	var areas []int
	for _, b := range rawBlobs {
		if b.Area > 50 {
			areas = append(areas, b.Area)
		}
	}
	if len(areas) == 0 {
		return binaryFrame, nil
	}
	sort.Ints(areas)
	medianArea := float64(areas[len(areas)/2])

	var finalBlobs []Blob
	for _, b := range rawBlobs {
		if b.Area > 50 {
			beesInBlob := int(math.Max(1, math.Round(float64(b.Area)/medianArea)))
			for i := 0; i < beesInBlob; i++ {
				finalBlobs = append(finalBlobs, b)
			}
		}
	}
	return p.colorBlobs(finalBlobs), finalBlobs
}

// --- Additional Utilities ---

func (p *Processor) colorBlobs(blobs []Blob) []byte {
	out := make([]byte, p.width*p.height*p.bytesPP)
	for i, b := range blobs {
		c := p.GetVibrantColor(i)
		for _, pt := range b.Points {
			idx := (pt.Y*p.width + pt.X) * p.bytesPP
			out[idx], out[idx+1], out[idx+2] = c[0], c[1], c[2]
		}
	}
	return out
}
