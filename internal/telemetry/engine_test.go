package telemetry

import (
	"testing"

	tmodel "github.com/weaming/gt7-go/internal/models"
)

func TestTelemetryModeNormalRaceAlwaysSavesCompletedLap(t *testing.T) {
	t.Parallel()

	mode := newTelemetryMode(&tmodel.TelemetrySnapshot{
		InRace:     true,
		CurrentLap: 2,
	}, false)

	if mode.phase != telemetryPhaseRace {
		t.Fatalf("phase = %s, want %s", mode.phase, telemetryPhaseRace)
	}
	if !mode.CanStartLap() {
		t.Fatal("canStartLap = false, want true")
	}
	if !mode.CanAdvanceLap() {
		t.Fatal("canAdvanceLap = false, want true")
	}
	if !mode.CanAppendCurrentLapTick() {
		t.Fatal("canAppendCurrentLapTick = false, want true")
	}
	if !mode.ShouldSaveCompletedLap() {
		t.Fatal("shouldSaveCompletedLap = false, want true")
	}
}

func TestTelemetryModeReplaySavesCompletedLapOnlyWhenEnabled(t *testing.T) {
	t.Parallel()

	replaySnapshot := &tmodel.TelemetrySnapshot{
		InRace:     true,
		CurrentLap: 2,
		IsReplay:   true,
	}

	onlyMode := newTelemetryMode(replaySnapshot, false)
	if onlyMode.phase != telemetryPhaseReplayOnly {
		t.Fatalf("replay only phase = %s, want %s", onlyMode.phase, telemetryPhaseReplayOnly)
	}
	if onlyMode.ShouldSaveCompletedLap() {
		t.Fatal("replay only shouldSaveCompletedLap = true, want false")
	}
	if !onlyMode.CanAppendCurrentLapTick() {
		t.Fatal("replay only canAppendCurrentLapTick = false, want true")
	}

	recordMode := newTelemetryMode(replaySnapshot, true)
	if recordMode.phase != telemetryPhaseReplayAndRecord {
		t.Fatalf("record phase = %s, want %s", recordMode.phase, telemetryPhaseReplayAndRecord)
	}
	if !recordMode.ShouldSaveCompletedLap() {
		t.Fatal("recording replay shouldSaveCompletedLap = false, want true")
	}
}

func TestTelemetryModePauseKeepsRaceOrReplayContext(t *testing.T) {
	t.Parallel()

	pausedMode := newTelemetryMode(&tmodel.TelemetrySnapshot{
		InRace:     true,
		CurrentLap: 2,
		IsPaused:   true,
	}, false)
	if pausedMode.phase != telemetryPhaseRace {
		t.Fatalf("paused phase = %s, want %s", pausedMode.phase, telemetryPhaseRace)
	}
	if !pausedMode.isPaused {
		t.Fatal("paused mode isPaused = false, want true")
	}
	if pausedMode.CanAdvanceLap() || pausedMode.CanAppendCurrentLapTick() {
		t.Fatal("paused state advanced or appended ticks")
	}
	if !pausedMode.CanStartLap() {
		t.Fatal("paused canStartLap = false, want true")
	}

	pausedReplayMode := newTelemetryMode(&tmodel.TelemetrySnapshot{
		InRace:     true,
		CurrentLap: 2,
		IsReplay:   true,
		IsPaused:   true,
	}, true)
	if pausedReplayMode.phase != telemetryPhaseReplayAndRecord {
		t.Fatalf("paused replay phase = %s, want %s", pausedReplayMode.phase, telemetryPhaseReplayAndRecord)
	}
	if !pausedReplayMode.isPaused {
		t.Fatal("paused replay isPaused = false, want true")
	}
	if pausedReplayMode.CanAdvanceLap() || pausedReplayMode.CanAppendCurrentLapTick() {
		t.Fatal("paused replay advanced or appended ticks")
	}
	if !pausedReplayMode.ShouldSaveCompletedLap() {
		t.Fatal("paused replay record should preserve save policy")
	}
}

func TestTelemetryModeCompleteDoesNotAppendTicks(t *testing.T) {
	t.Parallel()

	completeMode := newTelemetryMode(&tmodel.TelemetrySnapshot{
		InRace:         true,
		CurrentLap:     2,
		IsRaceComplete: true,
	}, false)
	if completeMode.phase != telemetryPhaseRace {
		t.Fatalf("complete phase = %s, want %s", completeMode.phase, telemetryPhaseRace)
	}
	if !completeMode.isFinished {
		t.Fatal("complete isFinished = false, want true")
	}
	if completeMode.CanStartLap() || completeMode.CanAppendCurrentLapTick() {
		t.Fatal("complete state started or appended ticks")
	}
	if !completeMode.CanAdvanceLap() {
		t.Fatal("complete canAdvanceLap = false, want true")
	}
}

func TestTelemetryModeNotLapping(t *testing.T) {
	t.Parallel()

	for _, snapshot := range []*tmodel.TelemetrySnapshot{
		{InRace: false, CurrentLap: 2},
		{InRace: true, CurrentLap: 0},
	} {
		mode := newTelemetryMode(snapshot, false)
		if mode.phase != telemetryPhaseNotLapping {
			t.Fatalf("phase = %s, want %s", mode.phase, telemetryPhaseNotLapping)
		}
		if mode.CanStartLap() || mode.CanAdvanceLap() || mode.CanAppendCurrentLapTick() || mode.ShouldSaveCompletedLap() {
			t.Fatalf("not lapping mode has recording capability: %+v", mode)
		}
	}
}
