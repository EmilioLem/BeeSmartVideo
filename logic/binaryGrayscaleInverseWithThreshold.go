package logic

// BinaryGrayscaleInverseWithThreshold converts a color frame to binary grayscale and inverts it
// Each colored pixel is converted to grayscale, then to binary (white/black) based on threshold,
// then inverted (white becomes black, black becomes white)
func (p *Processor) BinaryGrayscaleInverseWithThreshold(frame []byte, threshold uint8) []byte {
	binaryFrame := make([]byte, len(frame))

	for i := 0; i < len(frame); i += p.bytesPP {
		// Extract RGB values
		r := frame[i]
		g := frame[i+1]
		b := frame[i+2]

		// Convert to grayscale using luminosity method
		// Formula: 0.299*R + 0.587*G + 0.114*B
		gray := uint8(float64(r)*0.299 + float64(g)*0.587 + float64(b)*0.114)

		// Convert to binary based on threshold
		var binaryValue uint8
		if gray >= threshold {
			binaryValue = 255 // White
		} else {
			binaryValue = 0 // Black
		}

		// Invert the binary value
		inverted := 255 - binaryValue

		// Set RGB to the inverted binary value
		binaryFrame[i] = inverted
		binaryFrame[i+1] = inverted
		binaryFrame[i+2] = inverted
	}

	// Store original frame for later use
	p.originalFrame = make([]byte, len(frame))
	copy(p.originalFrame, frame)

	// Count white pixels in the binary inverted frame
	p.lastWhitePixelCount = p.countWhitePixels(binaryFrame)

	return binaryFrame
}

// countWhitePixels counts the number of white pixels in a frame
func (p *Processor) countWhitePixels(frame []byte) int {
	count := 0
	for i := 0; i < len(frame); i += p.bytesPP {
		if frame[i] == 255 && frame[i+1] == 255 && frame[i+2] == 255 {
			count++
		}
	}
	return count
}

// GetLastWhitePixelCount returns the white pixel count from the last processing
func (p *Processor) GetLastWhitePixelCount() int {
	return p.lastWhitePixelCount
}
