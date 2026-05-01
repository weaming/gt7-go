package lap

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	startsAtTrackStart           bool

	circuitID        string
	circuitName      string
	circuitVariation string
}

type currentLapHeader struct {
	StartTime          time.Time `json:"start"`
	FuelAtStart        int       `json:"fuel"`
	TotalLaps          int       `json:"laps"`
	Number             int       `json:"num"`
	CarID              int       `json:"car"`
	CarName            string    `json:"car_name,omitempty"`
	CircuitID          string    `json:"circuit_id,omitempty"`
	CircuitName        string    `json:"circuit_name,omitempty"`
	CircuitVariation   string    `json:"circuit_variation,omitempty"`
	StartsAtTrackStart bool      `json:"starts_at_track_start,omitempty"`
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

func (m *Manager) StartNewLap(carID int, totalLaps int, lapNumber int, fuel float64, carName, circuitID, circuitName, circuitVariation string, startsAtTrackStart bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentLap = &CurrentLap{
		startTime:          time.Now(),
		fuelAtStart:        int(fuel),
		lapTicks:           0,
		totalLaps:          totalLaps,
		number:             lapNumber,
		carID:              carID,
		carName:            carName,
		startsAtTrackStart: startsAtTrackStart,

		circuitID:        circuitID,
		circuitName:      circuitName,
		circuitVariation: circuitVariation,
	}

	if m.currentLapSavePath != "" {
		if err := m.closeCurrentLapFile(); err != nil {
			log.Printf("close current lap file: %v", err)
		}
		if err := m.openCurrentLapFile(); err != nil {
			log.Printf("open current lap file: %v", err)
			return
		}
		if err := m.writeCurrentLapHeader(); err != nil {
			log.Printf("write current lap header: %v", err)
		}
	}
}

func (m *Manager) FinishCurrentLap(lastLapTimeMs int64, fuelAtEnd float64, carID int, carName string) *models.Lap {
	m.mu.Lock()
	lap := m.finishCurrentLapLocked(lastLapTimeMs, fuelAtEnd, carID, carName)
	savePath := m.savePath
	onLapCompleted := m.onLapCompleted
	m.mu.Unlock()

	if lap == nil {
		return nil
	}

	if savePath != "" {
		if err := writeLapsFile(savePath, m.GetLaps()); err != nil {
			log.Printf("save laps: %v", err)
		}
	}

	if onLapCompleted != nil {
		onLapCompleted(lap)
	}

	return lap
}

func (m *Manager) finishCurrentLapLocked(lastLapTimeMs int64, fuelAtEnd float64, carID int, carName string) *models.Lap {
	cl := m.currentLap
	if cl == nil {
		return nil
	}

	now := time.Now()

	isPitLap := isPitStopLap(cl.dataSpeed, cl.dataPositionX, cl.dataPositionY, cl.dataPositionZ, cl.dataLapNum)
	isComplete := cl.startsAtTrackStart && isLapComplete(cl.dataPositionX, cl.dataPositionY, cl.dataPositionZ)
	lap := &models.Lap{
		Title:                        secondsToLapTime(float64(lastLapTimeMs) / 1000),
		LapTicks:                     cl.lapTicks,
		LapFinishTime:                lastLapTimeMs,
		LapLiveTime:                  cl.lapLiveTime,
		TotalLaps:                    cl.totalLaps,
		Number:                       cl.number,
		ThrottleAndBrakeTicks:        cl.throttleAndBrakeTicks,
		NoThrottleAndNoBrakeTicks:    cl.noThrottleAndNoBrakeTicks,
		FullBrakeTicks:               cl.fullBrakeTicks,
		FullThrottleTicks:            cl.fullThrottleTicks,
		TiresOverheatedTicks:         cl.tiresOverheatedTicks,
		TiresSpinningTicks:           cl.tiresSpinningTicks,
		DataThrottle:                 cl.dataThrottle,
		DataBraking:                  cl.dataBraking,
		DataCoasting:                 cl.dataCoasting,
		DataSpeed:                    cl.dataSpeed,
		DataTime:                     cl.dataTime,
		DataRPM:                      cl.dataRPM,
		DataGear:                     cl.dataGear,
		DataTires:                    cl.dataTires,
		DataBoost:                    cl.dataBoost,
		DataRotationYaw:              cl.dataRotationYaw,
		DataAbsoluteYawRatePerSecond: cl.dataAbsoluteYawRatePerSecond,
		DataPositionX:                cl.dataPositionX,
		DataPositionY:                cl.dataPositionY,
		DataPositionZ:                cl.dataPositionZ,
		TotalDistance:                computeTotalDistance(cl.dataPositionX, cl.dataPositionZ),
		FuelAtStart:                  cl.fuelAtStart,
		FuelAtEnd:                    int(fuelAtEnd),
		FuelConsumed:                 cl.fuelAtStart - int(fuelAtEnd),
		CarID:                        carID,
		CarName:                      carName,
		CircuitID:                    cl.circuitID,
		CircuitName:                  cl.circuitName,
		CircuitVariation:             cl.circuitVariation,
		IsPitLap:                     isPitLap,
		IsComplete:                   isComplete,
		StartsAtTrackStart:           cl.startsAtTrackStart,
		LapStartTimestamp:            &cl.startTime,
		LapEndTimestamp:              &now,
	}

	m.laps = append([]*models.Lap{lap}, m.laps...)
	m.addLapToSession(lap)
	m.currentLap = nil
	if err := m.closeCurrentLapFile(); err != nil {
		log.Printf("close current lap file: %v", err)
	}

	return lap
}

func (m *Manager) HandleLapTransition(currentLap int16, lastLapTimeMs int64, fuelAtEnd float64, carID int, carName string, shouldSaveCompleted bool) *models.Lap {
	m.mu.Lock()

	if m.previousLapNum >= 0 && currentLap == m.previousLapNum+1 {
		ticks := 0
		if m.currentLap != nil {
			ticks = m.currentLap.lapTicks
		}
		log.Printf("lap: fwd transition prev=%d cur=%d ticks=%d", m.previousLapNum, currentLap, ticks)
		if m.currentLap == nil {
			m.previousLapNum = currentLap
			m.mu.Unlock()
			return nil
		}

		if !shouldSaveCompleted {
			m.previousLapNum = currentLap
			m.currentLap = nil
			if err := m.closeCurrentLapFile(); err != nil {
				log.Printf("close current lap file: %v", err)
			}
			m.mu.Unlock()
			return nil
		}

		specialTime := float64(lastLapTimeMs) - float64(m.currentLap.lapTicks)*1000.0/60.0
		if specialTime > 0 {
			m.specialPacketTime += specialTime
		}
		m.previousLapNum = currentLap
		lap := m.finishCurrentLapLocked(lastLapTimeMs, fuelAtEnd, carID, carName)
		savePath := m.savePath
		onLapCompleted := m.onLapCompleted
		m.mu.Unlock()

		if lap != nil && savePath != "" {
			if err := writeLapsFile(savePath, m.GetLaps()); err != nil {
				log.Printf("save laps: %v", err)
			}
		}
		if lap != nil && onLapCompleted != nil {
			onLapCompleted(lap)
		}
		return lap
	}
	if m.previousLapNum >= 0 && currentLap != m.previousLapNum+1 && currentLap != m.previousLapNum {
		ticks := 0
		if m.currentLap != nil {
			ticks = m.currentLap.lapTicks
		}
		log.Printf("lap: gap detected prev=%d cur=%d curTicks=%d — discarding current lap", m.previousLapNum, currentLap, ticks)
		m.currentLap = nil
		if err := m.closeCurrentLapFile(); err != nil {
			log.Printf("close current lap file: %v", err)
		}
	}
	m.previousLapNum = currentLap
	m.mu.Unlock()
	return nil
}

func (m *Manager) LogData(speed, throttle, brake float64, rpm float64, gear int,
	tireRatios []float64, boost float64, yaw float64,
	posX, posY, posZ float64, tyreTemps []float64, lapNum int16) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cl := m.currentLap
	if cl == nil {
		return nil
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
		sumTireRatio += math.Abs(r)
	}
	cl.dataTires = append(cl.dataTires, sumTireRatio/4)

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

	if err := m.appendCurrentLapLine(speed, throttle, brake, rpm, gear, tireRatios, boost, yaw, posX, posY, posZ, tyreTemps, lapNum); err != nil {
		return fmt.Errorf("append current lap line: %w", err)
	}

	return nil
}

