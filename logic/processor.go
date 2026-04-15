package logic

import (
	"strconv"
)

// Point represents a 2D point (pixel coordinate)
type Point struct {
	X, Y int
}

// BBox represents a bounding box
type BBox struct {
	MinX, MinY, MaxX, MaxY int
}

// Blob represents a detected object with metadata
type Blob struct {
	Points   []Point
	Centroid Point
	Area     int
	BBox     BBox
	Solidity float64
	Ratio    float64 // Isoperimetric ratio
}

// Track represents a persistently tracked object
type Track struct {
	ID            int
	Status        string // "tentative", "confirmed"
	Hits          int    // Consecutive detections
	Centroid      Point
	History       []Point
	LastSeenFrame int
	Area          int
	Solidity      float64 // Morphological memory
	Ratio         float64 // Morphological memory
	// Add velocity for Kalman-style prediction
	VX, VY float64
}

// Processor handles all video frame processing logic
type Processor struct {
	width               int
	height              int
	bytesPP             int
	originalFrame       []byte
	lastWhitePixelCount int
	backgroundModel     []float64
	bgDelta             float64
	// High-resolution storage
	FullResFrame []byte
	// Tracking state
	Tracks      []Track
	NextTrackID int
	FrameCount  int
	// Counting state
	CountUp   int
	CountDown int
	// Visualization toggle
	ShowIDs bool
	// Smoothness level (0-4)
	Smoothness int
	// Export options
	ExportData  bool
	DatasetPath string
	// Persistent buffers for optimization
	visitedTemp []bool
	binaryTemp  []byte
	generalTemp []byte
}

// NewProcessor creates a new processor instance
func NewProcessor(width, height, bytesPP int, bgDelta float64) *Processor {
	return &Processor{
		width:       width,
		height:      height,
		bytesPP:     bytesPP,
		bgDelta:     bgDelta,
		NextTrackID: 1,
		ShowIDs:     true, // Default to true
		// Pre-allocate buffers
		visitedTemp: make([]bool, width*height),
		binaryTemp:  make([]byte, width*height*bytesPP),
		generalTemp: make([]byte, width*height*bytesPP),
	}
}

// GetOriginalFrame returns the original frame before processing
func (p *Processor) GetOriginalFrame() []byte {
	return p.originalFrame
}

// FindBlobs identifies connected components in a binary frame
func (p *Processor) FindBlobs(frame []byte) [][]Point {
	for i := range p.visitedTemp {
		p.visitedTemp[i] = false
	}
	var blobs [][]Point

	for y := 0; y < p.height; y++ {
		for x := 0; x < p.width; x++ {
			pixelIdx := y*p.width + x
			if !p.visitedTemp[pixelIdx] && frame[pixelIdx*p.bytesPP] == 255 {
				blob := p.floodFill(frame, p.visitedTemp, x, y)
				if len(blob) > 10 { // Filter out noise
					blobs = append(blobs, blob)
				}
			}
		}
	}
	return blobs
}

// ExtractBlobs converts raw point sets into Blob structs with metadata
func (p *Processor) ExtractBlobs(pointGroups [][]Point) []Blob {
	var blobs []Blob
	for _, points := range pointGroups {
		// BBox
		minX, minY := p.width, p.height
		maxX, maxY := 0, 0
		sumX, sumY := 0, 0
		for _, pt := range points {
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
			sumX += pt.X
			sumY += pt.Y
		}

		area := len(points)
		bbox := BBox{minX, minY, maxX, maxY}
		bboxArea := (maxX - minX + 1) * (maxY - minY + 1)

		blobs = append(blobs, Blob{
			Points:   points,
			Centroid: Point{X: sumX / area, Y: sumY / area},
			Area:     area,
			BBox:     bbox,
			Solidity: float64(area) / float64(bboxArea),
		})
	}
	return blobs
}

func (p *Processor) floodFill(frame []byte, visited []bool, x, y int) []Point {
	var blob []Point
	stack := []Point{{X: x, Y: y}}
	visited[y*p.width+x] = true

	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		blob = append(blob, curr)

		// Check 4-neighbors
		neighbors := []Point{
			{X: curr.X, Y: curr.Y - 1},
			{X: curr.X, Y: curr.Y + 1},
			{X: curr.X - 1, Y: curr.Y},
			{X: curr.X + 1, Y: curr.Y},
		}

		for _, n := range neighbors {
			if n.X >= 0 && n.X < p.width && n.Y >= 0 && n.Y < p.height {
				nIdx := n.Y*p.width + n.X
				if !visited[nIdx] && frame[nIdx*p.bytesPP] == 255 {
					visited[nIdx] = true
					stack = append(stack, n)
				}
			}
		}
	}
	return blob
}

