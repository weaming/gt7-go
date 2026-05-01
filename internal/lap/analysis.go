package lap

import (
	"math"
	"sort"

	"github.com/weaming/gt7-go/internal/models"
)

// ComputeAllTimeDiffs computes time differences for all laps against the best
// lap and sets the TimeDiff field on each lap.
func ComputeAllTimeDiffs(laps []*models.Lap) {
	var best *models.Lap
	var bestTime int64 = 1<<63 - 1
	for _, l := range laps {
		if IsRankableLap(l) && l.LapFinishTime < bestTime {
			bestTime = l.LapFinishTime
			best = l
		}
	}
	if best == nil {
		return
	}
	for _, l := range laps {
		if l == best {
			l.TimeDiff = nil
			continue
		}
		if !IsCompleteLap(l) {
			l.TimeDiff = nil
			continue
		}
		if len(l.DataPositionX) < 2 || len(l.DataPositionZ) < 2 {
			l.TimeDiff = nil
			continue
		}
		l.TimeDiff = ComputeTimeDiff(best, l)
	}
}

func IsRankableLap(lap *models.Lap) bool {
	if lap == nil || lap.IsPitLap || lap.LapFinishTime <= 0 {
		return false
	}
	return IsCompleteLap(lap)
}

func IsCompleteLap(lap *models.Lap) bool {
	if lap == nil || lap.LapFinishTime <= 0 {
		return false
	}
	if len(lap.DataPositionX) >= 2 && len(lap.DataPositionY) >= 2 && len(lap.DataPositionZ) >= 2 {
		return isLapComplete(lap.DataPositionX, lap.DataPositionY, lap.DataPositionZ)
	}
	return false
}

// ComputeTimeDiff computes the time difference between a comparison lap and
// a reference lap using direct position matching: for each point along the
// reference lap's track, find the nearest corresponding position in the
// comparison lap and compare their elapsed times. This is more accurate than
// cumulative-distance matching because it handles different racing lines.
func ComputeTimeDiff(ref, comp *models.Lap) *models.TimeDiffResult {
	if len(ref.DataPositionX) < 10 || len(comp.DataPositionX) < 10 {
		return nil
	}

	refDist := trackDistance(ref.DataPositionX, ref.DataPositionZ)
	compDist := trackDistance(comp.DataPositionX, comp.DataPositionZ)
	if refDist == nil || compDist == nil {
		return nil
	}

	totalDist := refDist[len(refDist)-1]
	compTotalDist := compDist[len(compDist)-1]
	maxDist := math.Min(totalDist, compTotalDist)
	if maxDist <= 0 {
		return nil
	}

	numPoints := 500
	dist := make([]float64, numPoints)
	delta := make([]float64, numPoints)

	// Precompute comparison position array for nearest-neighbor search.
	// Track length ratio guides the initial search position.
	ratio := float64(len(comp.DataPositionX)) / float64(len(ref.DataPositionX))

	for i := range numPoints {
		d := maxDist * float64(i) / float64(numPoints-1)
		dist[i] = d

		// Time on reference lap at this distance
		refTime := interpTimeAtDist(refDist, d)

		// Find the corresponding index in the reference lap
		refIdx := sort.SearchFloat64s(refDist, d)
		if refIdx >= len(refDist) {
			refIdx = len(refDist) - 1
		}

		// Estimate starting position in comparison lap using track ratio
		estIdx := int(float64(refIdx) * ratio)
		if estIdx >= len(comp.DataPositionX) {
			estIdx = len(comp.DataPositionX) - 1
		}

		// Nearest-neighbor search within a ±200 index window
		searchHalf := 200
		lo := max(0, estIdx-searchHalf)
		hi := min(len(comp.DataPositionX)-1, estIdx+searchHalf)

		rx, rz := ref.DataPositionX[refIdx], ref.DataPositionZ[refIdx]
		bestIdx := estIdx
		bestDistSq := math.MaxFloat64
		for j := lo; j <= hi; j++ {
			dx := comp.DataPositionX[j] - rx
			dz := comp.DataPositionZ[j] - rz
			d2 := dx*dx + dz*dz
			if d2 < bestDistSq {
				bestDistSq = d2
				bestIdx = j
			}
		}

		// Time on comparison lap at the matched position
		compTime := float64(bestIdx) / 60.0

		if refTime >= 0 && compTime >= 0 {
			delta[i] = (compTime - refTime) * 1000
		}
	}

	return &models.TimeDiffResult{
		Distance:  dist,
		Timedelta: delta,
	}
}

// trackDistance computes cumulative 2D distance along the track from XZ
// position data. Returns meters.
func trackDistance(posX, posZ []float64) []float64 {
	n := len(posX)
	if nz := len(posZ); nz < n {
		n = nz
	}
	if n < 2 {
		return nil
	}
	dist := make([]float64, n)
	for i := 1; i < n; i++ {
		dx := posX[i] - posX[i-1]
		dz := posZ[i] - posZ[i-1]
		dist[i] = dist[i-1] + math.Sqrt(dx*dx+dz*dz)
	}
	return dist
}

// interpTimeAtDist interpolates the time (in seconds, assuming 60fps) at a
// given cumulative track distance. Returns -1 if distance is out of range.
func interpTimeAtDist(dist []float64, target float64) float64 {
	if len(dist) == 0 {
		return -1
	}
	if target <= dist[0] {
		return 0
	}
	last := len(dist) - 1
	if target >= dist[last] {
		return float64(last) / 60.0
	}

	lo := sort.SearchFloat64s(dist, target)
	if lo <= 0 {
		return 0
	}
	if lo >= len(dist) {
		return float64(last) / 60.0
	}

	frac := (target - dist[lo-1]) / (dist[lo] - dist[lo-1])
	return (float64(lo-1) + frac) / 60.0
}
