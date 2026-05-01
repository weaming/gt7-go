package telemetry

import (
	"testing"

	tmodel "github.com/weaming/gt7-go/internal/models"
)

func TestTelemetryDataStateNormalRaceAlwaysSavesCompletedLap(t *testing.T) {
	t.Parallel()

	state := newTelemetryDataState(&tmodel.TelemetrySnapshot{
		InRace:     true,
		CurrentLap: 2,
	}, false)

	if !state.canStartLap {
		t.Fatal("canStartLap = false, want true")
	}
	if !state.canAdvanceLap {
		t.Fatal("canAdvanceLap = false, want true")
	}
	if !state.canAppendCurrentLapTick {
		t.Fatal("canAppendCurrentLapTick = false, want true")
	}
	if !state.shouldSaveCompletedLap {
		t.Fatal("shouldSaveCompletedLap = false, want true")
	}
}

func TestTelemetryDataStateReplaySavesCompletedLapOnlyWhenEnabled(t *testing.T) {
	t.Parallel()

	replaySnapshot := &tmodel.TelemetrySnapshot{
		InRace:     true,
		CurrentLap: 2,
		IsReplay:   true,
	}

	previewState := newTelemetryDataState(replaySnapshot, false)
	if previewState.shouldSaveCompletedLap {
		t.Fatal("preview replay shouldSaveCompletedLap = true, want false")
	}
	if !previewState.canAppendCurrentLapTick {
		t.Fatal("preview replay canAppendCurrentLapTick = false, want true")
	}

	recordState := newTelemetryDataState(replaySnapshot, true)
	if !recordState.shouldSaveCompletedLap {
		t.Fatal("recording replay shouldSaveCompletedLap = false, want true")
	}
}

func TestTelemetryDataStatePausedOrCompleteDoesNotAppendTicks(t *testing.T) {
	t.Parallel()

	pausedState := newTelemetryDataState(&tmodel.TelemetrySnapshot{
		InRace:     true,
		CurrentLap: 2,
		IsPaused:   true,
	}, false)
	if pausedState.canAdvanceLap || pausedState.canAppendCurrentLapTick {
		t.Fatal("paused state advanced or appended ticks")
	}
	if !pausedState.canStartLap {
		t.Fatal("paused canStartLap = false, want true")
	}

	completeState := newTelemetryDataState(&tmodel.TelemetrySnapshot{
		InRace:         true,
		CurrentLap:     2,
		IsRaceComplete: true,
	}, false)
	if completeState.canStartLap || completeState.canAppendCurrentLapTick {
		t.Fatal("complete state started or appended ticks")
	}
	if !completeState.canAdvanceLap {
		t.Fatal("complete canAdvanceLap = false, want true")
	}
}