// GetVibrantColor returns a high-contrast color based on index.
func (p *Processor) GetVibrantColor(i int) [3]uint8 {
	// Use a fixed palette of vibrant colors
	palette := [][3]uint8{
		{255, 0, 0},     // Red
		{0, 255, 0},     // Lime
		{0, 0, 255},     // Blue
		{255, 255, 0},   // Yellow
		{255, 0, 255},   // Magenta
		{0, 255, 255},   // Cyan
		{255, 165, 0},   // Orange
		{128, 0, 128},   // Purple
		{0, 128, 0},     // Green
		{255, 192, 203}, // Pink
		{165, 42, 42},   // Brown
		{240, 230, 140}, // Khaki
	}
	return palette[i%len(palette)]
}

// --- Visualization Helpers (Bitmap Font) ---

// font5x7 defines digits 0-9 as 7 rows of 5 boolean states for maximum legibility
var font5x7 = [10][7][5]bool{
	// 0
	{
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	// 1
	{
		{false, false, true, false, false},
		{false, true, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{true, true, true, true, true},
	},
	// 2
	{
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, true, true, true, true},
	},
	// 3
	{
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
	// 4
	{
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
	},
	// 5
	{
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
	// 6
	{
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, false, false, false, false},
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	// 7
	{
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
		{false, false, false, true, false},
		{false, false, true, false, false},
		{false, true, false, false, false},
		{true, false, false, false, false},
	},
	// 8
	{
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	// 9
	{
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
}

// DrawDigit renders a single 5x7 digit at (x,y) with scaling using the boolean font
func (p *Processor) DrawDigit(buf []byte, digit, x, y, scale int, color [3]byte) {
	if digit < 0 || digit > 9 {
		return
	}
	f := font5x7[digit]
	for row := 0; row < 7; row++ {
		for col := 0; col < 5; col++ {
			if f[row][col] {
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						px, py := x+col*scale+sx, y+row*scale+sy
						if px >= 0 && px < p.width && py >= 0 && py < p.height {
							idx := (py*p.width + px) * p.bytesPP
							buf[idx], buf[idx+1], buf[idx+2] = color[0], color[1], color[2]
						}
					}
				}
			}
		}
	}
}

// DrawNumber renders a multi-digit number with a background box and scaling
func (p *Processor) DrawNumber(buf []byte, num, x, y, scale int, color [3]byte) {
	s := strconv.Itoa(num)
	charWidth := 5 * scale
	charHeight := 7 * scale
	spacing := 1 * scale
	totalWidth := len(s)*charWidth + (len(s)-1)*spacing

	// Draw background box (black) with padding
	padding := 2 * scale
	bgX, bgY := x-padding, y-padding
	bgW, bgH := totalWidth+padding*2, charHeight+padding*2

	for py := bgY; py < bgY+bgH; py++ {
		for px := bgX; px < bgX+bgW; px++ {
			if px >= 0 && px < p.width && py >= 0 && py < p.height {
				idx := (py*p.width + px) * p.bytesPP
				buf[idx], buf[idx+1], buf[idx+2] = 0, 0, 0
			}
		}
	}

	// Draw digits
	for i, char := range s {
		digit := int(char - '0')
		p.DrawDigit(buf, digit, x+i*(charWidth+spacing), y, scale, color)
	}
}

// OverlayTracks draws track IDs near their current centroids and the counting boundary
func (p *Processor) OverlayTracks(buf []byte) {
	// Draw horizontal boundary line (White and Black for contrast)
	midY := p.height / 2
	for x := 0; x < p.width; x++ {
		idx := (midY*p.width + x) * p.bytesPP
		// Black line
		buf[idx], buf[idx+1], buf[idx+2] = 0, 0, 0
		// White dotted/dashed look or just a second line above/below
		if x%4 < 2 {
			idx2 := ((midY+1)*p.width + x) * p.bytesPP
			if midY+1 < p.height {
				buf[idx2], buf[idx2+1], buf[idx2+2] = 255, 255, 255
			}
		}
	}

	if p.ShowIDs {
		for i, t := range p.Tracks {
			color := p.GetVibrantColor(i)
			// Draw path history (dots)
			for _, pt := range t.History {
				// Draw a 2x2 dot for each history point
				for dy := 0; dy < 2; dy++ {
					for dx := 0; dx < 2; dx++ {
						px, py := pt.X+dx, pt.Y+dy
						if px >= 0 && px < p.width && py >= 0 && py < p.height {
							idx := (py*p.width + px) * p.bytesPP
							buf[idx], buf[idx+1], buf[idx+2] = color[0], color[1], color[2]
						}
					}
				}
			}

			// Centered above the bee, scale 2 usually works well for 5x7
			scale := 2
			// Offset slightly to be above the centroid
			p.DrawNumber(buf, t.ID, t.Centroid.X-10, t.Centroid.Y-25, scale, [3]byte{255, 255, 255})
		}
	}
}

// ApplyBlur applies a box blur to the frame based on p.Smoothness level (0-4).
// Uses a separable horizontal and vertical pass for efficiency.
func (p *Processor) ApplyBlur(frame []byte) []byte {
	if p.Smoothness <= 0 {
		return frame
	}

	// Kernel sizes corresponding to 0, 1, 2, 3, 4
	kernelSizes := []int{0, 3, 5, 7, 9}
	kSize := kernelSizes[p.Smoothness]
	radius := kSize / 2

	if len(p.generalTemp) != len(frame) {
		p.generalTemp = make([]byte, len(frame))
		p.binaryTemp = make([]byte, len(frame))
	}
	temp := p.generalTemp
	out := p.binaryTemp

	// Horizontal pass
	for y := 0; y < p.height; y++ {
		for x := 0; x < p.width; x++ {
			var rSum, gSum, bSum int
			count := 0
			for kx := -radius; kx <= radius; kx++ {
				nx := x + kx
				if nx >= 0 && nx < p.width {
					idx := (y*p.width + nx) * p.bytesPP
					rSum += int(frame[idx])
					gSum += int(frame[idx+1])
					bSum += int(frame[idx+2])
					count++
				}
			}
			idx := (y*p.width + x) * p.bytesPP
			temp[idx] = uint8(rSum / count)
			temp[idx+1] = uint8(gSum / count)
			temp[idx+2] = uint8(bSum / count)
		}
	}

	// Vertical pass
	for x := 0; x < p.width; x++ {
		for y := 0; y < p.height; y++ {
			var rSum, gSum, bSum int
			count := 0
			for ky := -radius; ky <= radius; ky++ {
				ny := y + ky
				if ny >= 0 && ny < p.height {
					idx := (ny*p.width + x) * p.bytesPP
					rSum += int(temp[idx])
					gSum += int(temp[idx+1])
					bSum += int(temp[idx+2])
					count++
				}
			}
			idx := (y*p.width + x) * p.bytesPP
			out[idx] = uint8(rSum / count)
			out[idx+1] = uint8(gSum / count)
			out[idx+2] = uint8(bSum / count)
		}
	}

	return out
}

// GetCounts returns the current Up/Down counts
func (p *Processor) GetCounts() (int, int) {
	return p.CountUp, p.CountDown
}

// Subsample2x reduces a frame's resolution by 2x (e.g., 720p to 360p)
func (p *Processor) Subsample2x(src []byte, srcW, srcH int) []byte {
	dstW, dstH := srcW/2, srcH/2
	dst := make([]byte, dstW*dstH*p.bytesPP)

	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			srcIdx := (y*2*srcW + x*2) * p.bytesPP
			dstIdx := (y*dstW + x) * p.bytesPP

			dst[dstIdx] = src[srcIdx]
			dst[dstIdx+1] = src[srcIdx+1]
			dst[dstIdx+2] = src[srcIdx+2]
		}
	}
	return dst
}

// Subsample3x reduces a frame's resolution by 3x (e.g., 720p to ~240p)
// For 1280x720, this creates a 426x240 image.
func (p *Processor) Subsample3x(src []byte, srcW, srcH int) []byte {
	dstW, dstH := srcW/3, srcH/3
	if len(p.generalTemp) != dstW*dstH*p.bytesPP {
		p.generalTemp = make([]byte, dstW*dstH*p.bytesPP)
	}
	dst := p.generalTemp

	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			srcIdx := (y*3*srcW + x*3) * p.bytesPP
			dstIdx := (y*dstW + x) * p.bytesPP

			dst[dstIdx] = src[srcIdx]
			dst[dstIdx+1] = src[srcIdx+1]
			dst[dstIdx+2] = src[srcIdx+2]
		}
	}
	return dst
}
