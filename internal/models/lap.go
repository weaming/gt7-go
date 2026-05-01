package models

import "time"

type Lap struct {
	Title         string  `json:"title"`
	LapTicks      int     `json:"lap_ticks"`
	LapFinishTime int64   `json:"lap_finish_time"`
	LapLiveTime   float64 `json:"lap_live_time"`
	TotalLaps     int     `json:"total_laps"`
	Number        int     `json:"number"`

	ThrottleAndBrakeTicks     int `json:"throttle_and_brake_ticks"`
	NoThrottleAndNoBrakeTicks int `json:"no_throttle_and_no_brake_ticks"`
	FullBrakeTicks            int `json:"full_brake_ticks"`
	FullThrottleTicks         int `json:"full_throttle_ticks"`
	TiresOverheatedTicks      int `json:"tires_overheated_ticks"`
	TiresSpinningTicks        int `json:"tires_spinning_ticks"`

	DataThrottle                 []float64 `json:"data_throttle"`
	DataBraking                  []float64 `json:"data_braking"`
	DataCoasting                 []int     `json:"data_coasting"`
	DataSpeed                    []float64 `json:"data_speed"`
	DataTime                     []float64 `json:"data_time"`
	DataRPM                      []float64 `json:"data_rpm"`
	DataGear                     []int     `json:"data_gear"`
	DataTires                    []float64 `json:"data_tires"`
	DataBoost                    []float64 `json:"data_boost"`
	DataRotationYaw              []float64 `json:"data_rotation_yaw"`
	DataAbsoluteYawRatePerSecond []float64 `json:"data_absolute_yaw_rate_per_second"`
	DataPositionX                []float64 `json:"data_position_x"`
	DataPositionY                []float64 `json:"data_position_y"`
	DataPositionZ                []float64 `json:"data_position_z"`

	FuelAtStart  int    `json:"fuel_at_start"`
	FuelAtEnd    int    `json:"fuel_at_end"`
	FuelConsumed int    `json:"fuel_consumed"`
	CarID        int    `json:"car_id"`
	CarName      string `json:"car_name"`

	TotalDistance float64 `json:"total_distance"`

	IsReplay   bool `json:"is_replay"`
	IsManual   bool `json:"is_manual"`
	IsPitLap   bool `json:"is_pit_lap"`
	IsComplete bool `json:"is_complete"`

	TimeDiff *TimeDiffResult `json:"time_diff,omitempty"`

	CircuitID        string `json:"circuit_id,omitempty"`
	CircuitName      string `json:"circuit_name,omitempty"`
	CircuitVariation string `json:"circuit_variation,omitempty"`

	LapStartTimestamp *time.Time `json:"lap_start_timestamp"`
	LapEndTimestamp   *time.Time `json:"lap_end_timestamp"`
}

// Session represents a group of laps for a specific track+car combination.
type Session struct {
	CircuitID   string `json:"circuit_id"`
	CircuitName string `json:"circuit_name"`
	Variation   string `json:"variation"`
	CarName     string `json:"car_name"`
	CarID       int    `json:"car_id"`
	Laps        []*Lap `json:"laps"`
	BestTime    int64  `json:"best_time"`
}
