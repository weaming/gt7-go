package lap

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/weaming/gt7-go/internal/models"
)

func TestLoadLapsFromFileDeduplicatesAndRewrites(t *testing.T) {
	t.Parallel()

	dataPath := filepath.Join(t.TempDir(), "laps.json")
	laps := []*models.Lap{
		{
			Title:         "1:23.456",
			LapTicks:      5000,
			LapFinishTime: 83456,
			Number:        1,
			CarID:         100,
			CarName:       "Car A",
			CircuitID:     "track-a",
			DataSpeed:     []float64{10, 20, 30},
		},
		{
			Title:         "1:23.456",
			LapTicks:      5000,
			LapFinishTime: 83456,
			Number:        1,
			CarID:         100,
			CarName:       "Car A",
			CircuitID:     "track-a",
			DataSpeed:     []float64{10, 20, 30},
			TimeDiff: &models.TimeDiffResult{
				Distance:  []float64{0, 1},
				Timedelta: []float64{0, 5},
			},
		},
		{
			Title:         "1:24.000",
			LapTicks:      5040,
			LapFinishTime: 84000,
			Number:        2,
			CarID:         100,
			CarName:       "Car A",
			CircuitID:     "track-a",
			DataSpeed:     []float64{11, 21, 31},
		},
	}

	data, err := json.Marshal(laps)
	if err != nil {
		t.Fatalf("marshal laps: %v", err)
	}
	if err := os.WriteFile(dataPath, data, 0644); err != nil {
		t.Fatalf("write laps file: %v", err)
	}

	manager := NewManager(nil)
	if err := manager.LoadLapsFromFile(dataPath); err != nil {
		t.Fatalf("load laps: %v", err)
	}

	loadedLaps := manager.GetLaps()
	if len(loadedLaps) != 2 {
		t.Fatalf("loaded laps = %d, want 2", len(loadedLaps))
	}

	rewrittenData, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("read rewritten laps file: %v", err)
	}

	var rewrittenLaps []*models.Lap
	if err := json.Unmarshal(rewrittenData, &rewrittenLaps); err != nil {
		t.Fatalf("unmarshal rewritten laps: %v", err)
	}
	if len(rewrittenLaps) != 2 {
		t.Fatalf("rewritten laps = %d, want 2", len(rewrittenLaps))
	}
}

func TestClearAllLapDataClearsCurrentAndStoredLaps(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	currentLapPath := filepath.Join(tempDir, "current_lap.jsonl")
	manager := NewManager(nil)
	manager.SetCurrentLapSavePath(currentLapPath)
	manager.LoadLaps([]*models.Lap{
		{
			Title:         "1:23.456",
			LapTicks:      5000,
			LapFinishTime: 83456,
			Number:        1,
			CarID:         100,
			CarName:       "Car A",
			CircuitID:     "track-a",
		},
	}, true)
	manager.StartNewLap(100, 3, 2, 70, "Car A", "track-a", "Track A", "", true)
	if err := manager.LogData(100, 50, 0, 7000, 3, []float64{1, 1, 1, 1}, 0, 0, 1, 2, 3, []float64{80, 80, 80, 80}, 2); err != nil {
		t.Fatalf("log data: %v", err)
	}

	if err := manager.ClearAllLapData(); err != nil {
		t.Fatalf("clear all lap data: %v", err)
	}
	if got := len(manager.GetLaps()); got != 0 {
		t.Fatalf("laps after clear = %d, want 0", got)
	}
	if cur := manager.GetCurrentLapState(); cur != nil {
		t.Fatalf("current lap after clear = %#v, want nil", cur)
	}
	data, err := os.ReadFile(currentLapPath)
	if err != nil {
		t.Fatalf("read current lap file: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("current lap file length = %d, want 0", len(data))
	}
}

func TestHandleLapTransitionSavesLapWhenEnteringFinalLap(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil)
	manager.StartNewLap(100, 15, 14, 70, "Car A", "track-a", "Track A", "", true)
	manager.SetPreviousLapNum(14)
	logCompleteLapPositions(t, manager, 14, 120)

	lap := manager.HandleLapTransition(15, 90000, 60, 100, "Car A", true)
	if lap == nil {
		t.Fatal("lap = nil, want completed lap")
	}
	if lap.Number != 14 {
		t.Fatalf("lap number = %d, want 14", lap.Number)
	}
	if !lap.IsComplete {
		t.Fatal("lap is_complete = false, want true")
	}
	if got := len(manager.GetLaps()); got != 1 {
		t.Fatalf("stored laps = %d, want 1", got)
	}
}

