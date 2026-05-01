package lap

import (
	"encoding/json"
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
	manager.StartNewLap(100, 3, 2, 70, "Car A", "track-a", "Track A", "")
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
