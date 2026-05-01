package telemetry

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	gttelemetry "github.com/zetetos/gt-telemetry/v2"
	"github.com/zetetos/gt-telemetry/v2/pkg/models"

	"github.com/weaming/gt7-go/internal/hub"
	"github.com/weaming/gt7-go/internal/lap"
	tmodel "github.com/weaming/gt7-go/internal/models"
)

type Engine struct {
	client *gttelemetry.Client
	cancel context.CancelFunc
	mu     sync.RWMutex

	hub        *hub.Hub
	lapManager *lap.Manager

	lastSnapshot  *tmodel.TelemetrySnapshot
	playstationIP string

	hasLoggedWaiting   bool
	hasLoggedConnected bool
	lastPacketSequence uint32
	lastPacketAt       time.Time
	lastStatusSentAt   time.Time

	forceRecord        atomic.Bool
	liveLapDiffTick    atomic.Int32
	currentLapSaveTick atomic.Int32

	circuitID        string
	circuitName      string
	circuitVariation string
}

func New(h *hub.Hub, lapMgr *lap.Manager, psIP string) *Engine {
	return &Engine{
		hub:           h,
		lapManager:    lapMgr,
		playstationIP: psIP,
	}
}

func (e *Engine) Start(ctx context.Context) error {
	opts := gttelemetry.Options{
		LogLevel: "info",
	}

	if e.playstationIP != "" {
		opts.Source = "udp://" + e.playstationIP + ":33739"
		log.Printf("telemetry: connecting to PS5 at %s:33739", e.playstationIP)
	} else {
		log.Printf("telemetry: auto-discovering PS5 on UDP 33739...")
	}

	client, err := gttelemetry.New(opts)
	if err != nil {
		return fmt.Errorf("create telemetry client: %w", err)
	}
	e.client = client

	ctx, e.cancel = context.WithCancel(ctx)

	go e.telemetryRunLoop(ctx)
	go e.snapshotBroadcastLoop(ctx)

	return nil
}

func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *Engine) telemetryRunLoop(ctx context.Context) {
	recoverable, err := e.client.Run(ctx)
	if err != nil {
		log.Printf("telemetry client stopped (recoverable=%v): %v", recoverable, err)
	}
}

func (e *Engine) snapshotBroadcastLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.broadcastSnapshot()
		}
	}
}

