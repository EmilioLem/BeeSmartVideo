package aruco

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
)

// CropJPEG extracts a square crop centered on a track centroid and returns the
// JPEG bytes.
//
// fullFrame is the full-resolution RGB24 frame, frameW/frameH its size, cx/cy
// the centroid in processing coordinates, scale the processing->full-res factor
// (3 with the current pipeline) and cropSize the full-resolution side length.
func CropJPEG(fullFrame []byte, frameW, frameH, cx, cy, scale, cropSize int) ([]byte, error) {
	if fullFrame == nil {
		return nil, fmt.Errorf("aruco: nil frame")
	}
	if cropSize <= 0 {
		return nil, fmt.Errorf("aruco: invalid crop size %d", cropSize)
	}

	const bytesPP = 3 // RGB24
	origX := cx * scale
	origY := cy * scale
	half := cropSize / 2

	img := image.NewRGBA(image.Rect(0, 0, cropSize, cropSize))
	for y := 0; y < cropSize; y++ {
		for x := 0; x < cropSize; x++ {
			sx := origX - half + x
			sy := origY - half + y
			if sx < 0 || sx >= frameW || sy < 0 || sy >= frameH {
				continue // out of bounds stays black
			}
			idx := (sy*frameW + sx) * bytesPP
			if idx+2 >= len(fullFrame) {
				continue
			}
			img.SetRGBA(x, y, color.RGBA{
				R: fullFrame[idx],
				G: fullFrame[idx+1],
				B: fullFrame[idx+2],
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("aruco: encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

// EncodeCropJPEG is CropJPEG plus base64, ready for the worker protocol.
func EncodeCropJPEG(fullFrame []byte, frameW, frameH, cx, cy, scale, cropSize int) (string, error) {
	raw, err := CropJPEG(fullFrame, frameW, frameH, cx, cy, scale, cropSize)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
