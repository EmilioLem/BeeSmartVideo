package logic

import (
	"math"
)

// ApplyTracking handles blob-to-track assignment based on the selected method
func (p *Processor) ApplyTracking(blobs []Blob, method int) int {
	p.FrameCount++

	switch method {
	case 1:
		p.trackNearestCentroid(blobs)
	case 2:
		p.trackHungarianSimplified(blobs)
	case 3:
		p.trackKalmanPredictive(blobs)
	case 4:
		p.trackMultiFeature(blobs)
	case 5:
		p.trackMotionGated(blobs)
	default:
		// No tracking or invalid method - return count of blobs as default fallback
		return len(blobs)
	}

	p.cleanupDeadTracks(10) // Release tracks not seen for 10 frames
	return len(p.Tracks)
}

// 1. Nearest-Centroid Tracker (Baseline)
func (p *Processor) trackNearestCentroid(blobs []Blob) {
	matchedBlobs := make([]bool, len(blobs))

	for i := range p.Tracks {
		track := &p.Tracks[i]
		bestDist := 50.0 // Threshold in pixels
		bestIdx := -1

		for j, blob := range blobs {
			if matchedBlobs[j] {
				continue
			}
			dist := p.euclideanDistance(track.Centroid, blob.Centroid)
			if dist < bestDist {
				bestDist = dist
				bestIdx = j
			}
		}

		if bestIdx != -1 {
			p.updateTrack(track, blobs[bestIdx])
			matchedBlobs[bestIdx] = true
		}
	}

	// Create new tracks for unmatched blobs
	p.createNewTracks(blobs, matchedBlobs)
}

// 2. Hungarian Assignment Tracker (Simplified Greedy Version)
// Note: A full O(n^3) Hungarian algorithm is complex to implement from scratch.
// This greedy version with global sorting of costs mimics the "optimal" matching better than raw nearest.
func (p *Processor) trackHungarianSimplified(blobs []Blob) {
	type Pair struct {
		trackIdx int
		blobIdx  int
		cost     float64
	}
	var pairs []Pair

	for i, track := range p.Tracks {
		for j, blob := range blobs {
			dist := p.euclideanDistance(track.Centroid, blob.Centroid)
			if dist < 60.0 {
				pairs = append(pairs, Pair{i, j, dist})
			}
		}
	}

	// Sort pairs by cost (shortest distance first)
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].cost > pairs[j].cost {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	matchedTracks := make([]bool, len(p.Tracks))
	matchedBlobs := make([]bool, len(blobs))

	for _, pair := range pairs {
		if !matchedTracks[pair.trackIdx] && !matchedBlobs[pair.blobIdx] {
			p.updateTrack(&p.Tracks[pair.trackIdx], blobs[pair.blobIdx])
			matchedTracks[pair.trackIdx] = true
			matchedBlobs[pair.blobIdx] = true
		}
	}

	p.createNewTracks(blobs, matchedBlobs)
}

// 3. Kalman Filter Tracker (Prediction-Based)
func (p *Processor) trackKalmanPredictive(blobs []Blob) {
	matchedBlobs := make([]bool, len(blobs))

	for i := range p.Tracks {
		track := &p.Tracks[i]

		// Predict next position based on velocity
		predicted := Point{
			X: track.Centroid.X + int(track.VX),
			Y: track.Centroid.Y + int(track.VY),
		}

		bestDist := 50.0
		bestIdx := -1

		for j, blob := range blobs {
			if matchedBlobs[j] {
				continue
			}
			dist := p.euclideanDistance(predicted, blob.Centroid)
			if dist < bestDist {
				bestDist = dist
				bestIdx = j
			}
		}

		if bestIdx != -1 {
			// Update velocity based on actual movement
			dx := float64(blobs[bestIdx].Centroid.X - track.Centroid.X)
			dy := float64(blobs[bestIdx].Centroid.Y - track.Centroid.Y)
			track.VX = track.VX*0.3 + dx*0.7 // Smoothing
			track.VY = track.VY*0.3 + dy*0.7

			p.updateTrack(track, blobs[bestIdx])
			matchedBlobs[bestIdx] = true
		} else {
			// Coasting: keep velocity but mark as not updated
			track.Centroid.X += int(track.VX)
			track.Centroid.Y += int(track.VY)
		}
	}

	p.createNewTracks(blobs, matchedBlobs)
}

// 4. Multi-Feature Matching
func (p *Processor) trackMultiFeature(blobs []Blob) {
	matchedBlobs := make([]bool, len(blobs))

	for i := range p.Tracks {
		track := &p.Tracks[i]
		bestCost := 100.0 // Combined cost threshold
		bestIdx := -1

		for j, blob := range blobs {
			if matchedBlobs[j] {
				continue
			}

			dist := p.euclideanDistance(track.Centroid, blob.Centroid)
			areaDiff := math.Abs(float64(track.Area-blob.Area)) / float64(track.Area+1)

			// Weighted cost
			cost := dist*1.0 + areaDiff*500.0

			if cost < bestCost {
				bestCost = cost
				bestIdx = j
			}
		}

		if bestIdx != -1 {
			p.updateTrack(track, blobs[bestIdx])
			matchedBlobs[bestIdx] = true
		}
	}

	p.createNewTracks(blobs, matchedBlobs)
}

// 5. Motion-Gated Assignment
func (p *Processor) trackMotionGated(blobs []Blob) {
	matchedBlobs := make([]bool, len(blobs))
	maxSpeed := 40.0 // Pixels per frame max

	for i := range p.Tracks {
		track := &p.Tracks[i]
		bestDist := maxSpeed
		bestIdx := -1

		for j, blob := range blobs {
			if matchedBlobs[j] {
				continue
			}
			dist := p.euclideanDistance(track.Centroid, blob.Centroid)

			// Forbid assignment if too fast
			if dist < bestDist {
				bestDist = dist
				bestIdx = j
			}
		}

		if bestIdx != -1 {
			p.updateTrack(track, blobs[bestIdx])
			matchedBlobs[bestIdx] = true
		}
	}

	p.createNewTracks(blobs, matchedBlobs)
}

// --- Internal Track Management ---

func (p *Processor) updateTrack(track *Track, blob Blob) {
	track.Centroid = blob.Centroid
	track.Area = blob.Area
	track.LastSeenFrame = p.FrameCount
	track.History = append(track.History, blob.Centroid)
	if len(track.History) > 30 {
		track.History = track.History[1:]
	}
}

func (p *Processor) createNewTracks(blobs []Blob, matched []bool) {
	for i, blob := range blobs {
		if !matched[i] {
			p.Tracks = append(p.Tracks, Track{
				ID:            p.NextTrackID,
				Centroid:      blob.Centroid,
				Area:          blob.Area,
				LastSeenFrame: p.FrameCount,
				History:       []Point{blob.Centroid},
			})
			p.NextTrackID++
		}
	}
}

func (p *Processor) cleanupDeadTracks(maxMissing int) {
	var activeTracks []Track
	for _, track := range p.Tracks {
		if p.FrameCount-track.LastSeenFrame < maxMissing {
			activeTracks = append(activeTracks, track)
		}
	}
	p.Tracks = activeTracks
}
