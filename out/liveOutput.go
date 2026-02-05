package out

import (
	"fmt"
	"io"
	"os/exec"
)

// LiveOutput handles video output to a display window using ffplay
type LiveOutput struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	width  int
	height int
}

// NewLiveOutput creates a new live output display window
func NewLiveOutput(width, height int) (*LiveOutput, error) {
	cmd := exec.Command(
		"ffplay",
		"-loglevel", "quiet",
		"-f", "rawvideo",
		"-pixel_format", "rgb24",
		"-video_size", fmt.Sprintf("%dx%d", width, height),
		"-framerate", "30",
		"-window_title", "K-Means Clustering Demo",
		"-i", "-",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffplay output: %w", err)
	}

	return &LiveOutput{
		cmd:    cmd,
		stdin:  stdin,
		width:  width,
		height: height,
	}, nil
}

// WriteFrame writes a single frame to the output display
func (lo *LiveOutput) WriteFrame(frame []byte) error {
	_, err := lo.stdin.Write(frame)
	if err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}
	return nil
}

// Close cleans up the output stream
func (lo *LiveOutput) Close() error {
	if lo.stdin != nil {
		lo.stdin.Close()
	}
	if lo.cmd != nil && lo.cmd.Process != nil {
		lo.cmd.Process.Kill()
		lo.cmd.Wait()
	}
	return nil
}
