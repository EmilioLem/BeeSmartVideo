package in

import (
	"fmt"
	"io"
	"os/exec"
)

// LiveInput handles video input from a webcam using ffmpeg
type LiveInput struct {
	cmd       *exec.Cmd
	stdout    io.ReadCloser
	width     int
	height    int
	frameSize int
}

// NewLiveInput creates a new live input stream from the specified video device
func NewLiveInput(device string, width, height int) (*LiveInput, error) {
	bytesPP := 3 // RGB24
	frameSize := width * height * bytesPP

	cmd := exec.Command(
		"ffmpeg",
		"-loglevel", "quiet",
		"-f", "v4l2",
		"-video_size", fmt.Sprintf("%dx%d", width, height),
		"-i", device,
		"-pix_fmt", "rgb24",
		"-f", "rawvideo",
		"-",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg input: %w", err)
	}

	return &LiveInput{
		cmd:       cmd,
		stdout:    stdout,
		width:     width,
		height:    height,
		frameSize: frameSize,
	}, nil
}

// ReadFrame reads a single frame from the input stream
func (li *LiveInput) ReadFrame() ([]byte, error) {
	frame := make([]byte, li.frameSize)
	_, err := io.ReadFull(li.stdout, frame)
	if err != nil {
		return nil, fmt.Errorf("failed to read frame: %w", err)
	}
	return frame, nil
}

// Close cleans up the input stream
func (li *LiveInput) Close() error {
	if li.stdout != nil {
		li.stdout.Close()
	}
	if li.cmd != nil && li.cmd.Process != nil {
		li.cmd.Process.Kill()
		li.cmd.Wait()
	}
	return nil
}
