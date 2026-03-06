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

## 4. Improved Segmentation Methods (Implementation Details)

These strategies are implemented in `logic/advanced_segmentation.go` and are optimized for movement-subtraction masks.

### 5. Morphological Repair (`ApplyMorphRepair`)
- **Logic:** Performs a **Closing** (Dilation -> Erosion) to bridge head/body gaps, followed by an **Opening** (Erosion -> Dilation) to eliminate salt-and-pepper noise.
- **Workflow:** `Input -> Dilate -> Erode -> Erode -> Dilate -> CCL`.

### 6. Distance Transform + Watershed (`ApplyWatershed`)
- **Logic:** Uses area as a proxy for watershed complexity. Blobs significantly larger than a single bee (> 800px) are mathematically divided by the expected bee area (450px) to estimate the cluster count.
- **Heuristic:** $Count = round(Area / 450.0)$ for large blobs.

### 7. Convexity Defect Splitting (`ApplyDefectSplitting`)
- **Logic:** Calculates **Solidity** ($Area / BBoxArea$). If solidity is low (< 0.45) and the blob is large, it assumes two bees are touching at an angle (concave defect) and counts as 2+.

### 8. Skeleton-Based Splitting (`ApplySkeletonSplitting`)
- **Logic:** Uses **Aspect Ratio** as a proxy for elongation. If $W/H > 2.5$ or $< 0.4$, the blob is treated as a linear cluster of bees and divided by the expected bee area (400px).

### 9. Dynamic Area Estimation (`ApplyDynamicAreaEstimation`)
- **Logic:** Calculates the **Median Area** of all current blobs.
- **Adaptive Count:** $Count = round(Area_{blob} / Area_{median})$. This allows the system to remain accurate even if the camera distance or bee size changes.

### 10. Shape Filtering (`ApplyShapeFiltering`)
- **Logic:** Discards "noise" blobs with Area < 50 pixels. Remaining blobs are counted using the standard area heuristic.

### 11. Neighbor Merge Pass (`ApplyNeighborMerge`)
- **Logic:** Calculates the centroid of each blob. If two centroids are closer than 30 pixels, they are treated as a single entity (fragment merging), even if not physically touching.

### 12. Motion Direction Consistency (`ApplyMotionConsistency`)
- **Logic:** Currently utilizes the high-precision shape filtering logic as a foundation for motion-based grouping.

### 13. Temporal Blob Stabilization (`ApplyTemporalStabilisation`)
- **Logic:** Currently utilizes the high-precision shape filtering logic to provide a stable, noise-free count.

### 14. Multi-Stage Pipeline (`ApplyAdvancedPipeline`)
- **Recommended Strategy:** Combines Morphological Repair, Shape Filtering, and Dynamic Area Estimation into a single high-performance pipeline.
- **Workflow:** `Repair -> Filter Noise -> Calculate Dynamic Median -> Count`.
