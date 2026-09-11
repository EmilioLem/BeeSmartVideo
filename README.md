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

### Headless / Systemd Mode (Default)

By default, launching `BeeSmartVideo` runs in **Headless CLI mode** (no interactive terminal setup required), making it ideal for background daemons, automated scripts, and `systemd` service integration:

```bash
go run main.go
# or compile and run:
go build -o program main.go
./program
```

In headless mode, settings are loaded directly from `settings.json` (or `DefaultOptions()`), and the graphical display window (`ffplay`) is disabled so the process can run purely headless without X11/GUI requirements. All frame processing, tracking, dataset saving, Web Dashboard stats, and MQTT telemetry continue to execute as normal.

### Interactive TUI Menu (`huhForm`)

To launch the interactive selection menu (powered by `huh`), run with the `huhForm` or `--tui` flag:

```bash
go run main.go huhForm
# or
go run main.go --tui
```

The interactive menu lets you visually select input video sources, thresholding modes, tracking algorithms, telemetry toggles, and dataset export settings, automatically updating `settings.json` upon completion.

### Configuration Settings (`settings.json` & `menu/menu.go`)

Hardcoded settings are fully commented in `menu/menu.go` and configured in `settings.json`:

* `source`: Path to a video file (e.g. `./videoSamples/40secBeesWithPolen.mp4`) or `"live"` for camera input.
* `device_index`: Camera device index when `source` is `"live"` (e.g. `"0"` for `/dev/video0`).
* `method`: Processing algorithm (1 = Efficient filtering v1).
* `threshold_mode`: Strategy ID (e.g. 1 = Static 128 [0], 5 = Static 118 [-10], 4 = Movement Layer).
* `tracking_method`: Honeybee tracker ID (0 = None, 1 = Centroid, 2 = Hungarian, 3 = Kalman, etc.).
* `show_ids`: `true`/`false` to overlay track IDs on video output.
* `smoothness`: Blur intensity (0 = disabled).
* `save_data`: `true`/`false` to export dataset CSV & cropped bee images to `dataset/`.
* `loop_video`: `true`/`false` to loop video file playback.
* `enable_telemetry`: `true`/`false` to publish live counts via MQTT.
* `red_channel`: `true`/`false` to threshold on the red channel (see below). Default `true`.
* `enable_aruco`: `true`/`false` to decode ArUco marker IDs with the Python worker (v2 only). Default `true`.
* `aruco_dict`: dictionary the worker must match, e.g. `DICT_4X4_50`; empty = custom 10000x5 (v2 only).

### IMPORTANT: Red LED illumination (red channel)

The hive entrance is illuminated with **red LEDs only**. Therefore the whole
detection pipeline thresholds on the **red channel** instead of luminance:

* At the binary-threshold step (`logic/binaryGrayscaleInverseWithThreshold.go`),
  `gray` is the red value directly, not `0.299R + 0.587G + 0.114B`.
* Under red light, green and blue are almost pure sensor noise, so including
  them only adds noise. Using red matches the illumination.
* The ArUco worker (`v2/ArUcoReader02.py`) reads the same red channel, so the
  whole pipeline is consistent from thresholding to marker decoding.
* Set `"red_channel": false` in `settings.json` **only** when testing under
  white light (e.g. the existing sample videos). Expect very different counts
  otherwise.

### ArUco marker identity (v2)

When `enable_aruco` is on, `v2` starts `v2/ArUcoReader02.py` once as a
long-lived worker. For each newly confirmed track it crops the bee from the
full-resolution frame, sends it to the worker, and stores the decoded marker id
on the track (`logic.Track.MarkerID` / `MarkerLabel`). The dictionary is set by
`aruco_dict` (default `DICT_4X4_50`). Disable with `--no-aruco` or
`"enable_aruco": false`. If Python/OpenCV is missing, v2 logs a warning and
keeps running without marker decoding.

#### Tag-size discrepancy (4x4 vs 5x5)

The printed tags are **4×4** markers, so the default dictionary is
`DICT_4X4_50`. `ArUcoReader01.py` instead used a custom **5×5**
`extendDictionary(10000, 5)`, which can hold 10000 ids for the H1..H5 scheme.
A 4×4 dictionary is much smaller (`DICT_4X4_50` = 50 ids, `DICT_4X4_1000` =
1000), so **the 10000-marker / H1..H5 scheme cannot be used with 4×4 tags** and
the `H1..H5` label collapses to `H1-xxxx`. Set `"aruco_dict": "custom"` to use
the old 5×5 dictionary, or reprint the tags with a larger 4×4 dictionary if you
need more than 50 unique ids.


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

* **Dimensions**: Configurable square crops via `crop_size` (default **64 processing px**, i.e. **192x192** at full resolution).
* **Centering**: The crop is centered on the bee's centroid.
* **High-Res Source**: Coordinates are automatically scaled by 3x to map from the internal processing resolution (240p) back to the **original 720p capture resolution**, ensuring high-quality training data.
* **File Naming**: Saved as `{ID}_{Frame}.jpg` for easy mapping to the CSV.
* **Edge Handling**: If a bee is near the border, the crop is padded with black pixels to keep the configured square format.

## Project Structure

- `main.go`: Entry point, handles the menu and processing loop.
- `logic/`: Core image processing including 14 segmentation methods.
- `menu/`: Interactive TUI settings menu using `huh`.
- `in/`: FFMPEG capture wrapper.
- `out/`: FFplay visualization and Web Dashboard.

## License

MIT License.
