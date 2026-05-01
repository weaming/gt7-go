package lap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/weaming/gt7-go/internal/models"
)

type Manager struct {
	mu sync.RWMutex

	laps       []*models.Lap
	sessions   map[string]*models.Session
	currentLap *CurrentLap

	previousLapNum    int16
	specialPacketTime float64

	onLapCompleted func(lap *models.Lap)
	savePath       string

	currentLapSavePath string
	currentLapFile     *os.File
	currentLapBuf      *bufio.Writer
}

type CurrentLap struct {
	startTime       time.Time
	dataThrottle    []float64
	dataBraking     []float64
	dataCoasting    []int
	dataSpeed       []float64
	dataTime        []float64
	dataRPM         []float64
	dataGear        []int
	dataTires       []float64
	dataBoost       []float64
	dataRotationYaw []float64
	dataPositionX   []float64
	dataPositionY   []float64
	dataPositionZ   []float64
	dataLapNum      []int16

	fuelAtStart int

	throttleAndBrakeTicks     int
	noThrottleAndNoBrakeTicks int
	fullBrakeTicks            int
	fullThrottleTicks         int
	tiresOverheatedTicks      int
	tiresSpinningTicks        int

	lapTicks    int
	lapLiveTime float64
	totalLaps   int
	number      int
	carID       int

	dataAbsoluteYawRatePerSecond []float64
	rotationYawHistory           []float64
	carName                      string

	circuitID        string
	circuitName      string
	circuitVariation string
}

type currentLapHeader struct {
	StartTime   time.Time `json:"start"`
	FuelAtStart int       `json:"fuel"`
	TotalLaps   int       `json:"laps"`
	Number      int       `json:"num"`
	CarID       int       `json:"car"`
}

type currentLapLine struct {
	Lap        int16     `json:"l"`
	Speed      float64   `json:"s"`
	Throttle   float64   `json:"t"`
	Brake      float64   `json:"b"`
	RPM        float64   `json:"r"`
	Gear       int       `json:"g"`
	TireRatios []float64 `json:"tr"`
	Boost      float64   `json:"bo"`
	Yaw        float64   `json:"y"`
	PosX       float64   `json:"px"`
	PosY       float64   `json:"py"`
	PosZ       float64   `json:"pz"`
	TyreTemps  []float64 `json:"tt"`
}

func NewManager(onLapCompleted func(lap *models.Lap)) *Manager {
	return &Manager{
		laps:           make([]*models.Lap, 0),
		sessions:       make(map[string]*models.Session),
		previousLapNum: -1,
		onLapCompleted: onLapCompleted,
	}
}

func (m *Manager) StartNewLap(carID int, totalLaps int, lapNumber int, fuel float64, carName, circuitID, circuitName, circuitVariation string) {
	m.currentLap = &CurrentLap{
		startTime:   time.Now(),
		fuelAtStart: int(fuel),
		lapTicks:    0,
		totalLaps:   totalLaps,
		number:      lapNumber,
		carID:       carID,
		carName:     carName,

		circuitID:        circuitID,
		circuitName:      circuitName,
		circuitVariation: circuitVariation,
	}

	if m.currentLapSavePath != "" {
		m.closeCurrentLapFile()
		m.openCurrentLapFile()
		m.writeCurrentLapHeader()
	}
}

