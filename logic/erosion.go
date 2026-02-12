package logic

// ApplyErosion applies erosion to the binary frame to separate tight groups of bees
// then counts the connected components.
func (p *Processor) ApplyErosion(binaryFrame []byte) ([]byte, int) {
	// 1. Perform Erosion
	erodedFrame := make([]byte, len(binaryFrame))
	copy(erodedFrame, binaryFrame)

	// Simple 3x3 kernel erosion
	for y := 1; y < p.height-1; y++ {
		for x := 1; x < p.width-1; x++ {
			idx := (y*p.width + x) * p.bytesPP
			if binaryFrame[idx] == 255 {
				// Check 4-neighbors (simpler/faster than 8-neighbors)
				top := ((y-1)*p.width + x) * p.bytesPP
				bottom := ((y+1)*p.width + x) * p.bytesPP
				left := (y*p.width + (x - 1)) * p.bytesPP
				right := (y*p.width + (x + 1)) * p.bytesPP

				if binaryFrame[top] == 0 || binaryFrame[bottom] == 0 ||
					binaryFrame[left] == 0 || binaryFrame[right] == 0 {
					erodedFrame[idx] = 0
					erodedFrame[idx+1] = 0
					erodedFrame[idx+2] = 0
				}
			}
		}
	}

	// 2. Count Connected Components (Blobs) using shared FindBlobs
	blobs := p.FindBlobs(erodedFrame)
	count := len(blobs)

	// Create output frame with colored blobs
	outputFrame := make([]byte, len(binaryFrame))

	for i, blob := range blobs {
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
