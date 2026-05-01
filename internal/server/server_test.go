package server

import (
	"testing"
	"time"

	"github.com/weaming/gt7-go/internal/models"
)

func TestLapArchiveFilenameUsesFirstTickAndStaysStableAfterAppend(t *testing.T) {
	t.Parallel()

	firstTick := time.Date(2026, 5, 1, 10, 11, 12, 345*int(time.Millisecond), time.UTC)
	secondTick := firstTick.Add(2 * time.Minute)
	thirdTick := firstTick.Add(4 * time.Minute)

	laps := []*models.Lap{
		newArchiveTestLap(2, secondTick),
		newArchiveTestLap(1, firstTick),
	}
	archive := lapArchive{
		Version: 1,
		SavedAt: archiveStartTime(laps),
		Label:   buildLapArchiveLabel(laps),
		Laps:    laps,
	}
	filename, err := buildLapArchiveFilename(archive)
	if err != nil {
		t.Fatalf("build filename: %v", err)
	}

	appendedLaps := append([]*models.Lap{newArchiveTestLap(3, thirdTick)}, laps...)
	appendedArchive := lapArchive{
		Version: 1,
		SavedAt: archiveStartTime(appendedLaps),
		Label:   buildLapArchiveLabel(appendedLaps),
		Laps:    appendedLaps,
	}
	appendedFilename, err := buildLapArchiveFilename(appendedArchive)
	if err != nil {
		t.Fatalf("build appended filename: %v", err)
	}

	if appendedFilename != filename {
		t.Fatalf("appended filename = %q, want %q", appendedFilename, filename)
	}

	wantFilename := "20260501_1011__TrackA__CarA.json"
	if filename != wantFilename {
		t.Fatalf("filename = %q, want %q", filename, wantFilename)
	}
	if !appendedArchive.SavedAt.Equal(firstTick) {
		t.Fatalf("archive start = %s, want %s", appendedArchive.SavedAt, firstTick)
	}
}

func TestLapArchiveFilenameCompactsMixedTrackAndLongCar(t *testing.T) {
	t.Parallel()

	firstTick := time.Date(2026, 5, 1, 15, 32, 43, 224*int(time.Millisecond), time.UTC)
	laps := []*models.Lap{
		{
			Title:             "1:30.000",
			LapTicks:          5400,
			LapFinishTime:     90000,
			Number:            1,
			CarID:             100,
			CarName:           "SF19 Super Formula Toyota '19",
			CircuitID:         "track-a",
			CircuitName:       "Track A",
			LapStartTimestamp: &firstTick,
		},
		{
			Title:             "1:31.000",
			LapTicks:          5460,
			LapFinishTime:     91000,
			Number:            2,
			CarID:             100,
			CarName:           "SF19 Super Formula Toyota '19",
			CircuitID:         "track-b",
			CircuitName:       "Track B",
			LapStartTimestamp: &firstTick,
		},
	}
	archive := lapArchive{
		Version: 1,
		SavedAt: archiveStartTime(laps),
		Label:   buildLapArchiveLabel(laps),
		Laps:    laps,
	}

	filename, err := buildLapArchiveFilename(archive)
	if err != nil {
		t.Fatalf("build filename: %v", err)
	}

	wantFilename := "20260501_1532__mixed__SF19Toyota19.json"
	if filename != wantFilename {
		t.Fatalf("filename = %q, want %q", filename, wantFilename)
	}
}

func newArchiveTestLap(number int, startTime time.Time) *models.Lap {
	endTime := startTime.Add(90 * time.Second)
	return &models.Lap{
		Title:             "1:30.000",
		LapTicks:          5400,
		LapFinishTime:     90000,
		Number:            number,
		CarID:             100,
		CarName:           "Car A",
		CircuitID:         "track-a",
		CircuitName:       "Track A",
		LapStartTimestamp: &startTime,
		LapEndTimestamp:   &endTime,
	}
}