func (m *Manager) FinishCurrentLap(lastLapTimeMs int64, fuelAtEnd float64, carID int, carName string) *models.Lap {
	cl := m.currentLap
	if cl == nil {
		return nil
	}

	now := time.Now()

	isPitLap := isPitStopLap(cl.dataSpeed)
	isComplete := isLapComplete(cl.dataPositionX, cl.dataPositionY, cl.dataPositionZ)
	lap := &models.Lap{
		Title:                     secondsToLapTime(float64(lastLapTimeMs) / 1000),
		LapTicks:                  cl.lapTicks,
		LapFinishTime:             lastLapTimeMs,
		LapLiveTime:               cl.lapLiveTime,
		TotalLaps:                 cl.totalLaps,
		Number:                    cl.number,
		ThrottleAndBrakeTicks:     cl.throttleAndBrakeTicks,
		NoThrottleAndNoBrakeTicks: cl.noThrottleAndNoBrakeTicks,
		FullBrakeTicks:            cl.fullBrakeTicks,
		FullThrottleTicks:         cl.fullThrottleTicks,
		TiresOverheatedTicks:      cl.tiresOverheatedTicks,
		TiresSpinningTicks:        cl.tiresSpinningTicks,
		DataThrottle:              cl.dataThrottle,
		DataBraking:               cl.dataBraking,
		DataCoasting:              cl.dataCoasting,
		DataSpeed:                 cl.dataSpeed,
		DataTime:                  cl.dataTime,
		DataRPM:                   cl.dataRPM,
		DataGear:                  cl.dataGear,
		DataTires:                 cl.dataTires,
		DataBoost:                 cl.dataBoost,
		DataRotationYaw:           cl.dataRotationYaw,
		DataPositionX:             cl.dataPositionX,
		DataPositionY:             cl.dataPositionY,
		DataPositionZ:             cl.dataPositionZ,
		FuelAtStart:               cl.fuelAtStart,
		FuelAtEnd:                 int(fuelAtEnd),
		FuelConsumed:              cl.fuelAtStart - int(fuelAtEnd),
		CarID:                     carID,
		CarName:                   carName,
		CircuitID:                 cl.circuitID,
		CircuitName:               cl.circuitName,
		CircuitVariation:          cl.circuitVariation,
			IsPitLap:					isPitLap,
			IsComplete:					isComplete,
		LapStartTimestamp:         &cl.startTime,
		LapEndTimestamp:           &now,
	}

	m.mu.Lock()
	m.laps = append([]*models.Lap{lap}, m.laps...)
	m.addLapToSession(lap)
	m.mu.Unlock()

	m.currentLap = nil
	m.closeCurrentLapFile()

	if m.savePath != "" {
		if err := m.SaveLaps(); err != nil {
			log.Printf("save laps: %v", err)
		}
	}

	if m.onLapCompleted != nil {
		m.onLapCompleted(lap)
	}

	return lap
}

func (m *Manager) HandleLapTransition(currentLap int16, lastLapTimeMs int64, fuelAtEnd float64, carID int, carName string) *models.Lap {
	if m.previousLapNum >= 0 && currentLap == m.previousLapNum+1 {
		specialTime := float64(lastLapTimeMs) - float64(m.currentLap.lapTicks)*1000.0/60.0
		if specialTime > 0 {
			m.specialPacketTime += specialTime
		}
		m.previousLapNum = currentLap
		return m.FinishCurrentLap(lastLapTimeMs, fuelAtEnd, carID, carName)
	}
	if m.previousLapNum >= 0 && currentLap != m.previousLapNum+1 && currentLap != m.previousLapNum {
		m.currentLap = nil
	}
	m.previousLapNum = currentLap
	return nil
}

