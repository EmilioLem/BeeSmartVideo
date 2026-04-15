# BeeSmartVideo

BeeSmartVideo is a real-time video processing tool written in Go, designed to count bees (or other similar objects) in a live video stream. It leverages FFMPEG for video input from a webcam and FFplay for real-time visualization of the processed frames.

## Features

## Features

### Segmentation Methods
BeeSmartVideo provides 14 different algorithms for object detection and counting, selectable via the interactive menu:

| ID | Method | Description |
|---|---|---|
| 1 | **K-Means Clustering** | Clusters white pixels to identify potential objects. |
| 2 | **Erosion Technique** | Uses morphological erosion to separate overlapping objects. |
| 3 | **Convex Hull Analysis** | Analyzes the convex area of contours to classify objects. |
| 4 | **Perimeter vs Area** | Uses isoperimetric ratios to distinguish bees. |
| 5 | **Morphological Repair** | Dilation/Erosion cycles to clean up noise and fragmented objects. |
| 6 | **Watershed (Heuristic)** | Uses area-based splitting for clusters (bees touching each other). |
| 7 | **Convexity Defect splitting** | Identifies "dents" in clusters to split multiple bees. |
| 8 | **Skeleton-Based Splitting** | Analyzes aspect ratios to split elongated clusters. |
| 9 | **Dynamic Area Estimation** | **Robust:** Uses median area of current blobs to estimate counts. |
| 10 | **Shape Filtering** | Filters blobs based on geometric descriptors. |
| 11 | **Neighbor Merge Pass** | Merges redundant detections based on proximity. |
| 12 | **Motion Consistency** | Validates detections based on temporal direction. |
| 13 | **Temporal Stabilization** | Reduces flicker by stabilizing blob persistence. |
| 14 | **Multi-Stage Pipeline** | **Advanced:** Combined Morphological Repair + Dynamic Area Estimation. |

### Processing Modes (Thresholding)
1. **Static (128)**: Fast, fixed value. Best for controlled lighting.
2. **Adaptive Peak Midpoint**: Automatically calculates threshold from image histogram.
3. **Otsu's Global Threshold**: Standard robust method for separating foreground from background.
4. **Basic Movement Layer**: **Powerful:** Detects only *moving* objects, ignoring the static background (like hive structure).

---

## Recommended Combinations 🐝

With many settings to choose from, here are the most effective combinations for common scenarios:

### 1. The "Gold Standard" (Most Accurate)
*Best for general outdoor counting in varied light.*
- **Method:** 14 (Multi-Stage Pipeline)
- **Threshold:** 3 (Otsu's Global)
- **Tracking:** 3 (Kalman Filter)
- **Smoothness:** 2 (Medium)

### 2. The "Busy Hive Entrance"
*Best for avoiding false counts from the hive structure itself.*
- **Method:** 9 (Dynamic Area Estimation)
- **Threshold:** 4 (Basic Movement Layer)
- **Tracking:** 2 (Hungarian Assignment)
- **Smoothness:** 1 (Light)

### 3. The "Low-Spec / High FPS"
*Best for maximizing performance on older CPUs or internal webcams.*
- **Method:** 2 (Erosion Technique)
- **Threshold:** 1 (Static)
- **Tracking:** 1 (Nearest-Centroid)
- **Smoothness:** 0 (None)

### 4. The "Night / Low-Contrast"
*Best for Grainy or noisy video feeds.*
- **Method:** 5 (Morphological Repair)
- **Threshold:** 2 (Adaptive Peak)
- **Tracking:** 3 (Kalman Filter)
- **Smoothness:** 4 (Aggressive)

---

## Prerequisites

- **Go**: Version 1.25.5 or higher.
- **FFMPEG**: Required for capturing live video.
- **FFplay**: Required for real-time visualization.

## Usage

Run the tool using `go run main.go`. An interactive menu will appear to guide you through the settings.

### CLI Arguments
You can also bypass the menu by passing arguments:
```bash
go run main.go [method] [threshold] [source] [tracking_method] [show_ids] [smoothness]
```

- **method**: 1-14 (See table above)
- **threshold**: 1-4
- **source**: Device index (e.g., 0) or path to a video file (e.g., ./videoSamples/bees.mp4)
- **tracking**: 0=None, 1=Centroid, 2=Hungarian, 3=Kalman, etc.
- **show_ids**: true/false
- **smoothness**: 0-4 (None to Aggressive)

## Project Structure

- `main.go`: Entry point, handles the menu and processing loop.
- `logic/`: Core image processing including 14 segmentation methods.
- `menu/`: Interactive TUI settings menu using `huh`.
- `in/`: FFMPEG capture wrapper.
- `out/`: FFplay visualization and Web Dashboard.

## License
MIT License.
