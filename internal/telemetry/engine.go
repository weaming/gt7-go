package telemetry

import (
	"context"
	"fmt"
	"log"
	"sync"
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

	hasConnected bool

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
	ticker := time.NewTicker(100 * time.Millisecond)
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
		if !e.hasConnected {
			log.Println("telemetry: waiting for PS5 data...")
			e.hasConnected = true
		}
		return
	}

	if t.SequenceID() == 0 {
		return
	}

	if !e.hasConnected {
		e.hasConnected = true
		log.Printf("telemetry: connected — seq=%d vehicle=%s lap=%d/%d",
			t.SequenceID(), t.VehicleModel(), t.CurrentLap(), t.RaceLaps())
	}

	flags := t.Flags()
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

		SuggestedGear: int(t.SuggestedGear()),

		SuspensionFL: float64(t.SuspensionHeightMetres().FrontLeft),
		SuspensionFR: float64(t.SuspensionHeightMetres().FrontRight),
		SuspensionRL: float64(t.SuspensionHeightMetres().RearLeft),
		SuspensionRR: float64(t.SuspensionHeightMetres().RearRight),

		CurrentLaptime: int64(t.TimeOfDay().Milliseconds()),
	}

	e.mu.Lock()
	e.lastSnapshot = snapshot
	e.mu.Unlock()

	if e.circuitID == "" {
		e.detectCircuit(t)
	}

	curLap := snapshot.CurrentLap
	lapTicks := e.lapManager.CurrentLapTicks()

	if lapTicks == 0 && curLap > 0 {
		e.lapManager.StartNewLap(
			int(t.VehicleID()), int(t.RaceLaps()), int(curLap),
			float64(t.FuelLevel()), t.VehicleModel(),
			e.circuitID, e.circuitName, e.circuitVariation,
		)
	}

	isLive := t.GameState() == models.GameStateLive
	hasActiveLap := lapTicks > 0 || e.lapManager.CurrentLapTicks() > 0

	// Log data for live laps; also keep logging during pit stops even
	// if GameState briefly leaves Live (e.g. pit menu overlay).
	if (isLive || hasActiveLap) && !flags.GamePaused {
		tireSpeeds := t.WheelSpeedMetresPerSecond()
		tireTemps := models.CornerSet{
			FrontLeft:  t.TyreTemperatureCelsius().FrontLeft,
			FrontRight: t.TyreTemperatureCelsius().FrontRight,
			RearLeft:   t.TyreTemperatureCelsius().RearLeft,
			RearRight:  t.TyreTemperatureCelsius().RearRight,
		}

		speed := snapshot.Speed
		tireRatios := make([]float64, 4)
		if speed > 0 {
			tireRatios[0] = float64(tireSpeeds.FrontLeft*3.6) / speed
			tireRatios[1] = float64(tireSpeeds.FrontRight*3.6) / speed
			tireRatios[2] = float64(tireSpeeds.RearLeft*3.6) / speed
			tireRatios[3] = float64(tireSpeeds.RearRight*3.6) / speed
		}

		e.lapManager.LogData(
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
		)

		if curLap != snapshot.TotalLaps {
			e.handleLapTransition(t)
		}
	}

	if e.hub.NumClients() > 0 {
		e.hub.Broadcast(tmodel.TelemetryMessage{
			Type: "telemetry",
			Data: snapshot,
		})
	}
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

func (e *Engine) handleLapTransition(t *gttelemetry.Transformer) {
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
	)

	if completedLap != nil {
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
		if l.LapFinishTime > 0 && l.LapFinishTime < bestTime {
			bestTime = l.LapFinishTime
		}
	}
	if bestTime == 1<<63-1 {
		bestTime = 0
	}

	e.hub.Broadcast(tmodel.LapsUpdatedMessage{
		Type:     "laps_updated",
		Laps:     laps,
		BestTime: bestTime,
	})
}

func (e *Engine) GetClient() *gttelemetry.Client {
	return e.client
}

func (e *Engine) GetLastSnapshot() *tmodel.TelemetrySnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastSnapshot
}