func (m *Manager) LogData(speed, throttle, brake float64, rpm float64, gear int,
	tireRatios []float64, boost float64, yaw float64,
	posX, posY, posZ float64, tyreTemps []float64, lapNum int16) {

	cl := m.currentLap
	if cl == nil {
		return
	}

	isCoasting := brake == 0 && throttle == 0
	coastVal := 0
	if isCoasting {
		coastVal = 1
	}

	cl.dataSpeed = append(cl.dataSpeed, speed)
	cl.dataThrottle = append(cl.dataThrottle, throttle)
	cl.dataBraking = append(cl.dataBraking, brake)
	cl.dataCoasting = append(cl.dataCoasting, coastVal)
	cl.dataRPM = append(cl.dataRPM, rpm)
	cl.dataGear = append(cl.dataGear, gear)
	if boost < 0 {
		boost = 0
	}
	cl.dataBoost = append(cl.dataBoost, boost)
	cl.dataPositionX = append(cl.dataPositionX, posX)
	cl.dataPositionY = append(cl.dataPositionY, posY)
	cl.dataPositionZ = append(cl.dataPositionZ, posZ)
	cl.dataLapNum = append(cl.dataLapNum, lapNum)

	sumTireRatio := 0.0
	for _, r := range tireRatios {
		sumTireRatio += r
	}
	cl.dataTires = append(cl.dataTires, sumTireRatio)

	cl.lapTicks++
	cl.lapLiveTime = float64(cl.lapTicks) / 60.0

	if throttle > 0 && brake > 0 {
		cl.throttleAndBrakeTicks++
	}
	if isCoasting {
		cl.noThrottleAndNoBrakeTicks++
	}
	if brake >= 100 {
		cl.fullBrakeTicks++
	}
	if throttle >= 100 {
		cl.fullThrottleTicks++
	}

	for _, temp := range tyreTemps {
		if temp > 100 {
			cl.tiresOverheatedTicks++
			break
		}
	}

	for _, r := range tireRatios {
		if r > 1.1 {
			cl.tiresSpinningTicks++
			break
		}
	}

	cl.rotationYawHistory = append(cl.rotationYawHistory, yaw)
	cl.dataRotationYaw = append(cl.dataRotationYaw, yaw)

	if len(cl.rotationYawHistory) > 60 {
		yawRate := yaw - cl.rotationYawHistory[len(cl.rotationYawHistory)-61]
		if yawRate < 0 {
			yawRate = -yawRate
		}
		cl.dataAbsoluteYawRatePerSecond = append(cl.dataAbsoluteYawRatePerSecond, yawRate)
	} else {
		cl.dataAbsoluteYawRatePerSecond = append(cl.dataAbsoluteYawRatePerSecond, 0)
	}

	m.appendCurrentLapLine(speed, throttle, brake, rpm, gear, tireRatios, boost, yaw, posX, posY, posZ, tyreTemps, lapNum)
}

func (m *Manager) SetPreviousLapNum(n int16) {
	m.previousLapNum = n
}

func (m *Manager) GetLaps() []*models.Lap {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.Lap, len(m.laps))
	copy(result, m.laps)
	return result
}

func sessionKey(circuitID, carName string) string {
	return circuitID + "|" + carName
}

func (m *Manager) GetSessions() []*models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result
}

func (m *Manager) addLapToSession(lap *models.Lap) {
	if lap.CircuitID == "" {
		return
	}
	key := sessionKey(lap.CircuitID, lap.CarName)
	s, ok := m.sessions[key]
	if !ok {
		s = &models.Session{
			CircuitID:   lap.CircuitID,
			CircuitName: lap.CircuitName,
			Variation:   lap.CircuitVariation,
			CarName:     lap.CarName,
			CarID:       lap.CarID,
			Laps:        make([]*models.Lap, 0),
		}
		m.sessions[key] = s
	}
	s.Laps = append([]*models.Lap{lap}, s.Laps...)
	if lap.LapFinishTime > 0 && (s.BestTime == 0 || lap.LapFinishTime < s.BestTime) {
		s.BestTime = lap.LapFinishTime
	}
}

func (m *Manager) rebuildSessions() {
	m.sessions = make(map[string]*models.Session)
	for _, l := range m.laps {
		m.addLapToSession(l)
	}
}

func (m *Manager) LoadLaps(laps []*models.Lap, replace bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if replace {
		m.laps = laps
	} else {
		seen := make(map[string]bool)
		for _, l := range m.laps {
			key := dedupKey(l)
			seen[key] = true
		}
		for _, l := range laps {
			key := dedupKey(l)
			if !seen[key] {
				m.laps = append(m.laps, l)
				seen[key] = true
			}
		}
	}
}

func dedupKey(l *models.Lap) string {
	return fmt.Sprintf("%d-%d", l.LapFinishTime, l.CarID)
}

