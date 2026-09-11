package logic

import (
	"math"
)

// ApplyTracking handles blob-to-track assignment based on the selected method
func (p *Processor) ApplyTracking(blobs []Blob, method int) int {
	p.FrameCount++

	switch method {
	case 1:
		p.trackFeatureConsensus(blobs)
	case 2:
		p.trackHungarianSimplified(blobs)
	case 3:
		p.trackKalmanPredictive(blobs)
	case 4:
		p.trackMultiFeature(blobs)
	case 5:
		p.trackMotionGated(blobs)
	case 6:
		p.trackPersistent(blobs)
	default:
		// No tracking or invalid method - return count of blobs as default fallback
		return len(blobs)
	}

	maxMissing := 3
	if method == 6 {
		maxMissing = 30 // Keep tracks alive for 30 frames
	}
	p.cleanupDeadTracks(maxMissing)
	return len(p.Tracks)
}

// 1. Feature Consensus Tracker (Contextual Cost Function)
func (p *Processor) trackFeatureConsensus(blobs []Blob) {
	matchedBlobs := make([]bool, len(blobs))

	for i := range p.Tracks {
		track := &p.Tracks[i]
		bestCost := 0.6 // Rejection threshold for poor affinity matches
		bestIdx := -1

		for j, blob := range blobs {
			if matchedBlobs[j] {
				continue
			}

			cost := p.calculateAffinities(track, blob)
			if cost < bestCost {
				bestCost = cost
				bestIdx = j
			}
		}

		if bestIdx != -1 {
			p.updateTrack(track, blobs[bestIdx])
			matchedBlobs[bestIdx] = true
		} else {
			// Coasting: keep moving towards the predicted velocity naturally
			track.Centroid.X += int(track.VX)
			track.Centroid.Y += int(track.VY)
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

// 6. Persistent Tracker (High Persistence + Path Length Priority)
func (p *Processor) trackPersistent(blobs []Blob) {
	type Match struct {
		trackIdx int
		blobIdx  int
		dist     float64
		pathLen  int
	}
	var matches []Match

	maxDist := 60.0

	for i, track := range p.Tracks {
		for j, blob := range blobs {
			dist := p.euclideanDistance(track.Centroid, blob.Centroid)
			if dist < maxDist {
				matches = append(matches, Match{
					trackIdx: i,
					blobIdx:  j,
					dist:     dist,
					pathLen:  len(track.History),
				})
			}
		}
	}

	// Sort matches:
	// 1. Prioritize longer paths (pathLen descending)
	// 2. Then prioritize shorter distances (dist ascending)
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			higherPriority := false
			if matches[i].pathLen < matches[j].pathLen {
				higherPriority = true
			} else if matches[i].pathLen == matches[j].pathLen && matches[i].dist > matches[j].dist {
				higherPriority = true
			}

			if higherPriority {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	matchedTracks := make([]bool, len(p.Tracks))
	matchedBlobs := make([]bool, len(blobs))

	for _, m := range matches {
		if !matchedTracks[m.trackIdx] && !matchedBlobs[m.blobIdx] {
			p.updateTrack(&p.Tracks[m.trackIdx], blobs[m.blobIdx])
			matchedTracks[m.trackIdx] = true
			matchedBlobs[m.blobIdx] = true
		}
	}

	// Create new tracks for unmatched blobs
	p.createNewTracks(blobs, matchedBlobs)
}

// --- Internal Track Management ---

func (p *Processor) calculateAffinities(track *Track, blob Blob) float64 {
	// Distancia Euclidiana (40%): Proximidad al centroide predicho.
	predictedX := float64(track.Centroid.X) + track.VX
	predictedY := float64(track.Centroid.Y) + track.VY
	dist := math.Sqrt(math.Pow(predictedX-float64(blob.Centroid.X), 2) + math.Pow(predictedY-float64(blob.Centroid.Y), 2))

	distScore := dist / 60.0
	if distScore > 1.0 {
		distScore = 1.0
	}

	// Consistencia de Velocidad (30%): Similitud entre el vector (VX, VY) previo y el actual
	vxDiff := float64(blob.Centroid.X-track.Centroid.X) - track.VX
	vyDiff := float64(blob.Centroid.Y-track.Centroid.Y) - track.VY
	speedDiff := math.Sqrt(vxDiff*vxDiff + vyDiff*vyDiff)

	speedScore := speedDiff / 40.0
	if speedScore > 1.0 {
		speedScore = 1.0
	}

	// Consistencia Morfológica (30%): Diferencia porcentual de Area y Solidity
	areaDiff := math.Abs(float64(track.Area-blob.Area)) / math.Max(float64(track.Area), 1.0)
	if areaDiff > 1.0 {
		areaDiff = 1.0
	}

	solidityDiff := math.Abs(track.Solidity - blob.Solidity)
	if solidityDiff > 1.0 {
		solidityDiff = 1.0
	}

	morphScore := (areaDiff + solidityDiff) / 2.0

	return (distScore * 0.4) + (speedScore * 0.3) + (morphScore * 0.3)
}

func (p *Processor) updateTrack(track *Track, blob Blob) {
	oldY := track.Centroid.Y
	midY := p.height / 2

	track.Hits++
	if track.Hits >= 5 {
		track.Status = "confirmed"
	}

	// Crossover Detection (Horizontal line in middle)
	if track.Status == "confirmed" {
		if oldY >= midY && blob.Centroid.Y < midY {
			p.CountUp++
		} else if oldY <= midY && blob.Centroid.Y > midY {
			p.CountDown++
		}
	}

	// Calculate velocity
	track.VX = float64(blob.Centroid.X - track.Centroid.X)
	track.VY = float64(blob.Centroid.Y - track.Centroid.Y)

	track.Centroid = blob.Centroid
	track.Area = blob.Area
	track.Solidity = blob.Solidity
	track.Ratio = blob.Ratio
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
				Status:        "tentative",
				Hits:          1,
				Centroid:      blob.Centroid,
				Area:          blob.Area,
				Solidity:      blob.Solidity,
				Ratio:         blob.Ratio,
				LastSeenFrame: p.FrameCount,
				History:       []Point{blob.Centroid},
				MarkerID:      MarkerUnknown,
			})
			p.NextTrackID++
		}
	}
}

func (p *Processor) cleanupDeadTracks(maxMissing int) {
	var activeTracks []Track
	for _, track := range p.Tracks {
		limit := maxMissing
		if track.Status == "confirmed" {
			limit = 30 // Coasting para tracks confirmados
		}
		if p.FrameCount-track.LastSeenFrame < limit {
			activeTracks = append(activeTracks, track)
		}
	}
	p.Tracks = activeTracks
}
