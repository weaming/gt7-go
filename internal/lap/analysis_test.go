package lap

import (
	"math"
	"testing"

	"github.com/weaming/gt7-go/internal/models"
)

func TestComputeTimeDiffMatchesCurrentPointToNearestReferencePosition(t *testing.T) {
	t.Parallel()

	ref := straightLap(1000, 0)
	comp := straightLap(1060, 60)

	diff := ComputeTimeDiff(ref, comp)
	if diff == nil {
		t.Fatal("diff = nil")
	}
	if len(diff.Distance) == 0 || len(diff.Distance) != len(diff.Timedelta) {
		t.Fatalf("invalid diff lengths: distance=%d timedelta=%d", len(diff.Distance), len(diff.Timedelta))
	}

	lastDelta := diff.Timedelta[len(diff.Timedelta)-1]
	if math.Abs(lastDelta-1000) > 1 {
		t.Fatalf("last delta = %.3fms, want about 1000ms", lastDelta)
	}
}

func TestComputeTimeDiffKeepsReferenceMatchMonotonic(t *testing.T) {
	t.Parallel()

	ref := &models.Lap{
		DataPositionX: []float64{0, 1, 2, 3, 4, 5, 4, 3, 2, 1, 0},
		DataPositionZ: []float64{0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
	}
	comp := &models.Lap{
		DataPositionX: []float64{0, 1, 2, 3, 4, 5, 4, 3, 2, 1, 0},
		DataPositionZ: []float64{0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
	}

	diff := ComputeTimeDiff(ref, comp)
	if diff == nil {
		t.Fatal("diff = nil")
	}
	for i, delta := range diff.Timedelta {
		if math.Abs(delta) > 0.001 {
			t.Fatalf("delta[%d] = %.3fms, want 0", i, delta)
		}
	}
}

func straightLap(length int, delayTicks int) *models.Lap {
	lap := &models.Lap{
		DataPositionX: make([]float64, length),
		DataPositionZ: make([]float64, length),
	}
	for i := range length {
		position := i - delayTicks
		if position < 0 {
			position = 0
		}
		if position > 999 {
			position = 999
		}
		lap.DataPositionX[i] = float64(position)
	}
	return lap
}