func (m *Manager) DeleteLaps(indices []int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idxSet := make(map[int]bool)
	for _, idx := range indices {
		idxSet[idx] = true
	}

	filtered := make([]*models.Lap, 0, len(m.laps))
	for i, lap := range m.laps {
		if !idxSet[i] {
			filtered = append(filtered, lap)
		}
	}
	m.laps = filtered
}

func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.laps = make([]*models.Lap, 0)
	m.sessions = make(map[string]*models.Session)
	m.currentLap = nil
	m.previousLapNum = -1
	m.specialPacketTime = 0
	m.closeCurrentLapFile()
}

func (m *Manager) CurrentLapTicks() int {
	if m.currentLap == nil {
		return 0
	}
	return m.currentLap.lapTicks
}

func (m *Manager) SetSavePath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savePath = path
}

func (m *Manager) SaveLaps() error {
	data, err := json.Marshal(m.laps)
	if err != nil {
		return fmt.Errorf("marshal laps: %w", err)
	}
	return os.WriteFile(m.savePath, data, 0644)
}

func (m *Manager) LoadLapsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read laps file: %w", err)
	}
	var laps []*models.Lap
	if err := json.Unmarshal(data, &laps); err != nil {
		return fmt.Errorf("unmarshal laps: %w", err)
	}
	m.mu.Lock()
	m.laps = laps
	m.rebuildSessions()
	m.mu.Unlock()
	log.Printf("loaded %d laps from %s", len(laps), path)
	return nil
}

// Current lap JSONL persistence

func (m *Manager) SetCurrentLapSavePath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentLapSavePath = path
}

func (m *Manager) openCurrentLapFile() {
	if err := os.MkdirAll(filepath.Dir(m.currentLapSavePath), 0755); err != nil {
		log.Printf("create current lap dir: %v", err)
		return
	}
	f, err := os.OpenFile(m.currentLapSavePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("open current lap file: %v", err)
		return
	}
	m.currentLapFile = f
	m.currentLapBuf = bufio.NewWriter(f)
}

func (m *Manager) closeCurrentLapFile() {
	if m.currentLapBuf != nil {
		m.currentLapBuf.Flush()
		m.currentLapBuf = nil
	}
	if m.currentLapFile != nil {
		m.currentLapFile.Close()
		m.currentLapFile = nil
	}
}

func (m *Manager) writeCurrentLapHeader() {
	if m.currentLapBuf == nil || m.currentLap == nil {
		return
	}
	h := currentLapHeader{
		StartTime:   m.currentLap.startTime,
		FuelAtStart: m.currentLap.fuelAtStart,
		TotalLaps:   m.currentLap.totalLaps,
		Number:      m.currentLap.number,
		CarID:       m.currentLap.carID,
	}
	data, _ := json.Marshal(h)
	m.currentLapBuf.Write(data)
	m.currentLapBuf.WriteByte('\n')
}

func (m *Manager) appendCurrentLapLine(speed, throttle, brake float64, rpm float64, gear int,
	tireRatios []float64, boost float64, yaw float64,
	posX, posY, posZ float64, tyreTemps []float64, lapNum int16) {

	if m.currentLapBuf == nil {
		return
	}
	line := currentLapLine{
		Lap:   lapNum,
		Speed: speed, Throttle: throttle, Brake: brake,
		RPM: rpm, Gear: gear,
		TireRatios: tireRatios, Boost: boost, Yaw: yaw,
		PosX: posX, PosY: posY, PosZ: posZ,
		TyreTemps: tyreTemps,
	}
	data, _ := json.Marshal(line)
	m.currentLapBuf.Write(data)
	m.currentLapBuf.WriteByte('\n')
}

// SaveCurrentLap flushes the current lap file for shutdown/crash safety.
func (m *Manager) SaveCurrentLap() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentLapBuf != nil {
		return m.currentLapBuf.Flush()
	}
	return nil
}

