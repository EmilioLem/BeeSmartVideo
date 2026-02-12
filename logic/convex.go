package logic

import (
	"math"
)

// ApplyConvexHull analyzes blobs based on their convexity/solidity.
func (p *Processor) ApplyConvexHull(binaryFrame []byte) ([]byte, int) {
	blobs := p.FindBlobs(binaryFrame)
	count := 0

	outputFrame := make([]byte, len(binaryFrame))

	for i, blob := range blobs {
		// Calculate Bounding Box
		minX, minY := p.width, p.height
		maxX, maxY := 0, 0
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

		width := maxX - minX + 1
		height := maxY - minY + 1
		bboxArea := width * height
		blobArea := len(blob)

		// Solidity = Area / BoundingBoxArea
		// A single bee (oval) should have a solidity around 0.6 - 0.8
		solidity := float64(blobArea) / float64(bboxArea)

		// Estimate number of bees
		// Average bee area is roughly 300-500 pixels (at 640x480 depending on distance)
		// We can also use solidity to guess if it's multiple bees
		beesInBlob := 1
		if blobArea > 600 { // Large blob
			if solidity < 0.5 {
				// Likely multiple bees because it's not "solid"
				beesInBlob = int(math.Round(float64(blobArea) / 400.0))
			} else {
				// Could be one very large bee or multiple very tight bees
				beesInBlob = int(math.Round(float64(blobArea) / 450.0))
			}
		}
		if beesInBlob < 1 {
			beesInBlob = 1
		}
		count += beesInBlob

		// Color the blob with a unique vibrant color
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