func (m *Manager) SetPreviousLapNum(n int16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.previousLapNum = n
}

func (m *Manager) GetLaps() []*models.Lap {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.Lap, len(m.laps))
	for i, lap := range m.laps {
		result[i] = cloneLap(lap)
	}
	return result
}

func cloneLap(lap *models.Lap) *models.Lap {
	if lap == nil {
		return nil
	}

	cloned := *lap
	cloned.DataThrottle = copyFloat64Slice(lap.DataThrottle)
	cloned.DataBraking = copyFloat64Slice(lap.DataBraking)
	cloned.DataCoasting = copyIntSlice(lap.DataCoasting)
	cloned.DataSpeed = copyFloat64Slice(lap.DataSpeed)
	cloned.DataTime = copyFloat64Slice(lap.DataTime)
	cloned.DataRPM = copyFloat64Slice(lap.DataRPM)
	cloned.DataGear = copyIntSlice(lap.DataGear)
	cloned.DataTires = copyFloat64Slice(lap.DataTires)
	cloned.DataBoost = copyFloat64Slice(lap.DataBoost)
	cloned.DataRotationYaw = copyFloat64Slice(lap.DataRotationYaw)
	cloned.DataAbsoluteYawRatePerSecond = copyFloat64Slice(lap.DataAbsoluteYawRatePerSecond)
	cloned.DataPositionX = copyFloat64Slice(lap.DataPositionX)
	cloned.DataPositionY = copyFloat64Slice(lap.DataPositionY)
	cloned.DataPositionZ = copyFloat64Slice(lap.DataPositionZ)

	if lap.TimeDiff != nil {
		clonedDiff := *lap.TimeDiff
		clonedDiff.Distance = copyFloat64Slice(lap.TimeDiff.Distance)
		clonedDiff.Timedelta = copyFloat64Slice(lap.TimeDiff.Timedelta)
		cloned.TimeDiff = &clonedDiff
	}

	if lap.LapStartTimestamp != nil {
		start := *lap.LapStartTimestamp
		cloned.LapStartTimestamp = &start
	}
	if lap.LapEndTimestamp != nil {
		end := *lap.LapEndTimestamp
		cloned.LapEndTimestamp = &end
	}

	return &cloned
}