// GetCurrentLapState returns the in-progress lap data for frontend resume.
func (m *Manager) GetCurrentLapState() *models.CurrentLapState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cl := m.currentLap
	if cl == nil {
		return nil
	}

	state := &models.CurrentLapState{
		Type:                         "current_lap",
		DataSpeed:                    cl.dataSpeed,
		DataThrottle:                 cl.dataThrottle,
		DataBraking:                  cl.dataBraking,
		DataCoasting:                 cl.dataCoasting,
		DataRPM:                      cl.dataRPM,
		DataGear:                     cl.dataGear,
		DataBoost:                    cl.dataBoost,
		DataRotationYaw:              cl.dataRotationYaw,
		DataAbsoluteYawRatePerSecond: cl.dataAbsoluteYawRatePerSecond,
		DataPositionX:                cl.dataPositionX,
		DataPositionY:                cl.dataPositionY,
		DataPositionZ:                cl.dataPositionZ,
		FuelAtStart:                  cl.fuelAtStart,
		FuelAtEnd:                    cl.fuelAtStart,
		FuelConsumed:                 0,
		IsLive:                       true,
		LapTicks:                     cl.lapTicks,
		YawHistory:                   cl.rotationYawHistory,
		ThrottleAndBrakeTicks:        cl.throttleAndBrakeTicks,
		NoThrottleAndNoBrakeTicks:    cl.noThrottleAndNoBrakeTicks,
		FullBrakeTicks:               cl.fullBrakeTicks,
		FullThrottleTicks:            cl.fullThrottleTicks,
		Number:                       cl.number,
		TotalLaps:                    cl.totalLaps,
		CarName:                      cl.carName,
	}
	return state
}

func (m *Manager) LoadCurrentLapFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open current lap file: %w", err)
	}
	defer f.Close()

	var h *currentLapHeader
	var lines []currentLapLine

	dec := json.NewDecoder(f)
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			break
		}
		if h == nil {
			var header currentLapHeader
			if err := json.Unmarshal(raw, &header); err == nil && header.Number > 0 {
				h = &header
				continue
			}
		}
		var line currentLapLine
		if err := json.Unmarshal(raw, &line); err == nil {
			lines = append(lines, line)
		}
	}

	if h == nil {
		log.Printf("no valid header in current lap file: %s", path)
		return nil
	}

	cl := &CurrentLap{}
	cl.startTime = h.StartTime
	cl.fuelAtStart = h.FuelAtStart
	cl.totalLaps = h.TotalLaps
	cl.number = h.Number
	cl.carID = h.CarID

	for _, line := range lines {
		cl.dataSpeed = append(cl.dataSpeed, line.Speed)
		cl.dataThrottle = append(cl.dataThrottle, line.Throttle)
		cl.dataBraking = append(cl.dataBraking, line.Brake)
		cl.dataRPM = append(cl.dataRPM, line.RPM)
		cl.dataGear = append(cl.dataGear, line.Gear)
		cl.dataBoost = append(cl.dataBoost, line.Boost)
		cl.dataRotationYaw = append(cl.dataRotationYaw, line.Yaw)
		cl.dataPositionX = append(cl.dataPositionX, line.PosX)
		cl.dataPositionY = append(cl.dataPositionY, line.PosY)
		cl.dataPositionZ = append(cl.dataPositionZ, line.PosZ)
		cl.dataLapNum = append(cl.dataLapNum, line.Lap)

		isCoasting := line.Brake == 0 && line.Throttle == 0
		if isCoasting {
			cl.dataCoasting = append(cl.dataCoasting, 1)
		} else {
			cl.dataCoasting = append(cl.dataCoasting, 0)
		}

		sumTireRatio := 0.0
		for _, r := range line.TireRatios {
			sumTireRatio += r
		}
		cl.dataTires = append(cl.dataTires, sumTireRatio)

		cl.lapTicks++
		cl.lapLiveTime = float64(cl.lapTicks) / 60.0

		if line.Throttle > 0 && line.Brake > 0 {
			cl.throttleAndBrakeTicks++
		}
		if isCoasting {
			cl.noThrottleAndNoBrakeTicks++
		}
		if line.Brake >= 100 {
			cl.fullBrakeTicks++
		}
		if line.Throttle >= 100 {
			cl.fullThrottleTicks++
		}

		for _, temp := range line.TyreTemps {
			if temp > 100 {
				cl.tiresOverheatedTicks++
				break
			}
		}
		for _, r := range line.TireRatios {
			if r > 1.1 {
				cl.tiresSpinningTicks++
				break
			}
		}

		cl.rotationYawHistory = append(cl.rotationYawHistory, line.Yaw)
		if len(cl.rotationYawHistory) > 60 {
			yawRate := line.Yaw - cl.rotationYawHistory[len(cl.rotationYawHistory)-61]
			if yawRate < 0 {
				yawRate = -yawRate
			}
			cl.dataAbsoluteYawRatePerSecond = append(cl.dataAbsoluteYawRatePerSecond, yawRate)
		} else {
			cl.dataAbsoluteYawRatePerSecond = append(cl.dataAbsoluteYawRatePerSecond, 0)
		}
	}

	// Extract completed laps from file data before trimming to current lap
	m.extractCompletedLapsFromCurrentLap(cl)

	if discarded := cl.TrimToLastLap(); discarded > 0 {
		log.Printf("current lap file contained %d data points across multiple laps; truncated to last lap (%d ticks)",
			len(cl.dataPositionX)+discarded, cl.lapTicks)
	}

	m.mu.Lock()
	m.currentLap = cl
	m.mu.Unlock()

	// Open file for continued appending with fresh header
	m.closeCurrentLapFile()
	m.openCurrentLapFile()
	m.writeCurrentLapHeader()

	log.Printf("loaded current lap (lap %d, %d ticks) from %s", cl.number, cl.lapTicks, path)
	return nil
}