func (e *Engine) broadcastSnapshot() {
	t := e.client.Telemetry
	if t == nil {
		if !e.hasLoggedWaiting {
			log.Println("telemetry: waiting for PS5 data...")
			e.hasLoggedWaiting = true
		}
		e.maybeBroadcastConnectionStatus(false)
		return
	}

	if t.SequenceID() == 0 {
		e.maybeBroadcastConnectionStatus(false)
		return
	}

	ps5Connected := e.updateConnectionStatus(t.SequenceID())
	e.maybeBroadcastConnectionStatus(ps5Connected)

	if !e.hasLoggedConnected {
		e.hasLoggedConnected = true
		log.Printf("telemetry: connected — seq=%d vehicle=%s lap=%d/%d",
			t.SequenceID(), t.VehicleModel(), t.CurrentLap(), t.RaceLaps())
	}

	flags := t.Flags()
	tireRatios := e.computeTireRatios(t)
	tireSlip := 0.0
	for _, ratio := range tireRatios {
		tireSlip += ratio
	}

	snapshot := &tmodel.TelemetrySnapshot{
		Speed:        float64(t.GroundSpeedMetresPerSecond()) * 3.6,
		RPM:          float64(t.EngineRPM()),
		Gear:         int(t.CurrentGear()),
		Throttle:     float64(t.ThrottleOutputPercent()),
		Brake:        float64(t.BrakeInputPercent()),
		Fuel:         float64(t.FuelLevel()),
		FuelCapacity: float64(t.FuelCapacity()),
		Boost:        float64(t.TurboBoostBar()),
		WaterTemp:    float64(t.WaterTemperatureCelsius()),
		OilTemp:      float64(t.OilTemperatureCelsius()),
		OilPressure:  float64(t.OilPressureKPA()),
		RideHeight:   float64(t.RideHeightMetres() * 1000),

		PositionX: float64(t.PositionalMapCoordinates().X),
		PositionY: float64(t.PositionalMapCoordinates().Y),
		PositionZ: float64(t.PositionalMapCoordinates().Z),

		VelocityX: float64(t.VelocityVector().X),
		VelocityY: float64(t.VelocityVector().Y),
		VelocityZ: float64(t.VelocityVector().Z),

		RotationPitch: float64(t.RotationEnvelope().Pitch),
		RotationYaw:   float64(t.RotationEnvelope().Yaw),
		RotationRoll:  float64(t.RotationEnvelope().Roll),

		TyreTempFL: float64(t.TyreTemperatureCelsius().FrontLeft),
		TyreTempFR: float64(t.TyreTemperatureCelsius().FrontRight),
		TyreTempRL: float64(t.TyreTemperatureCelsius().RearLeft),
		TyreTempRR: float64(t.TyreTemperatureCelsius().RearRight),
		TireSlip:   tireSlip,

		SequenceID: t.SequenceID(),
		CurrentLap: t.CurrentLap(),

		LastLaptime:  int64(t.LastLaptime().Milliseconds()),
		BestLaptime:  int64(t.BestLaptime().Milliseconds()),
		TotalLaps:    t.RaceLaps(),
		RaceEntrants: t.RaceEntrants(),
		GridPosition: t.GridPosition(),

		VehicleID:           t.VehicleID(),
		VehicleManufacturer: t.VehicleManufacturer(),
		VehicleModel:        t.VehicleModel(),
		VehicleCategory:     t.VehicleCategory(),

		IsPaused:       flags.GamePaused,
		InRace:         t.IsOnCircuit(),
		IsLoading:      flags.Loading,
		InGear:         flags.InGear,
		HasTurbo:       flags.HasTurbo,
		EnergyRecovery: float64(t.EnergyRecovery()),
		IsRaceComplete: t.RaceComplete(),
		PS5Connected:   ps5Connected,

		SuggestedGear: normalizedSuggestedGear(t),

		SuspensionFL: float64(t.SuspensionHeightMetres().FrontLeft),
		SuspensionFR: float64(t.SuspensionHeightMetres().FrontRight),
		SuspensionRL: float64(t.SuspensionHeightMetres().RearLeft),
		SuspensionRR: float64(t.SuspensionHeightMetres().RearRight),

		CurrentLaptime: int64(t.TimeOfDay().Milliseconds()),
	}
	isReplay := e.isReplayTelemetry(t)
	snapshot.IsReplay = isReplay

	e.mu.Lock()
	e.lastSnapshot = snapshot
	e.mu.Unlock()

	if e.circuitID == "" {
		e.detectCircuit(t)
	}

	curLap := snapshot.CurrentLap
	shouldRecordReplay := e.forceRecord.Load()
	shouldSaveCompletedLap := !isReplay || shouldRecordReplay

	if !e.lapManager.IsCurrentLapActive() && curLap > 0 && !snapshot.IsRaceComplete {
		e.startNewLap(t, snapshot)
	}

	if !flags.GamePaused && e.lapManager.IsCurrentLapActive() {
		e.handleLapTransition(t, shouldSaveCompletedLap)
		if !e.lapManager.IsCurrentLapActive() && curLap > 0 && !snapshot.IsRaceComplete {
			e.startNewLap(t, snapshot)
		}
	}

	// Do not persist telemetry samples while the game is paused or after the race is complete.
	if !flags.GamePaused && !snapshot.IsRaceComplete && e.lapManager.IsCurrentLapActive() {
		tireTemps := models.CornerSet{
			FrontLeft:  t.TyreTemperatureCelsius().FrontLeft,
			FrontRight: t.TyreTemperatureCelsius().FrontRight,
			RearLeft:   t.TyreTemperatureCelsius().RearLeft,
			RearRight:  t.TyreTemperatureCelsius().RearRight,
		}

		if err := e.lapManager.LogData(
			snapshot.Speed,
			snapshot.Throttle,
			snapshot.Brake,
			snapshot.RPM,
			snapshot.Gear,
			tireRatios,
			snapshot.Boost,
			float64(t.RotationEnvelope().Yaw),
			snapshot.PositionX,
			snapshot.PositionY,
			snapshot.PositionZ,
			[]float64{
				float64(tireTemps.FrontLeft),
				float64(tireTemps.FrontRight),
				float64(tireTemps.RearLeft),
				float64(tireTemps.RearRight),
			},
			curLap,
		); err != nil {
			log.Printf("log lap data: %v", err)
		}
	}
	saveTick := e.currentLapSaveTick.Add(1)
	if saveTick >= 60 {
		e.currentLapSaveTick.Store(0)
		if err := e.lapManager.SaveCurrentLap(); err != nil {
			log.Printf("save current lap: %v", err)
		}
	}

	if e.hub.NumClients() > 0 {
		e.hub.Broadcast(tmodel.TelemetryMessage{
			Type: "telemetry",
			Data: snapshot,
		})
		tick := e.liveLapDiffTick.Add(1)
		if tick >= 12 {
			e.liveLapDiffTick.Store(0)
			curState := e.lapManager.GetCurrentLapState()
			if curState != nil && curState.CircuitID != "" && curState.CarName != "" {
				ref := e.lapManager.GetBestLapForCircuit(curState.CircuitID, curState.CarName)
				if ref != nil {
					td := e.lapManager.GetCurrentLapTimeDiff(ref)
					if td != nil {
						e.hub.Broadcast(tmodel.LiveLapUpdate{
							Type:     "live_lap_diff",
							TimeDiff: td,
						})
					}
				}
			}
		}
	}
}

