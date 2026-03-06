package logic

import (
	"math"
	"math/rand"
)

// ApplyKMeans applies k-means clustering to white pixels in the binary frame
// and colors each cluster with a random color
func (p *Processor) ApplyKMeans(binaryFrame []byte, k int) ([]byte, []Blob) {
	// Extract white pixel coordinates
	whitePixels := p.extractWhitePixels(binaryFrame)

	if len(whitePixels) < k {
		// Not enough white pixels for clustering
		return binaryFrame, nil
	}

	// Perform k-means clustering
	clusters := p.kMeansClustering(whitePixels, k)

	// Group points by cluster
	clusterPoints := make([][]Point, k)
	for i, point := range whitePixels {
		clusterID := clusters[i]
		clusterPoints[clusterID] = append(clusterPoints[clusterID], point)
	}

	// Extract cluster blobs
	blobs := p.ExtractBlobs(clusterPoints)

	// Create output frame
	outputFrame := make([]byte, len(binaryFrame))
	copy(outputFrame, binaryFrame)

	for i, blob := range blobs {
		color := p.GetVibrantColor(i)
		for _, point := range blob.Points {
			pixelIndex := (point.Y*p.width + point.X) * p.bytesPP
			outputFrame[pixelIndex] = color[0]
			outputFrame[pixelIndex+1] = color[1]
			outputFrame[pixelIndex+2] = color[2]
		}
	}

	return outputFrame, blobs
}

// extractWhitePixels returns coordinates of all white pixels in the frame
func (p *Processor) extractWhitePixels(frame []byte) []Point {
	var pixels []Point

	for y := 0; y < p.height; y++ {
		for x := 0; x < p.width; x++ {
			i := (y*p.width + x) * p.bytesPP
			if frame[i] == 255 && frame[i+1] == 255 && frame[i+2] == 255 {
				pixels = append(pixels, Point{X: x, Y: y})
			}
		}
	}

	return pixels
}

// kMeansClustering performs k-means clustering on points
func (p *Processor) kMeansClustering(points []Point, k int) []int {
	if len(points) == 0 {
		return []int{}
	}

	// Initialize centroids randomly from existing points
	centroids := make([]Point, k)
	perm := rand.Perm(len(points))
	for i := 0; i < k; i++ {
		centroids[i] = points[perm[i]]
	}

	// Cluster assignments
	assignments := make([]int, len(points))

	// Iterate until convergence (max 10 iterations for performance)
	maxIterations := 10
	for iter := 0; iter < maxIterations; iter++ {
		changed := false

		// Assign each point to nearest centroid
		for i, point := range points {
			nearestCluster := 0
			minDist := math.MaxFloat64

			for j, centroid := range centroids {
				dist := p.euclideanDistance(point, centroid)
				if dist < minDist {
					minDist = dist
					nearestCluster = j
				}
			}

			if assignments[i] != nearestCluster {
				assignments[i] = nearestCluster
				changed = true
			}
		}

		// If no changes, we've converged
		if !changed {
			break
		}

		// Update centroids
		clusterSums := make([]Point, k)
		clusterCounts := make([]int, k)

		for i, point := range points {
			cluster := assignments[i]
			clusterSums[cluster].X += point.X
			clusterSums[cluster].Y += point.Y
			clusterCounts[cluster]++
		}

		for i := 0; i < k; i++ {
			if clusterCounts[i] > 0 {
				centroids[i].X = clusterSums[i].X / clusterCounts[i]
				centroids[i].Y = clusterSums[i].Y / clusterCounts[i]
			}
		}
	}

	return assignments
}

// euclideanDistance calculates the Euclidean distance between two points
func (p *Processor) euclideanDistance(a, b Point) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Sqrt(dx*dx + dy*dy)
}
