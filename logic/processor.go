package logic

// Point represents a 2D point (pixel coordinate)
type Point struct {
	X, Y int
}

// Processor handles all video frame processing logic
type Processor struct {
	width               int
	height              int
	bytesPP             int
	originalFrame       []byte
	lastWhitePixelCount int
}

// NewProcessor creates a new processor instance
func NewProcessor(width, height, bytesPP int) *Processor {
	return &Processor{
		width:   width,
		height:  height,
		bytesPP: bytesPP,
	}
}

// GetOriginalFrame returns the original frame before processing
func (p *Processor) GetOriginalFrame() []byte {
	return p.originalFrame
}

// FindBlobs identifies connected components in a binary frame
func (p *Processor) FindBlobs(frame []byte) [][]Point {
	visited := make([]bool, p.width*p.height)
	var blobs [][]Point

	for y := 0; y < p.height; y++ {
		for x := 0; x < p.width; x++ {
			pixelIdx := y*p.width + x
			if !visited[pixelIdx] && frame[pixelIdx*p.bytesPP] == 255 {
				blob := p.floodFill(frame, visited, x, y)
				if len(blob) > 10 { // Filter out noise
					blobs = append(blobs, blob)
				}
			}
		}
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