func (e *Engine) startNewLap(t *gttelemetry.Transformer, snapshot *tmodel.TelemetrySnapshot) {
	curLap := snapshot.CurrentLap
	startsAtTrackStart := e.isAtStartLine(t)
	log.Printf("telemetry: StartNewLap lap=%d seq=%d gameState=%v force=%v startLine=%v", curLap, t.SequenceID(), t.GameState(), e.forceRecord.Load(), startsAtTrackStart)
	e.lapManager.StartNewLap(
		int(t.VehicleID()), int(t.RaceLaps()), int(curLap),
		float64(t.FuelLevel()), t.VehicleModel(),
		e.circuitID, e.circuitName, e.circuitVariation, startsAtTrackStart,
	)

	if e.hub.NumClients() > 0 {
		if cur := e.lapManager.GetCurrentLapState(); cur != nil && cur.LapTicks > 0 {
			e.hub.Broadcast(cur)
		}
	}
}

func normalizedSuggestedGear(t *gttelemetry.Transformer) int {
	suggestedGear := int(t.SuggestedGear())
	if suggestedGear <= 0 || suggestedGear == 15 {
		return 0
	}

	totalGears := t.Transmission().Gears
	if totalGears > 0 && suggestedGear > totalGears {
		return 0
	}
	if totalGears == 0 && suggestedGear > 10 {
		return 0
	}

	return suggestedGear
}

func (e *Engine) computeTireRatios(t *gttelemetry.Transformer) []float64 {
	tireSpeeds := t.WheelSpeedMetresPerSecond()
	speed := float64(t.GroundSpeedMetresPerSecond()) * 3.6
	tireRatios := make([]float64, 4)
	if speed <= 0 {
		return tireRatios
	}

	tireRatios[0] = float64(tireSpeeds.FrontLeft*3.6) / speed
	tireRatios[1] = float64(tireSpeeds.FrontRight*3.6) / speed
	tireRatios[2] = float64(tireSpeeds.RearLeft*3.6) / speed
	tireRatios[3] = float64(tireSpeeds.RearRight*3.6) / speed
	return tireRatios
}

func (e *Engine) isReplayTelemetry(t *gttelemetry.Transformer) bool {
	if t.GameState() == models.GameStateReplay {
		return true
	}
	if e.client == nil {
		return false
	}
	isReplaySource, err := e.client.IsReplaySource()
	if err != nil {
		return false
	}
	return isReplaySource
}

func (e *Engine) detectCircuit(t *gttelemetry.Transformer) {
	if e.client == nil || e.client.CircuitDB == nil {
		return
	}
	coord := t.PositionalMapCoordinates()
	cid, found := e.client.CircuitDB.GetCircuitAtCoordinate(
		models.Coordinate{X: coord.X, Y: coord.Y, Z: coord.Z},
		models.CoordinateTypeCircuit,
	)
	if !found {
		return
	}
	e.circuitID = cid
	if info, ok := e.client.CircuitDB.GetCircuitByID(cid); ok {
		e.circuitName = info.Name
		e.circuitVariation = info.Variation
	}
	log.Printf("circuit detected: %s (%s - %s)", e.circuitID, e.circuitName, e.circuitVariation)
}