func copyFloat64Slice(values []float64) []float64 {
	if values == nil {
		return nil
	}
	copied := make([]float64, len(values))
	copy(copied, values)
	return copied
}

func copyIntSlice(values []int) []int {
	if values == nil {
		return nil
	}
	copied := make([]int, len(values))
	copy(copied, values)
	return copied
}

func sessionKey(circuitID, carName string) string {
	return circuitID + "|" + carName
}

func (m *Manager) GetSessions() []*models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		session := *s
		session.Laps = make([]*models.Lap, len(s.Laps))
		for i, lap := range s.Laps {
			session.Laps[i] = cloneLap(lap)
		}
		result = append(result, &session)
	}
	return result
}

// GetBestLapForCircuit finds the fastest completed lap for the given circuit
// and car across all sessions.
func (m *Manager) GetBestLapForCircuit(circuitID, carName string) *models.Lap {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := sessionKey(circuitID, carName)
	s, ok := m.sessions[key]
	if !ok || s.BestTime <= 0 {
		return nil
	}
	for _, l := range s.Laps {
		if l.LapFinishTime == s.BestTime {
			return cloneLap(l)
		}
	}
	return nil
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
	if IsRankableLap(lap) && (s.BestTime == 0 || lap.LapFinishTime < s.BestTime) {
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
	normalizeLapMetadata(laps)
	if replace {
		m.laps, _ = deduplicateLaps(laps)
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
	m.rebuildSessions()
}

func dedupKey(l *models.Lap) string {
	if l == nil {
		return "<nil>"
	}

	normalized := *l
	normalized.TimeDiff = nil

	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Sprintf("%d-%d-%d-%s-%d", l.LapFinishTime, l.LapTicks, l.CarID, l.CircuitID, l.Number)
	}

	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func deduplicateLaps(laps []*models.Lap) ([]*models.Lap, int) {
	seen := make(map[string]bool, len(laps))
	result := make([]*models.Lap, 0, len(laps))
	duplicates := 0

	for _, lap := range laps {
		key := dedupKey(lap)
		if seen[key] {
			duplicates++
			continue
		}
		seen[key] = true
		result = append(result, lap)
	}

	return result, duplicates
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
	m.rebuildSessions()
}

func (m *Manager) ClearLaps() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.laps = make([]*models.Lap, 0)
	m.rebuildSessions()
}

func (m *Manager) ClearAllLapData() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.laps = make([]*models.Lap, 0)
	m.sessions = make(map[string]*models.Session)
	m.currentLap = nil
	m.previousLapNum = -1
	m.specialPacketTime = 0
	if err := m.closeCurrentLapFile(); err != nil {
		return fmt.Errorf("close current lap file: %w", err)
	}
	if m.currentLapSavePath != "" {
		if err := os.MkdirAll(filepath.Dir(m.currentLapSavePath), 0755); err != nil {
			return fmt.Errorf("create current lap dir: %w", err)
		}
		if err := os.WriteFile(m.currentLapSavePath, nil, 0644); err != nil {
			return fmt.Errorf("clear current lap file: %w", err)
		}
	}
	return nil
}

