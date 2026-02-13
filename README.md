# BeeSmartVideo

BeeSmartVideo is a real-time video processing tool written in Go, designed to count bees (or other similar objects) in a live video stream. It leverages FFMPEG for video input from a webcam and FFplay for real-time visualization of the processed frames.

## Features

### Counting Methods
The tool implements several computer vision techniques for object counting:
1.  **K-Means Clustering**: Clusters white pixels to identify potential objects.
2.  **Erosion Technique**: Uses morphological erosion to separate overlapping objects and count distinct blobs.
3.  **Convex Area Classification**: Analyzes the convex hull of contours to classify objects based on area.
4.  **Perimeter vs Area Comparison**: Uses the ratio of perimeter to area to distinguish objects.

### Threshold Modes
To handle different lighting conditions, BeeSmartVideo supports multiple binarization strategies:
1.  **Static Threshold**: Uses a fixed threshold value (128).
2.  **Adaptive Peak Midpoint**: Automatically calculates a threshold based on pixel intensity sampling.
3.  **Otsu's Global Threshold**: Implements the Otsu method for optimal automatic thresholding.

## Prerequisites

- **Go**: Version 1.25.5 or higher.
- **FFMPEG**: Required for capturing live video from `/dev/video0`.
- **FFplay**: Required for displaying the processed video window.

## Installation

1.  Clone the repository:
    ```bash
    git clone [repository-url]
    cd BeeSmartVideo
    ```
2.  Ensure FFMPEG and FFplay are installed on your system.

## Usage

You can run the tool using `go run main.go` or by using the provided executable.

### Windows Users
Run the `program.exe` file followed by the method number and threshold mode:
```cmd
program.exe [method_number] [threshold_mode]
```

### Linux Users
Run the `program` binary:
```bash
./program [method_number] [threshold_mode]
```

### From Source
```bash
go run main.go [method_number] [threshold_mode]
```

### Arguments

- **method_number**:
    - `1`: K-Means Clustering
    - `2`: Erosion Technique
    - `3`: Convex Area Classification
    - `4`: Perimeter vs Area Comparison
- **threshold_mode** (optional, default: 1):
    - `1`: Static Threshold
    - `2`: Adaptive Peak Midpoint
    - `3`: Otsu's Global Threshold

### Example

To run the tool using the Erosion Technique with Otsu's Global Threshold:
```bash
go run main.go 2 3
```

## Project Structure

- `main.go`: Entry point of the application, handles CLI arguments and the processing loop.
- `logic/`: Contains the core image processing logic, including thresholding and counting algorithms.
- `in/`: Handles video input stream via FFMPEG.
- `out/`: Handles video output display via FFplay.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