func (e *Engine) isAtStartLine(t *gttelemetry.Transformer) bool {
	if e.client == nil || e.client.CircuitDB == nil {
		return false
	}

	cid, found := e.client.CircuitDB.GetCircuitAtCoordinate(
		t.PositionalMapCoordinates(),
		models.CoordinateTypeStartLine,
	)
	if !found {
		return false
	}

	if e.circuitID != "" {
		return cid == e.circuitID
	}

	e.circuitID = cid
	if info, ok := e.client.CircuitDB.GetCircuitByID(cid); ok {
		e.circuitName = info.Name
		e.circuitVariation = info.Variation
	}
	return true
}

func (e *Engine) handleLapTransition(t *gttelemetry.Transformer, shouldSaveCompleted bool) {
	curLap := t.CurrentLap()

	carName := e.client.Telemetry.VehicleModel()
	if carName == "" {
		carName = lap.CarNameForID(int(t.VehicleID()), "")
	}

	completedLap := e.lapManager.HandleLapTransition(
		curLap,
		int64(t.LastLaptime().Milliseconds()),
		float64(t.FuelLevel()),
		int(t.VehicleID()),
		carName,
		shouldSaveCompleted,
	)

	if completedLap != nil {
		log.Printf("telemetry: lap_completed num=%d time=%d ticks=%d", completedLap.Number, completedLap.LapFinishTime, completedLap.LapTicks)
		e.hub.Broadcast(tmodel.LapCompletedMessage{
			Type: "lap_completed",
			Data: completedLap,
		})
		e.broadcastLapsUpdated()
	}
}

func (e *Engine) broadcastLapsUpdated() {
	laps := e.lapManager.GetLaps()

	var bestTime int64 = 1<<63 - 1
	for _, l := range laps {
		if lap.IsRankableLap(l) && l.LapFinishTime < bestTime {
			bestTime = l.LapFinishTime
		}
	}
	if bestTime == 1<<63-1 {
		bestTime = 0
	}

	lap.ComputeAllTimeDiffs(laps)

	e.hub.Broadcast(tmodel.LapsUpdatedMessage{
		Type:     "laps_updated",
		Laps:     laps,
		BestTime: bestTime,
	})
}

func (e *Engine) SetForceRecord(enabled bool) {
	e.forceRecord.Store(enabled)
	log.Printf("replay recording: %v", enabled)
}

func (e *Engine) IsForceRecording() bool {
	return e.forceRecord.Load()
}

func (e *Engine) GetClient() *gttelemetry.Client {
	return e.client
}

func (e *Engine) GetLastSnapshot() *tmodel.TelemetrySnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastSnapshot
}

func (e *Engine) GetConnectionStatus() tmodel.TelemetryStatusMessage {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return tmodel.TelemetryStatusMessage{
		Type:          "telemetry_status",
		PS5Connected:  e.isConnectionFreshLocked(time.Now()),
		PlaystationIP: e.playstationIP,
	}
}

func (e *Engine) updateConnectionStatus(sequenceID uint32) bool {
	now := time.Now()
	e.mu.Lock()
	if sequenceID != e.lastPacketSequence {
		e.lastPacketSequence = sequenceID
		e.lastPacketAt = now
	}
	connected := e.isConnectionFreshLocked(now)
	e.mu.Unlock()
	return connected
}

func (e *Engine) maybeBroadcastConnectionStatus(connected bool) {
	if e.hub.NumClients() == 0 {
		return
	}

	now := time.Now()
	e.mu.Lock()
	if now.Sub(e.lastStatusSentAt) < time.Second {
		e.mu.Unlock()
		return
	}
	e.lastStatusSentAt = now
	status := tmodel.TelemetryStatusMessage{
		Type:          "telemetry_status",
		PS5Connected:  connected,
		PlaystationIP: e.playstationIP,
	}
	e.mu.Unlock()

	e.hub.Broadcast(status)
}

func (e *Engine) isConnectionFreshLocked(now time.Time) bool {
	if e.lastPacketAt.IsZero() {
		return false
	}
	return now.Sub(e.lastPacketAt) <= 2*time.Second
}
