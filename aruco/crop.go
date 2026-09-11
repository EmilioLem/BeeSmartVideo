package aruco

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
)

// EncodeCropJPEG extracts a square crop centered on a track centroid, encodes
// it as JPEG and returns it base64-encoded, ready for the worker protocol.
//
// fullFrame is the full-resolution RGB24 frame, frameW/frameH its size, cx/cy
// the centroid in processing coordinates, scale the processing->full-res factor
// (3 with the current pipeline) and cropSize the full-resolution side length.
func EncodeCropJPEG(fullFrame []byte, frameW, frameH, cx, cy, scale, cropSize int) (string, error) {
	if fullFrame == nil {
		return "", fmt.Errorf("aruco: nil frame")
	}
	if cropSize <= 0 {
		return "", fmt.Errorf("aruco: invalid crop size %d", cropSize)
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
		return "", fmt.Errorf("aruco: encode jpeg: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
