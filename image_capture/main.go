package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	devicePath  = "/dev/video0"
	interval    = 10 * time.Second
	imagesDir   = "images"
	logFilename = "logs.txt"
)

func main() {
	// Create images directory
	err := os.MkdirAll(imagesDir, 0755)
	if err != nil {
		log.Fatalf("Error creating images directory: %v\n", err)
	}

	// Open log file in append mode
	logFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v\n", err)
	}
	defer logFile.Close()

	// Setup logging to both stdout and log file
	mw := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(mw, "", log.LstdFlags)

	logger.Println("=========================================")
	logger.Println("Starting webcam image capture service...")
	logger.Printf("Device: %s\n", devicePath)
	logger.Printf("Interval: %v\n", interval)
	logger.Printf("Images Directory: %s\n", imagesDir)
	logger.Printf("Log File: %s\n", logFilename)
	logger.Println("=========================================")

	// Channel to handle termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Ticker for periodic captures
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Perform initial capture immediately
	captureFrame(logger)

	for {
		select {
		case <-ticker.C:
			captureFrame(logger)
		case sig := <-sigChan:
			logger.Printf("Received signal: %v. Shutting down webcam capture service...\n", sig)
			return
		}
	}
}

func captureFrame(logger *log.Logger) {
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(imagesDir, fmt.Sprintf("capture_%s.png", timestamp))

	logger.Printf("Attempting capture: %s...\n", filename)

	// FFmpeg command to capture 1 frame from /dev/video0
	// -y to overwrite existing file (should be unique due to timestamp)
	// -f v4l2 specifying video4linux2 format
	// -i /dev/video0 specifying the input source
	// -vframes 1 to capture only 1 frame
	// -f image2 to output as image sequence/single image
	cmd := exec.Command("ffmpeg", "-y", "-f", "v4l2", "-i", devicePath, "-vframes", "1", "-f", "image2", filename)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	if err != nil {
		logger.Printf("ERROR: Capture failed after %v: %v\n", duration, err)
		if stderr.Len() > 0 {
			logger.Printf("FFmpeg Stderr:\n%s\n", stderr.String())
		}
	} else {
		// Verify file was indeed created and is not empty
		if info, err := os.Stat(filename); err == nil && info.Size() > 0 {
			logger.Printf("SUCCESS: Saved %s (%d bytes) in %v\n", filename, info.Size(), duration)
		} else {
			logger.Printf("WARNING: FFmpeg succeeded but output file %s is missing or empty!\n", filename)
		}
	}
}
