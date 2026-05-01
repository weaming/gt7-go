package models

type FuelMap struct {
	MixtureSetting     int     `json:"mixture_setting"`
	PowerPercent       float64 `json:"power_percent"`
	ConsumptionPercent float64 `json:"consumption_percent"`
	FuelConsumedPerLap float64 `json:"fuel_consumed_per_lap"`
	LapsRemaining      float64 `json:"laps_remaining"`
	TimeRemaining      float64 `json:"time_remaining"`
	LapTimeDiff        float64 `json:"lap_time_diff"`
	LapTimeExpected    float64 `json:"lap_time_expected"`
}

type TimeDiffResult struct {
	Distance  []float64 `json:"distance"`
	Timedelta []float64 `json:"timedelta"`
}

type VarianceResult struct {
	Distance      []float64 `json:"distance"`
	SpeedVariance []float64 `json:"speed_variance"`
}

type BrakePoint struct {
	PositionX float64 `json:"position_x"`
	PositionZ float64 `json:"position_z"`
}

type SpeedPeakValley struct {
	Index int     `json:"index"`
	Speed float64 `json:"speed"`
	Type  string  `json:"type"`
}

type SegmentAnalysis struct {
	NumSegments       int       `json:"num_segments"`
	SegmentDistances  []float64 `json:"segment_distances"`
	ConsistencyStddev []float64 `json:"consistency_stddev"`
	TheoreticalTimes  []float64 `json:"theoretical_times"`
	BestLapTimes      []float64 `json:"best_lap_times"`
	TheoreticalTotal  float64   `json:"theoretical_total"`
	BestLapTotal      float64   `json:"best_lap_total"`
	PotentialGain     float64   `json:"potential_gain"`
}

type LapAnalysis struct {
	FuelMaps     []FuelMap         `json:"fuel_maps,omitempty"`
	TimeDiff     *TimeDiffResult   `json:"time_diff,omitempty"`
	Variance     *VarianceResult   `json:"variance,omitempty"`
	BrakePoints  []BrakePoint      `json:"brake_points,omitempty"`
	PeaksValleys []SpeedPeakValley `json:"peaks_valleys,omitempty"`
	Corner       *SegmentAnalysis  `json:"corner,omitempty"`
}

type LapTableRow struct {
	Number        int     `json:"number"`
	Time          float64 `json:"time"`
	TimeDiff      string  `json:"time_diff"`
	Timestamp     string  `json:"timestamp"`
	Info          string  `json:"info"`
	FuelConsumed  int     `json:"fuel_consumed"`
	FullThrottle  float64 `json:"full_throttle"`
	FullBrake     float64 `json:"full_brake"`
	Coasting      float64 `json:"coasting"`
	TireSpinning  float64 `json:"tire_spinning"`
	ThrottleBrake float64 `json:"throttle_brake"`
	CarName       string  `json:"car_name"`
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type TelemetryMessage struct {
	Type string             `json:"type"`
	Data *TelemetrySnapshot `json:"data"`
}

type TelemetryStatusMessage struct {
	Type          string `json:"type"`
	PS5Connected  bool   `json:"ps5_connected"`
	PlaystationIP string `json:"playstation_ip,omitempty"`
}

type LapCompletedMessage struct {
	Type     string       `json:"type"`
	Data     *Lap         `json:"lap"`
	Analysis *LapAnalysis `json:"analysis"`
}

type LapsUpdatedMessage struct {
	Type      string        `json:"type"`
	Laps      []*Lap        `json:"laps"`
	BestTime  int64         `json:"best_time"`
	TableRows []LapTableRow `json:"table_rows"`
}

type WSCommand struct {
	Type string      `json:"type"`
	Cmd  string      `json:"cmd"`
	ID   interface{} `json:"id,omitempty"`
}

// CurrentLapState mirrors the frontend's liveLap structure for resuming on WS reconnect.
type CurrentLapState struct {
	Type                         string          `json:"type"`
	DataSpeed                    []float64       `json:"data_speed"`
	DataThrottle                 []float64       `json:"data_throttle"`
	DataBraking                  []float64       `json:"data_braking"`
	DataCoasting                 []int           `json:"data_coasting"`
	DataRPM                      []float64       `json:"data_rpm"`
	DataGear                     []int           `json:"data_gear"`
	DataTires                    []float64       `json:"data_tires"`
	DataBoost                    []float64       `json:"data_boost"`
	DataRotationYaw              []float64       `json:"data_rotation_yaw"`
	DataAbsoluteYawRatePerSecond []float64       `json:"data_absolute_yaw_rate_per_second"`
	DataPositionX                []float64       `json:"data_position_x"`
	DataPositionY                []float64       `json:"data_position_y"`
	DataPositionZ                []float64       `json:"data_position_z"`
	FuelAtStart                  int             `json:"fuel_at_start"`
	FuelAtEnd                    int             `json:"fuel_at_end"`
	FuelConsumed                 int             `json:"fuel_consumed"`
	LapFinishTime                int64           `json:"lap_finish_time"`
	IsLive                       bool            `json:"_is_live"`
	LapTicks                     int             `json:"_lap_ticks"`
	YawHistory                   []float64       `json:"_yaw_history"`
	ThrottleAndBrakeTicks        int             `json:"throttle_and_brake_ticks"`
	NoThrottleAndNoBrakeTicks    int             `json:"no_throttle_and_no_brake_ticks"`
	FullBrakeTicks               int             `json:"full_brake_ticks"`
	FullThrottleTicks            int             `json:"full_throttle_ticks"`
	Number                       int             `json:"number"`
	TotalLaps                    int             `json:"total_laps"`
	TotalDistance                float64         `json:"total_distance"`
	CarName                      string          `json:"car_name"`
	CircuitID                    string          `json:"circuit_id,omitempty"`
	CircuitName                  string          `json:"circuit_name,omitempty"`
	CircuitVariation             string          `json:"circuit_variation,omitempty"`
	StartsAtTrackStart           bool            `json:"starts_at_track_start"`
	TimeDiff                     *TimeDiffResult `json:"time_diff,omitempty"`
}

// LiveLapUpdate is periodically broadcast during live telemetry with the
// backend-computed time diff for the in-progress lap.
type LiveLapUpdate struct {
	Type     string          `json:"type"`
	TimeDiff *TimeDiffResult `json:"time_diff"`
}