func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.laps = make([]*models.Lap, 0)
	m.sessions = make(map[string]*models.Session)
	m.currentLap = nil
	m.previousLapNum = -1
	m.specialPacketTime = 0
	if err := m.closeCurrentLapFile(); err != nil {
		log.Printf("close current lap file: %v", err)
	}
}

func (m *Manager) CurrentLapTicks() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
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
	m.mu.RLock()
	laps := make([]*models.Lap, len(m.laps))
	copy(laps, m.laps)
	savePath := m.savePath
	m.mu.RUnlock()

	if savePath == "" {
		return nil
	}

	return writeLapsFile(savePath, laps)
}

func writeLapsFile(path string, laps []*models.Lap) error {
	data, err := json.Marshal(laps)
	if err != nil {
		return fmt.Errorf("marshal laps: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create laps dir: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write laps temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace laps file: %w", err)
	}
	return nil
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
	normalizeLapMetadata(laps)
	laps, duplicates := deduplicateLaps(laps)
	m.mu.Lock()
	m.laps = laps
	m.rebuildSessions()
	m.mu.Unlock()

	if duplicates > 0 {
		if err := writeLapsFile(path, laps); err != nil {
			return fmt.Errorf("rewrite deduplicated laps file: %w", err)
		}
		log.Printf("loaded %d laps from %s, removed %d duplicates and rewrote file", len(laps), path, duplicates)
		return nil
	}

	log.Printf("loaded %d laps from %s", len(laps), path)
	return nil
}

func normalizeLapMetadata(laps []*models.Lap) {
	for _, lap := range laps {
		if lap == nil {
			continue
		}
		lap.IsComplete = IsCompleteLap(lap)
	}
}

// Current lap JSONL persistence

func (m *Manager) SetCurrentLapSavePath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentLapSavePath = path
}

func (m *Manager) openCurrentLapFile() error {
	if err := os.MkdirAll(filepath.Dir(m.currentLapSavePath), 0755); err != nil {
		return fmt.Errorf("create current lap dir: %w", err)
	}
	f, err := os.OpenFile(m.currentLapSavePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open current lap file: %w", err)
	}
	m.currentLapFile = f
	m.currentLapBuf = bufio.NewWriter(f)
	return nil
}

func (m *Manager) openCurrentLapFileForAppend() error {
	if err := os.MkdirAll(filepath.Dir(m.currentLapSavePath), 0755); err != nil {
		return fmt.Errorf("create current lap dir: %w", err)
	}
	f, err := os.OpenFile(m.currentLapSavePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open current lap file for append: %w", err)
	}
	m.currentLapFile = f
	m.currentLapBuf = bufio.NewWriter(f)
	return nil
}

func (m *Manager) closeCurrentLapFile() error {
	var closeErr error
	if m.currentLapBuf != nil {
		if err := m.currentLapBuf.Flush(); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
		m.currentLapBuf = nil
	}
	if m.currentLapFile != nil {
		if err := m.currentLapFile.Close(); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
		m.currentLapFile = nil
	}
	return closeErr
}

func (m *Manager) writeCurrentLapHeader() error {
	if m.currentLapBuf == nil || m.currentLap == nil {
		return nil
	}
	h := currentLapHeader{
		StartTime:          m.currentLap.startTime,
		FuelAtStart:        m.currentLap.fuelAtStart,
		TotalLaps:          m.currentLap.totalLaps,
		Number:             m.currentLap.number,
		CarID:              m.currentLap.carID,
		CarName:            m.currentLap.carName,
		CircuitID:          m.currentLap.circuitID,
		CircuitName:        m.currentLap.circuitName,
		CircuitVariation:   m.currentLap.circuitVariation,
		StartsAtTrackStart: m.currentLap.startsAtTrackStart,
	}
	data, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("marshal current lap header: %w", err)
	}
	if _, err := m.currentLapBuf.Write(data); err != nil {
		return fmt.Errorf("write current lap header: %w", err)
	}
	if err := m.currentLapBuf.WriteByte('\n'); err != nil {
		return fmt.Errorf("write current lap header newline: %w", err)
	}
	return nil
}

