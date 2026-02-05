package logic

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
