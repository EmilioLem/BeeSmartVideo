package logic

// ProcessWithThresholdMode applies the binary conversion using the selected threshold mode.
func (p *Processor) ProcessWithThresholdMode(frame []byte, mode int) []byte {
	var threshold uint8
	switch mode {
	case 5:
		threshold = 118 // 128 - 10
	case 6:
		threshold = 108 // 128 - 20
	case 7:
		threshold = 98  // 128 - 30
	case 8:
		threshold = 138 // 128 + 10
	case 9:
		threshold = 148 // 128 + 20
	case 10:
		threshold = 158 // 128 + 30
	case 11:
		threshold = 78  // 128 - 50
	case 12:
		threshold = 88  // 128 - 40
	case 13:
		threshold = 168 // 128 + 40
	case 14:
		threshold = 178 // 128 + 50
	default:
		threshold = 128
	}
	return p.BinaryGrayscaleInverseWithThreshold(frame, threshold)
}