func (m *Manager) appendCurrentLapLine(speed, throttle, brake float64, rpm float64, gear int,
	tireRatios []float64, boost float64, yaw float64,
	posX, posY, posZ float64, tyreTemps []float64, lapNum int16) error {

	if m.currentLapSavePath == "" {
		return nil
	}
	if m.currentLapBuf == nil {
		if err := m.openCurrentLapFile(); err != nil {
			return err
		}
		if err := m.writeCurrentLapHeader(); err != nil {
			return err
		}
	}
	if m.currentLapBuf == nil {
		return nil
	}
	line := currentLapLine{
		Lap:   lapNum,
		Speed: speed, Throttle: throttle, Brake: brake,
		RPM: rpm, Gear: gear,
		TireRatios: tireRatios, Boost: boost, Yaw: yaw,
		PosX: posX, PosY: posY, PosZ: posZ,
		TyreTemps: tyreTemps,
	}
	data, err := json.Marshal(line)
	if err != nil {
		return fmt.Errorf("marshal current lap line: %w", err)
	}
	if _, err := m.currentLapBuf.Write(data); err != nil {
		return fmt.Errorf("write current lap line: %w", err)
	}
	if err := m.currentLapBuf.WriteByte('\n'); err != nil {
		return fmt.Errorf("write current lap line newline: %w", err)
	}
	return nil
}

// SaveCurrentLap flushes the current lap file for shutdown/crash safety.
func (m *Manager) SaveCurrentLap() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentLapBuf != nil {
		if err := m.currentLapBuf.Flush(); err != nil {
			return err
		}
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
		DataSpeed:                    copyFloat64Slice(cl.dataSpeed),
		DataThrottle:                 copyFloat64Slice(cl.dataThrottle),
		DataBraking:                  copyFloat64Slice(cl.dataBraking),
		DataCoasting:                 copyIntSlice(cl.dataCoasting),
		DataRPM:                      copyFloat64Slice(cl.dataRPM),
		DataGear:                     copyIntSlice(cl.dataGear),
		DataTires:                    copyFloat64Slice(cl.dataTires),
		DataBoost:                    copyFloat64Slice(cl.dataBoost),
		DataRotationYaw:              copyFloat64Slice(cl.dataRotationYaw),
		DataAbsoluteYawRatePerSecond: copyFloat64Slice(cl.dataAbsoluteYawRatePerSecond),
		DataPositionX:                copyFloat64Slice(cl.dataPositionX),
		DataPositionY:                copyFloat64Slice(cl.dataPositionY),
		DataPositionZ:                copyFloat64Slice(cl.dataPositionZ),
		TotalDistance:                computeTotalDistance(cl.dataPositionX, cl.dataPositionZ),
		FuelAtStart:                  cl.fuelAtStart,
		FuelAtEnd:                    cl.fuelAtStart,
		FuelConsumed:                 0,
		IsLive:                       true,
		LapTicks:                     cl.lapTicks,
		YawHistory:                   copyFloat64Slice(cl.rotationYawHistory),
		ThrottleAndBrakeTicks:        cl.throttleAndBrakeTicks,
		NoThrottleAndNoBrakeTicks:    cl.noThrottleAndNoBrakeTicks,
		FullBrakeTicks:               cl.fullBrakeTicks,
		FullThrottleTicks:            cl.fullThrottleTicks,
		Number:                       cl.number,
		TotalLaps:                    cl.totalLaps,
		CarName:                      cl.carName,
		CircuitID:                    cl.circuitID,
		CircuitName:                  cl.circuitName,
		CircuitVariation:             cl.circuitVariation,
		StartsAtTrackStart:           cl.startsAtTrackStart,
	}
	return state
}

// GetCurrentLapTimeDiff computes the time difference of the in-progress lap
// against the given reference lap.
func (m *Manager) GetCurrentLapTimeDiff(ref *models.Lap) *models.TimeDiffResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cl := m.currentLap
	if cl == nil || len(cl.dataPositionX) < 10 {
		return nil
	}
	comp := &models.Lap{
		DataPositionX: copyFloat64Slice(cl.dataPositionX),
		DataPositionZ: copyFloat64Slice(cl.dataPositionZ),
	}
	return ComputeTimeDiff(ref, comp)
}

