# Efficient Filtering V1

Implementation and logic behind the "Efficient Filtering V1" algorithm found in `logic/efficient_filtering.go`. This method is designed to process binary frames to detect, clean, and segment bees effectively, handling challenges like image noise and overlapping objects.

## Overview

The algorithm takes a binary frame (where white pixels represent potential objects/bees and black pixels represent background) and applies a series of steps:

1. **Morphological Repair**: Cleans up noise and fills gaps in detected objects.
2. **Connected Component Analysis**: Identifies distinct groups of pixels (blobs).
3. **Intelligent Splitting and Merging**: Adjusts blobs based on an expected bee size to handle fragmented detections and overlapping bees.
4. **Visualization**: Outputs a colored frame with a reference guide.

---

## Configuration Parameters

At the top of the `ApplyEfficientFilteringV1` function, several tweakable parameters control the algorithm's behavior:

* **`morphPasses`** (Default: `2`)
  * The number of iterations for erosion and dilation operations. Increasing this makes the cleaning more aggressive but might distort shapes.
* **`divergenceThreshold`** (Default: `0.20` or 20%)
  * The allowed deviation from the `expectedBeeArea` before splitting or merging logic is triggered.
  * *Note: The code comment mentions 40%, but the actual value used is `0.20`.*
* **`closenessThreshold`** (Default: `30.0`)
  * The maximum distance (in pixels) between the centroids of two blobs to consider them for merging.
* **`expectedBeeArea`** (Default: `1050.0`)
  * The expected average size of a single bee in pixels. This is a critical baseline for the splitting and merging logic.

---

## Detailed Processing Steps

### Step 1: Initialization

The function receives a `binaryFrame`. It creates a `workingFrame` copy to perform modifications without affecting the original input.

### Step 2: Morphological Repair (Noise Reduction & Gap Filling)

To handle noise and fragmented parts of bees, the algorithm performs a sequence of morphological operations (Closing followed by Opening):

1. **Dilation** (`morphPasses` times): Expands white regions. This helps connect fragmented parts of a bee that might have been separated by thresholding.
2. **Erosion** (`morphPasses` times): Shrinks white regions back. This restores the size after dilation and removes small noise. (Steps 1 & 2 constitute a **Closing** operation).
3. **Erosion** (`morphPasses` times): Further shrinks regions to remove thin connections and small noise.
4. **Dilation** (`morphPasses` times): Expands regions back. (Steps 3 & 4 constitute an **Opening** operation).

The implementation uses custom 3x3 erosion and dilation functions (`customErode` and `customDilate`) that consider 4-neighbors (up, down, left, right).

### Step 3: Connected Component Analysis

After cleaning the frame, the algorithm identifies distinct "blobs" (groups of connected white pixels):

* `FindBlobs`: Scans the `workingFrame` to group adjacent pixels into separate lists.
* `ExtractBlobs`: Processes these groups to calculate properties such as area and centroid.

### Step 4: Intelligent Splitting and Merging

This is the core heuristic of the algorithm. It iterates through the detected blobs and decides whether to keep them, merge them, or split them based on their size relative to `expectedBeeArea`.

* **Noise Filtering**

  * Blobs with an area smaller than **10 pixels** are ignored entirely.
* **Merging (Handling Fragmented Bees)**

  * **Condition**: If a blob's area is smaller than the lower bound: `Area < expectedBeeArea * (1.0 - divergenceThreshold)` (e.g., $< 840$ pixels).
  * **Action**: The algorithm looks for other blobs whose centroids are within the `closenessThreshold` (30 pixels).
  * **Result**: If close neighbors are found, their pixels are merged into a single larger blob.
* **Splitting (Handling Overlapping Bees)**

  * **Condition**: If a blob's area is larger than the upper bound: `Area > expectedBeeArea * (1.0 + divergenceThreshold)` (e.g., $> 1260$ pixels).
  * **Action**: The algorithm assumes multiple bees are overlapping. It calculates the expected number of bees: $k = \text{round}(\text{Area} / \text{expectedBeeArea})$. If $k \le 1$, it defaults to $k = 2$.
  * **Result**: It applies **K-Means Clustering** on the pixels of that specific blob to split it into $k$ separate, smaller blobs.
* **Normal Blobs**

  * Blobs falling within the acceptable range are kept as they are.

---

## Visualization and Output

The function produces two outputs:

1. **Final Blobs**: The list of processed `Blob` structures.
2. **Output Frame**: A frame where each final blob is filled with a unique, vibrant color for debugging and visualization.

### Reference Oval

To assist in tuning the `expectedBeeArea` parameter, the algorithm draws a white reference oval in the top-left corner (centered at $x=60, y=30$).

* **Math**: It assumes a bee has a 2:1 aspect ratio (semi-major axis $a = 2b$, where $b$ is the semi-minor axis).
* Given $\text{Area} = \pi \cdot a \cdot b = 2\pi b^2$, it calculates:
  * $b = \sqrt{\text{expectedBeeArea} / (2\pi)}$
  * $a = 2b$
* This visualizes the expected size of a bee directly on the video output.