func TestRankableLapRequiresCompleteNonPitLap(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil)
	manager.LoadLaps([]*models.Lap{
		{
			Title:              "0:58.000",
			LapTicks:           3480,
			LapFinishTime:      58000,
			Number:             1,
			CarName:            "Car A",
			CircuitID:          "track-a",
			StartsAtTrackStart: true,
			IsComplete:         false,
			DataPositionX:      []float64{0, 100},
			DataPositionY:      []float64{0, 0},
			DataPositionZ:      []float64{0, 100},
		},
		{
			Title:              "1:01.000",
			LapTicks:           3660,
			LapFinishTime:      61000,
			Number:             2,
			CarName:            "Car A",
			CircuitID:          "track-a",
			StartsAtTrackStart: true,
			IsComplete:         true,
			DataPositionX:      []float64{0, 10},
			DataPositionY:      []float64{0, 0},
			DataPositionZ:      []float64{0, 10},
		},
		{
			Title:              "1:00.000",
			LapTicks:           3600,
			LapFinishTime:      60000,
			Number:             3,
			CarName:            "Car A",
			CircuitID:          "track-a",
			StartsAtTrackStart: true,
			IsPitLap:           true,
			IsComplete:         true,
			DataPositionX:      []float64{0, 10},
			DataPositionY:      []float64{0, 0},
			DataPositionZ:      []float64{0, 10},
		},
	}, true)

	best := manager.GetBestLapForCircuit("track-a", "Car A")
	if best == nil {
		t.Fatal("best lap = nil, want rankable lap")
	}
	if best.Number != 2 {
		t.Fatalf("best lap number = %d, want 2", best.Number)
	}
}

func TestIncompleteLapUsesLastTickDistanceFromStart(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil)
	manager.StartNewLap(100, 3, 1, 70, "Car A", "track-a", "Track A", "", true)
	manager.SetPreviousLapNum(1)
	logIncompleteLapPositions(t, manager, 1, 120)

	lap := manager.HandleLapTransition(2, 60000, 60, 100, "Car A", true)
	if lap == nil {
		t.Fatal("lap = nil, want completed transition")
	}
	if lap.IsComplete {
		t.Fatal("lap is_complete = true, want false")
	}
	if IsRankableLap(lap) {
		t.Fatal("incomplete lap is rankable")
	}
}

func TestLapStartedAwayFromTrackStartIsIncomplete(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil)
	manager.StartNewLap(100, 3, 1, 70, "Car A", "track-a", "Track A", "", false)
	manager.SetPreviousLapNum(1)
	logCompleteLapPositions(t, manager, 1, 120)

	lap := manager.HandleLapTransition(2, 60000, 60, 100, "Car A", true)
	if lap == nil {
		t.Fatal("lap = nil, want completed transition")
	}
	if lap.StartsAtTrackStart {
		t.Fatal("starts_at_track_start = true, want false")
	}
	if lap.IsComplete {
		t.Fatal("lap is_complete = true, want false")
	}
	if IsRankableLap(lap) {
		t.Fatal("mid-start lap is rankable")
	}
}

func TestLapWithoutPositionDataIsIncomplete(t *testing.T) {
	t.Parallel()

	lap := &models.Lap{
		LapTicks:           3600,
		LapFinishTime:      60000,
		StartsAtTrackStart: true,
		IsComplete:         true,
	}

	if IsCompleteLap(lap) {
		t.Fatal("lap without position data is complete")
	}
}

func logCompleteLapPositions(t *testing.T, manager *Manager, lapNumber int16, ticks int) {
	t.Helper()

	for tick := range ticks {
		angle := 2 * float64(tick) / float64(ticks-1) * 3.141592653589793
		posX := 100 * (1 - math.Cos(angle))
		posZ := 100 * math.Sin(angle)
		if err := manager.LogData(100, 80, 0, 7000, 3, []float64{1, 1, 1, 1}, 0, 0, posX, 0, posZ, []float64{80, 80, 80, 80}, lapNumber); err != nil {
			t.Fatalf("log data: %v", err)
		}
	}
}

func logIncompleteLapPositions(t *testing.T, manager *Manager, lapNumber int16, ticks int) {
	t.Helper()

	for tick := range ticks {
		posX := float64(tick) * 2
		posZ := float64(tick)
		if err := manager.LogData(100, 80, 0, 7000, 3, []float64{1, 1, 1, 1}, 0, 0, posX, 0, posZ, []float64{80, 80, 80, 80}, lapNumber); err != nil {
			t.Fatalf("log data: %v", err)
		}
	}
}