// parseCurrentLapJSONL parses a JSONL current lap file (header + data lines)
// and returns a populated CurrentLap. Returns nil if file has no data.
func parseCurrentLapJSONL(f *os.File) (*CurrentLap, error) {
	var h *currentLapHeader
	var lines []currentLapLine

	dec := json.NewDecoder(f)
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode current lap line: %w", err)
		}
		if h == nil {
			var header currentLapHeader
			if err := json.Unmarshal(raw, &header); err == nil && header.Number > 0 {
				h = &header
				continue
			}
		}
		var line currentLapLine
		if err := json.Unmarshal(raw, &line); err != nil {
			return nil, fmt.Errorf("unmarshal current lap data: %w", err)
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return nil, nil
	}

	cl := &CurrentLap{}
	if h != nil {
		cl.startTime = h.StartTime
		cl.fuelAtStart = h.FuelAtStart
		cl.totalLaps = h.TotalLaps
		cl.number = h.Number
		cl.carID = h.CarID
		cl.carName = h.CarName
		cl.circuitID = h.CircuitID
		cl.circuitName = h.CircuitName
		cl.circuitVariation = h.CircuitVariation
		cl.startsAtTrackStart = h.StartsAtTrackStart
	}

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

	// Fallback: extract lap number from data if header missing
	if h == nil && len(cl.dataLapNum) > 0 {
		cl.number = int(cl.dataLapNum[len(cl.dataLapNum)-1])
	}

	return cl, nil
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

	cl, err := parseCurrentLapJSONL(f)
	if err != nil {
		return err
	}
	if cl == nil || cl.lapTicks == 0 {
		log.Printf("no valid data in current lap file: %s", path)
		return nil
	}

	if discarded := cl.TrimToLastLap(); discarded > 0 {
		log.Printf("current lap file contained %d data points across multiple laps; truncated to last lap (%d ticks)",
			len(cl.dataPositionX)+discarded, cl.lapTicks)
	}

	m.mu.Lock()
	m.currentLap = cl
	if m.currentLapSavePath != "" {
		if err := m.closeCurrentLapFile(); err != nil {
			log.Printf("close current lap file: %v", err)
		}
		if err := m.openCurrentLapFileForAppend(); err != nil {
			m.mu.Unlock()
			return err
		}
	}
	m.mu.Unlock()

	log.Printf("loaded current lap (lap %d, %d ticks) from %s", cl.number, cl.lapTicks, path)
	return nil
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

func isPitStopLap(speeds, posX, posY, posZ []float64, lapNums []int16) bool {
	const jumpDistanceMeters = 10.0
	const movingSpeedKPH = 1.0

	n := min(len(speeds), len(posX), len(posY), len(posZ))
	if n < 2 {
		return false
	}

	hasLapNums := len(lapNums) >= n
	for i := 1; i < n; i++ {
		if hasLapNums && lapNums[i] != lapNums[i-1] {
			continue
		}
		if speeds[i] <= movingSpeedKPH {
			continue
		}

		dx := posX[i] - posX[i-1]
		dy := posY[i] - posY[i-1]
		dz := posZ[i] - posZ[i-1]
		if math.Sqrt(dx*dx+dy*dy+dz*dz) > jumpDistanceMeters {
			return true
		}
	}

	return false
}

func isLapComplete(posX, posY, posZ []float64) bool {
	// A complete lap ends near where it started (close to the start/finish line).
	n := len(posX)
	if len(posY) < n {
		n = len(posY)
	}
	if len(posZ) < n {
		n = len(posZ)
	}
	if n < 2 {
		return false
	}
	dx := posX[n-1] - posX[0]
	dy := posY[n-1] - posY[0]
	dz := posZ[n-1] - posZ[0]
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	return dist < 30 // meters from start point
}

// computeTotalDistance returns the total 2D distance traveled (in meters)
// by summing consecutive position deltas on the X-Z plane.
func computeTotalDistance(posX, posZ []float64) float64 {
	n := len(posX)
	if nz := len(posZ); nz < n {
		n = nz
	}
	if n < 2 {
		return 0
	}
	total := 0.0
	for i := 1; i < n; i++ {
		dx := posX[i] - posX[i-1]
		dz := posZ[i] - posZ[i-1]
		total += math.Sqrt(dx*dx + dz*dz)
	}
	return total
}

// IsCurrentLapActive returns true if there is an in-progress lap.
func (m *Manager) IsCurrentLapActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentLap != nil
}
