# BeeSmartVideo

BeeSmartVideo is a real-time video processing tool written in Go, designed to count bees (or other similar objects) in a live video stream. It leverages FFMPEG for video input from a webcam and FFplay for real-time visualization of the processed frames.

## Features

## Features

### Segmentation Methods

BeeSmartVideo provides 14 different algorithms for object detection and counting, selectable via the interactive menu:

| ID | Method                               | Description                                                                  |
| -- | ------------------------------------ | ---------------------------------------------------------------------------- |
| 1  | **K-Means Clustering**         | Clusters white pixels to identify potential objects.                         |
| 2  | **Erosion Technique**          | Uses morphological erosion to separate overlapping objects.                  |
| 3  | **Convex Hull Analysis**       | Analyzes the convex area of contours to classify objects.                    |
| 4  | **Perimeter vs Area**          | Uses isoperimetric ratios to distinguish bees.                               |
| 5  | **Morphological Repair**       | Dilation/Erosion cycles to clean up noise and fragmented objects.            |
| 6  | **Watershed (Heuristic)**      | Uses area-based splitting for clusters (bees touching each other).           |
| 7  | **Convexity Defect splitting** | Identifies "dents" in clusters to split multiple bees.                       |
| 8  | **Skeleton-Based Splitting**   | Analyzes aspect ratios to split elongated clusters.                          |
| 9  | **Dynamic Area Estimation**    | **Robust:** Uses median area of current blobs to estimate counts.      |
| 10 | **Shape Filtering**            | Filters blobs based on geometric descriptors.                                |
| 11 | **Neighbor Merge Pass**        | Merges redundant detections based on proximity.                              |
| 12 | **Motion Consistency**         | Validates detections based on temporal direction.                            |
| 13 | **Temporal Stabilization**     | Reduces flicker by stabilizing blob persistence.                             |
| 14 | **Multi-Stage Pipeline**       | **Advanced:** Combined Morphological Repair + Dynamic Area Estimation. |
| 15 | **Hybrid Convex Pipeline**     | **Mix:** Morphological Repair + Convex Hull Analysis + Area Estimation.      |
| 16 | **Geometric Density Filter**   | **Mix:** Shape Filtering + Neighbor Merge + Dynamic Area Estimation.         |
| 17 | **Structural Skeleton Split**  | **Mix:** Morphological Repair + Skeleton Splitting + Watershed Heuristic.    |
| 18 | **Temporal Consistency Mix**   | **Mix:** Advanced Pipeline + Motion/Temporal filtering (Most Stable).        |
| 19 | **Perfect Bee Finder**         | **Mix:** Strict Shape Filtering + Isoperimetric Quotient + Defect Splitting. |

### Processing Modes (Thresholding)

1. **Static (128)**: Fast, fixed value. Best for controlled lighting.
2. **Static (118, 108, 98)**: Lighter static thresholds (-10, -20, -30).
3. **Static (138, 148, 158)**: Darker static thresholds (+10, +20, +30).
4. **Adaptive Peak Midpoint**: Automatically calculates threshold from image histogram.
5. **Otsu's Global Threshold**: Standard robust method for separating foreground from background.
6. **Basic Movement Layer**: **Powerful:** Detects only *moving* objects, ignoring the static background.

---

## Recommended Combinations 

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
go run main.go [method] [threshold] [source] [tracking_method] [show_ids] [smoothness] [aggressiveness]
```

- **method**: 1-19 (See table above)
- **threshold**: 1-10 (Standard and offset modes)
- **source**: Device index or video path
- **tracking**: 0-6 (Centroid to Deep Path Tracker)
- **show_ids**: true/false
- **smoothness**: 0-4 (None to Aggressive blur)
- **aggressiveness**: 0.5-5.0 (Scaling of segmentation sensitivity)

### Export AI Dataset 

When enabled in the menu, this feature generates a structured dataset in the `dataset/` folder for machine learning purposes.

#### CSV Data (`bee_data.csv`)

For every confirmed tracked bee, a row is added to `bee_data.csv` with the following columns:

* **Frame**: Sequential frame number in the video stream.
* **ID**: Persistent tracking ID of the bee.
* **X, Y**: Centroid coordinates (relative to the 240p processing resolution).
* **Area**: Total pixel count of the bee blob.
* **Solidity**: Measure of shape compactness (Area / Convex Hull Area).
* **Ratio**: Aspect ratio of the detected blob.
* **VX, VY**: Estimated horizontal and vertical velocity components.

#### Image Cropping logic

Individual bee images are extracted to the `dataset/images/` directory:

* **Dimensions**: Fixed **192x192 pixel** square crops.
* **Centering**: The crop is centered on the bee's centroid.
* **High-Res Source**: Coordinates are automatically scaled by 3x to map from the internal processing resolution (240p) back to the **original 720p capture resolution**, ensuring high-quality training data.
* **File Naming**: Saved as `{ID}_{Frame}.jpg` for easy mapping to the CSV.
* **Edge Handling**: If a bee is near the border, the crop is padded with black pixels to maintain the 192x192 format.

## Project Structure

- `main.go`: Entry point, handles the menu and processing loop.
- `logic/`: Core image processing including 14 segmentation methods.
- `menu/`: Interactive TUI settings menu using `huh`.
- `in/`: FFMPEG capture wrapper.
- `out/`: FFplay visualization and Web Dashboard.

## License

MIT License.