// extractCompletedLapsFromCurrentLap detects completed lap segments in loaded
// CurrentLap data and adds them as completed laps. This recovers data from
// a multi-lap current lap file (e.g. after crash).
func (m *Manager) extractCompletedLapsFromCurrentLap(cl *CurrentLap) int {
	if len(cl.dataLapNum) != len(cl.dataSpeed) || len(cl.dataLapNum) == 0 {
		return 0
	}

	var boundaries []int
	for i := 1; i < len(cl.dataLapNum); i++ {
		if cl.dataLapNum[i] != cl.dataLapNum[i-1] {
			boundaries = append(boundaries, i)
		}
	}
	if len(boundaries) == 0 {
		return 0
	}

	start := 0
	count := 0
	for i, end := range boundaries {
		if i == len(boundaries)-1 {
			break // keep last segment as current lap
		}
		lapTicks := end - start
		lapLiveTime := float64(lapTicks) / 60.0
		lap := &models.Lap{
			Title:                        secondsToLapTime(lapLiveTime),
			LapTicks:                     lapTicks,
			LapFinishTime:                int64(lapLiveTime * 1000),
			LapLiveTime:                  lapLiveTime,
			Number:                       int(cl.dataLapNum[start]),
			CarID:                        cl.carID,
			CarName:                      cl.carName,
			DataSpeed:                    copySlice(cl.dataSpeed, start, end),
			DataThrottle:                 copySlice(cl.dataThrottle, start, end),
			DataBraking:                  copySlice(cl.dataBraking, start, end),
			DataCoasting:                 copySliceInt(cl.dataCoasting, start, end),
			DataRPM:                      copySlice(cl.dataRPM, start, end),
			DataGear:                     copySliceInt(cl.dataGear, start, end),
			DataBoost:                    copySlice(cl.dataBoost, start, end),
			DataTires:                    copySlice(cl.dataTires, start, end),
			DataRotationYaw:              copySlice(cl.dataRotationYaw, start, end),
			DataAbsoluteYawRatePerSecond: copySlice(cl.dataAbsoluteYawRatePerSecond, start, end),
			DataPositionX:                copySlice(cl.dataPositionX, start, end),
			DataPositionY:                copySlice(cl.dataPositionY, start, end),
			DataPositionZ:                copySlice(cl.dataPositionZ, start, end),
		}
		m.laps = append([]*models.Lap{lap}, m.laps...)
		m.addLapToSession(lap)
		count++
		start = end
	}

	if count > 0 && m.savePath != "" {
		if err := m.SaveLaps(); err != nil {
			log.Printf("save laps after extracting from current lap file: %v", err)
		}
	}
	return count
}

