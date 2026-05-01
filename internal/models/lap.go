package models

import (
	"encoding/json"
	"fmt"
	"time"
)

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

	IsReplay           bool `json:"is_replay"`
	IsManual           bool `json:"is_manual"`
	IsPitLap           bool `json:"is_pit_lap"`
	IsComplete         bool `json:"is_complete"`
	StartsAtTrackStart bool `json:"starts_at_track_start"`

	TimeDiff *TimeDiffResult `json:"time_diff,omitempty"`

	CircuitID        string `json:"circuit_id,omitempty"`
	CircuitName      string `json:"circuit_name,omitempty"`
	CircuitVariation string `json:"circuit_variation,omitempty"`

	LapStartTimestamp *time.Time `json:"lap_start_timestamp"`
	LapEndTimestamp   *time.Time `json:"lap_end_timestamp"`
}

func (lap *Lap) UnmarshalJSON(data []byte) error {
	type lapAlias Lap

	var decoded lapAlias
	wireLap := struct {
		*lapAlias
		LapStartTimestamp json.RawMessage `json:"lap_start_timestamp"`
		LapEndTimestamp   json.RawMessage `json:"lap_end_timestamp"`
	}{
		lapAlias: &decoded,
	}
	if err := json.Unmarshal(data, &wireLap); err != nil {
		return err
	}

	*lap = Lap(decoded)

	startTimestamp, err := parseLapTimestamp(wireLap.LapStartTimestamp)
	if err != nil {
		return fmt.Errorf("parse lap_start_timestamp: %w", err)
	}
	endTimestamp, err := parseLapTimestamp(wireLap.LapEndTimestamp)
	if err != nil {
		return fmt.Errorf("parse lap_end_timestamp: %w", err)
	}
	lap.LapStartTimestamp = startTimestamp
	lap.LapEndTimestamp = endTimestamp
	return nil
}

func parseLapTimestamp(data json.RawMessage) (*time.Time, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return &parsed, nil
		}
	}

	return nil, fmt.Errorf("unsupported timestamp %q", value)
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
