---
description: Deep-dive logic of BeeSmartVideo thresholding, movement layers, and segmentation alternatives
---

# BeeSmartVideo: Core Logic & Algorithm Deep Dive

This document details the exact programming logic for the selectable thresholding, movement, and segmentation methods in the BeeSmartVideo project.

## 1. Core Binary Conversion (`BinaryGrayscaleInverseWithThreshold`)

Standard transformation for single-frame modes:
1.  **Grayscale:** $Gray_i = 0.299 \cdot R_i + 0.587 \cdot G_i + 0.114 \cdot B_i$
2.  **Binary:** $Bin_i = 255$ if $Gray_i \ge T$, else $0$.
3.  **Inversion:** $Inv_i = 255 - Bin_i$ (Bees = White).

---

## 2. Thresholding & Movement Modes

### Category: Single Frame Thresholding
- **Mode 1: Static:** $T = 128$.
- **Mode 2: Adaptive:** Midpoint between two major histogram peaks (distance > 50).
- **Mode 3: Otsu:** Minimizes intra-class variance over the full histogram.

### Category: Frame-over-Time (Background Subtraction)
- **Mode 4: Basic Movement Layer**
    1.  **Background Model ($BG$):** A running average stored at pixel level.
    2.  **Update:** $BG_{i, t} = BG_{i, t-1} \cdot (1 - \delta) + Frame_{i, t} \cdot \delta$ (Current $\delta = 0.05$).
    3.  **Difference:** $Diff_i = |Frame_{i, t} - BG_{i, t}|$.
    4.  **Binary Output:** $Out_i = 255$ if $Diff_i > 30$, else $0$.
    - *Result:* Isolates moving objects from the static background without needing grayscale inversion.

---

## 3. Current Segmentation Methods (Implementation)

- **Method 1: K-Means:** Iterative centroid clustering ($K=7$).
- **Method 2: Erosion + CCL:** 3x3 erosion kernel before Flood Fill blob detection.
- **Method 3: Solidity:** $Count = Area / Area_{avg}$ where $Area_{avg}$ varies by Solidity ($Area/BBoxArea$).
- **Method 4: Perimeter/Area:** $Count = Area / Area_{avg}$ where $Area_{avg}$ varies by Isoperimetric Ratio ($P^2/A$).

---

## 4. Advanced Segmentation Strategy Roadmap (AI Alternatives)

These strategies are specifically suited for movement-subtraction masks (Mode 4).

### 1. Morphological Repair
- **Step 1:** `Close(mask, head_size)` to reconnect heads/bodies.
- **Step 2:** `FillHoles(mask)` to fix internal subtraction artifacts.
- **Step 3:** `Open(mask, noise_kernel)` to remove salt-and-pepper noise.

### 2. Distance Transform + Watershed (Overlap Splitting)
- **Logic:** Compute distance to nearest edge for each pixel. Find local maxima (centers). Use these as seeds for a Watershed algorithm to grow regions until they hit neighbors.
- *Why:* Effectively splits bees that are touching side-by-side.

### 3. Convexity Defect Splitting
- **Logic:** Compare blob contour to its Convex Hull. Deep "valleys" (defects) indicate the point where two bees meet. Split the blob at the line between the two deepest defects.

### 4. Skeleton-Based Splitting
- **Logic:** Reduce blob to a 1-pixel wide skeleton. Find "branch points" (pixels with > 2 neighbors). If branches exist, the blob is a cluster; split at the junctions.

### 5. Dynamic Area Estimation
- **Logic:** Calculate the median area of all "small" blobs to find the $Area_{median}$ of a single bee in the current frame.
- **Count:** $Count = round(Area_{blob} / Area_{median})$.
- *Why:* Adapts to camera zoom/distance automatically.

### 6. Shape Filtering descriptors
- **Logic:** Compute `Eccentricity` (elongation), `Aspect Ratio`, and `Compactness` ($P^2/A$).
- **Discard:** Blobs with area < noise_threshold.
- **Merge Target:** High aspect ratio fragments should be merged with nearby center-mass blobs.

### 7. Neighbor Merge Pass
- **Logic:** For each small fragment, find the nearest neighboring blob within radius $R$. If their combined area $\approx Area_{median}$, merge them.

### 8. Motion Direction Consistency
- **Logic:** Track centroid displacement $(\Delta x, \Delta y)$ between frames. If two distinct blobs move with identical velocity vectors, they are likely fragments of the same bee.

### 9. Temporal Blob Stabilization
- **Logic:** Maintain a list of "Active Blobs". If a blob disappears and two new ones appear at the same location, mark them as fragments of the original.

### 10. Multi-Stage Pipeline (Recommended)
`MovedPixels -> Morph-Close -> Fill-Holes -> Noise-Removal -> CCL -> Watershed-Split -> Area-Estimation -> Temporal-Merge -> Count`
