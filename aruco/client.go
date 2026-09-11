// Package aruco talks to the Python ArUco worker (v2/ArUcoReader02.py).
//
// The worker is started once as a long-lived process. Crops are sent as
// newline-delimited JSON on stdin and the decoded markers come back the same
// way on stdout, so no Python interpreter is spawned per frame.
package aruco

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// ProtocolVersion must match PROTOCOL_VERSION in ArUcoReader02.py.
const ProtocolVersion = 1

// Image is one bee crop to decode. JPEGB64 is a base64-encoded JPEG.
type Image struct {
	TrackID int    `json:"track_id"`
	JPEGB64 string `json:"jpeg_b64"`
}

type request struct {
	V      int     `json:"v"`
	ID     int     `json:"id"`
	Images []Image `json:"images"`
}

// Marker is the decoded result for a single crop.
type Marker struct {
	TrackID    int     `json:"track_id"`
	Found      bool    `json:"found"`
	MarkerID   *int    `json:"marker_id"`
	MarkerIDs  []int   `json:"marker_ids"`
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
	Error      *string `json:"error"`
}

type response struct {
	V       int      `json:"v"`
	ID      int      `json:"id"`
	Results []Marker `json:"results"`
	Error   string   `json:"error"`
}

// Client is a running ArUco worker process.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	nextID int
}

// NewClient launches the worker. python is the interpreter (e.g. "python3") and
// script is the path to ArUcoReader02.py. The worker's stderr is forwarded to
// this process's stderr so its log lines stay visible.
func NewClient(python, script string, extraArgs ...string) (*Client, error) {
	args := append([]string{script}, extraArgs...)
	cmd := exec.Command(python, args...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("aruco: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("aruco: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("aruco: start worker: %w", err)
	}

	return &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}, nil
}

// Decode sends a batch of crops and returns one Marker per crop, in the same
// order as the worker reports them (keyed by TrackID).
func (c *Client) Decode(images []Image) ([]Marker, error) {
	if c == nil {
		return nil, nil
	}
	if len(images) == 0 {
		return nil, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	payload, err := json.Marshal(request{V: ProtocolVersion, ID: c.nextID, Images: images})
	if err != nil {
		return nil, fmt.Errorf("aruco: marshal request: %w", err)
	}
	payload = append(payload, '\n')

	if _, err := c.stdin.Write(payload); err != nil {
		return nil, fmt.Errorf("aruco: write request: %w", err)
	}

	line, err := c.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("aruco: read response: %w", err)
	}

	var resp response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("aruco: bad response %q: %w", line, err)
	}
	if resp.V != ProtocolVersion {
		return nil, fmt.Errorf("aruco: protocol version mismatch (got %d)", resp.V)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("aruco: %s", resp.Error)
	}
	return resp.Results, nil
}

// Close shuts the worker down.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	return nil
}
