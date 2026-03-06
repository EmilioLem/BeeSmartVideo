package logic

import (
	"math"
)

// ApplyConvexHull analyzes blobs based on their convexity/solidity.
func (p *Processor) ApplyConvexHull(binaryFrame []byte) ([]byte, []Blob) {
	points := p.FindBlobs(binaryFrame)
	rawBlobs := p.ExtractBlobs(points)
	var finalBlobs []Blob

	outputFrame := make([]byte, len(binaryFrame))

	for i, b := range rawBlobs {
		// Estimate number of bees based on solidity
		beesInBlob := 1
		if b.Area > 600 { // Large blob
			if b.Solidity < 0.5 {
				beesInBlob = int(math.Round(float64(b.Area) / 400.0))
			} else {
				beesInBlob = int(math.Round(float64(b.Area) / 450.0))
			}
		}
		if beesInBlob < 1 {
			beesInBlob = 1
		}

		// Split into virtual blobs for tracking purposes if needed
		// For now, we add the same blob N times to the list if it's a cluster
		// (A better way would be to split the points, but that's complex)
		for j := 0; j < beesInBlob; j++ {
			finalBlobs = append(finalBlobs, b)
		}

		// Color the blob with a unique vibrant color
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
