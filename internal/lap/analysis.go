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

const (
	timeDiffMaxPoints        = 500
	timeDiffReferenceWindow  = 1200
	timeDiffReferenceLookout = 300
)

// ComputeTimeDiff computes the time difference between a comparison lap and
// a reference lap by anchoring each comparison point to the nearest plausible
// reference position. Matching is monotonic so a nearby crossing or the
// start/finish closure cannot jump backward to another segment.
func ComputeTimeDiff(ref, comp *models.Lap) *models.TimeDiffResult {
	if len(ref.DataPositionX) < 10 || len(comp.DataPositionX) < 10 {
		return nil
	}

	refDist := trackDistance(ref.DataPositionX, ref.DataPositionZ)
	compDist := trackDistance(comp.DataPositionX, comp.DataPositionZ)
	if refDist == nil || compDist == nil {
		return nil
	}

	if refDist[len(refDist)-1] <= 0 || compDist[len(compDist)-1] <= 0 {
		return nil
	}

	sampleIndices := timeDiffSampleIndices(len(compDist), timeDiffMaxPoints)
	dist := make([]float64, len(sampleIndices))
	delta := make([]float64, len(sampleIndices))
	lastRefIdx := 0

	for i, compIdx := range sampleIndices {
		dist[i] = compDist[compIdx]

		expectedRefIdx := sort.SearchFloat64s(refDist, compDist[compIdx])
		if expectedRefIdx >= len(refDist) {
			expectedRefIdx = len(refDist) - 1
		}

		lo := max(lastRefIdx, expectedRefIdx-timeDiffReferenceWindow)
		hi := min(len(refDist)-1, expectedRefIdx+timeDiffReferenceWindow)
		if hi < lo {
			hi = min(len(refDist)-1, lo+timeDiffReferenceLookout)
		}

		refIdx := nearestReferenceIndex(
			ref.DataPositionX,
			ref.DataPositionZ,
			comp.DataPositionX[compIdx],
			comp.DataPositionZ[compIdx],
			lo,
			hi,
		)
		lastRefIdx = refIdx
		delta[i] = (float64(compIdx) - float64(refIdx)) * 1000.0 / 60.0
	}

	return &models.TimeDiffResult{
		Distance:  dist,
		Timedelta: delta,
	}
}

func timeDiffSampleIndices(length, maxPoints int) []int {
	if length <= 0 || maxPoints <= 0 {
		return nil
	}
	if length <= maxPoints {
		indices := make([]int, length)
		for i := range length {
			indices[i] = i
		}
		return indices
	}

	indices := make([]int, maxPoints)
	last := length - 1
	for i := range maxPoints {
		indices[i] = int(math.Round(float64(last) * float64(i) / float64(maxPoints-1)))
	}
	return indices
}

func nearestReferenceIndex(posX, posZ []float64, targetX, targetZ float64, lo, hi int) int {
	if len(posX) == 0 || len(posZ) == 0 {
		return 0
	}

	limit := min(len(posX), len(posZ)) - 1
	lo = max(0, min(lo, limit))
	hi = max(lo, min(hi, limit))

	bestIdx := lo
	bestDistSq := math.MaxFloat64
	for i := lo; i <= hi; i++ {
		dx := posX[i] - targetX
		dz := posZ[i] - targetZ
		distSq := dx*dx + dz*dz
		if distSq < bestDistSq {
			bestDistSq = distSq
			bestIdx = i
		}
	}
	return bestIdx
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
