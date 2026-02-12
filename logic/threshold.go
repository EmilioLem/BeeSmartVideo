package logic

import (
	"math"
)

// AdaptiveThreshold calculates a threshold by finding two peaks in a sampled histogram
// and picking the midpoint.
func (p *Processor) AdaptiveThreshold(frame []byte) uint8 {
	histogram := make([]int, 256)

	// Sample 1 every 20 pixels (5%)
	for y := 0; y < p.height; y += 4 { // Simplified step for grid (not quite 1/20 but uniform)
		for x := 0; x < p.width; x += 5 {
			idx := (y*p.width + x) * p.bytesPP

			// Luminosity grayscale
			r := frame[idx]
			g := frame[idx+1]
			b := frame[idx+2]
			gray := uint8(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b))

			histogram[gray]++
		}
	}

	// Find two major peaks
	// We use a simple windowed peak detection
	firstPeak := 0
	for i := 1; i < 256; i++ {
		if histogram[i] > histogram[firstPeak] {
			firstPeak = i
		}
	}

	// Find second peak at least some distance away from the first
	secondPeak := 0
	minDistance := 50
	for i := 0; i < 256; i++ {
		dist := int(math.Abs(float64(i - firstPeak)))
		if dist > minDistance && histogram[i] > histogram[secondPeak] {
			secondPeak = i
		}
	}

	// If we didn't find a distinct second peak, return 128 as fallback
	if secondPeak == 0 {
		return 128
	}

	return uint8((firstPeak + secondPeak) / 2)
}

// OtsuThreshold calculates the optimal global threshold using Otsu's method.
func (p *Processor) OtsuThreshold(frame []byte) uint8 {
	histogram := make([]float64, 256)
	totalPixels := 0.0

	// Full histogram for Otsu
	for i := 0; i < len(frame); i += p.bytesPP {
		r := frame[i]
		g := frame[i+1]
		b := frame[i+2]
		gray := uint8(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b))
		histogram[gray]++
		totalPixels++
	}

	// Normalize histogram
	for i := 0; i < 256; i++ {
		histogram[i] /= totalPixels
	}

	var maxVar float64
	var threshold uint8 = 128

	var mGlobal float64
	for i := 0; i < 256; i++ {
		mGlobal += float64(i) * histogram[i]
	}

	var w0, m0 float64
	for t := 0; t < 256; t++ {
		w0 += histogram[t]
		if w0 == 0 {
			continue
		}

		w1 := 1.0 - w0
		if w1 == 0 {
			break
		}

		m0 += float64(t) * histogram[t]
		mB := (mGlobal*w0 - m0)

		varianceBetween := (mB * mB) / (w0 * w1)

		if varianceBetween > maxVar {
			maxVar = varianceBetween
			threshold = uint8(t)
		}
	}

	return threshold
}

// ProcessWithThresholdMode applies the binary conversion using the selected threshold mode.
func (p *Processor) ProcessWithThresholdMode(frame []byte, mode int) []byte {
	var threshold uint8
	switch mode {
	case 2:
		threshold = p.AdaptiveThreshold(frame)
	case 3:
		threshold = p.OtsuThreshold(frame)
	default:
		threshold = 128
	}

	return p.BinaryGrayscaleInverseWithThreshold(frame, threshold)
}
