package models

type TelemetrySnapshot struct {
	Speed         float64 `json:"speed"`
	RPM           float64 `json:"rpm"`
	RPMMax        float64 `json:"rpm_max"`
	Gear          int     `json:"gear"`
	SuggestedGear int     `json:"suggested_gear"`
	Throttle      float64 `json:"throttle"`
	Brake         float64 `json:"brake"`
	Fuel          float64 `json:"fuel"`
	FuelCapacity  float64 `json:"fuel_capacity"`
	Boost         float64 `json:"boost"`
	WaterTemp     float64 `json:"water_temp"`
	OilTemp       float64 `json:"oil_temp"`
	OilPressure   float64 `json:"oil_pressure"`
	RideHeight    float64 `json:"ride_height"`
	Clutch        float64 `json:"clutch"`

	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
	PositionZ float64 `json:"position_z"`

	VelocityX float64 `json:"velocity_x"`
	VelocityY float64 `json:"velocity_y"`
	VelocityZ float64 `json:"velocity_z"`

	RotationPitch float64 `json:"rotation_pitch"`
	RotationYaw   float64 `json:"rotation_yaw"`
	RotationRoll  float64 `json:"rotation_roll"`

	TyreTempFL  float64 `json:"tyre_temp_fl"`
	TyreTempFR  float64 `json:"tyre_temp_fr"`
	TyreTempRL  float64 `json:"tyre_temp_rl"`
	TyreTempRR  float64 `json:"tyre_temp_rr"`
	TireSlipFL  float64 `json:"tire_slip_fl"`
	TireSlipFR  float64 `json:"tire_slip_fr"`
	TireSlipRL  float64 `json:"tire_slip_rl"`
	TireSlipRR  float64 `json:"tire_slip_rr"`
	TireSlipAvg float64 `json:"tire_slip_avg"`
	TireSlipMax float64 `json:"tire_slip_max"`

	SuspensionFL float64 `json:"suspension_fl"`
	SuspensionFR float64 `json:"suspension_fr"`
	SuspensionRL float64 `json:"suspension_rl"`
	SuspensionRR float64 `json:"suspension_rr"`

	SequenceID     uint32 `json:"sequence_id"`
	CurrentLap     int16  `json:"current_lap"`
	CurrentLaptime int64  `json:"current_laptime"`
	LastLaptime    int64  `json:"last_laptime"`
	BestLaptime    int64  `json:"best_laptime"`
	TotalLaps      int16  `json:"total_laps"`
	RaceEntrants   int16  `json:"race_entrants"`
	GridPosition   int16  `json:"grid_position"`

	VehicleID           uint32 `json:"vehicle_id"`
	VehicleManufacturer string `json:"vehicle_manufacturer"`
	VehicleModel        string `json:"vehicle_model"`
	VehicleCategory     string `json:"vehicle_category"`

	IsPaused       bool `json:"is_paused"`
	InRace         bool `json:"in_race"`
	IsLoading      bool `json:"is_loading"`
	InGear         bool `json:"in_gear"`
	HasTurbo       bool `json:"has_turbo"`
	IsReplay       bool `json:"is_replay"`
	IsRaceComplete bool `json:"is_race_complete"`
	PS5Connected   bool `json:"ps5_connected"`

	EnergyRecovery float64 `json:"energy_recovery"`

	CircuitLength int `json:"circuit_length,omitempty"`
}