func copySlice(src []float64, s, e int) []float64 {
	dst := make([]float64, e-s)
	copy(dst, src[s:e])
	return dst
}

func copySliceInt(src []int, s, e int) []int {
	dst := make([]int, e-s)
	copy(dst, src[s:e])
	return dst
}

// TrimToLastLap detects lap transitions from per-tick lap numbers and discards
// all but the last lap segment. Falls back to position-based detection for old
// files without per-tick lap numbers. Returns the number of data points discarded.
func (cl *CurrentLap) TrimToLastLap() int {
	splitIdx := 0
	if len(cl.dataLapNum) == len(cl.dataPositionX) {
		for i := 1; i < len(cl.dataLapNum); i++ {
			if cl.dataLapNum[i] != cl.dataLapNum[i-1] {
				splitIdx = i
			}
		}
	} else {
		// Fallback: detect position resets for old files without lap numbers
		for i := 1; i < len(cl.dataPositionX); i++ {
			dx := cl.dataPositionX[i] - cl.dataPositionX[i-1]
			dz := cl.dataPositionZ[i] - cl.dataPositionZ[i-1]
			if math.Sqrt(dx*dx+dz*dz) > 100 {
				splitIdx = i
			}
		}
	}
	if splitIdx == 0 {
		return 0
	}
	cl.dataSpeed = cl.dataSpeed[splitIdx:]
	cl.dataThrottle = cl.dataThrottle[splitIdx:]
	cl.dataBraking = cl.dataBraking[splitIdx:]
	cl.dataCoasting = cl.dataCoasting[splitIdx:]
	cl.dataRPM = cl.dataRPM[splitIdx:]
	cl.dataGear = cl.dataGear[splitIdx:]
	cl.dataBoost = cl.dataBoost[splitIdx:]
	cl.dataTires = cl.dataTires[splitIdx:]
	cl.dataLapNum = cl.dataLapNum[splitIdx:]
	cl.dataRotationYaw = cl.dataRotationYaw[splitIdx:]
	cl.dataPositionX = cl.dataPositionX[splitIdx:]
	cl.dataPositionY = cl.dataPositionY[splitIdx:]
	cl.dataPositionZ = cl.dataPositionZ[splitIdx:]
	cl.dataAbsoluteYawRatePerSecond = cl.dataAbsoluteYawRatePerSecond[splitIdx:]
	cl.rotationYawHistory = cl.rotationYawHistory[splitIdx:]
	cl.lapTicks = len(cl.dataSpeed)
	cl.lapLiveTime = float64(cl.lapTicks) / 60.0
	return splitIdx
}

func isPitStopLap(speeds []float64) bool {
	const threshold = 3.0      // km/h
	const minConsecutive = 180 // ticks = 3 seconds at 60fps

	consecutive := 0
	for _, s := range speeds {
		if s < threshold {
			consecutive++
			if consecutive >= minConsecutive {
				return true
			}
		} else {
			consecutive = 0
		}
	}
	return false
}

func isLapComplete(posX, posY, posZ []float64) bool {
	// A complete lap ends near where it started (close to the start/finish line).
	if len(posX) < 2 {
		return false
	}
	dx := posX[len(posX)-1] - posX[0]
	dy := posY[len(posY)-1] - posY[0]
	dz := posZ[len(posZ)-1] - posZ[0]
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	return dist < 30 // meters from start point
}
