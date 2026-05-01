package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/weaming/gt7-go/internal/forwarder"
	"github.com/weaming/gt7-go/internal/hub"
	"github.com/weaming/gt7-go/internal/lap"
	"github.com/weaming/gt7-go/internal/models"
	"github.com/weaming/gt7-go/internal/recorder"
	"github.com/weaming/gt7-go/internal/telemetry"
)

//go:generate rm -rf web
//go:generate cp -R ../../web web

//go:embed web
var embeddedWeb embed.FS

type Server struct {
	mux        *http.ServeMux
	hub        *hub.Hub
	lapManager *lap.Manager
	telemetry  *telemetry.Engine
	recorder   *recorder.Recorder
	forwarder  *forwarder.Forwarder
	dataDir    string
}

type ServerInterface interface {
	http.Handler
}

type lapArchive struct {
	Version int           `json:"version"`
	SavedAt time.Time     `json:"saved_at"`
	Label   string        `json:"label"`
	Laps    []*models.Lap `json:"laps"`
}

type lapFileInfo struct {
	Filename  string    `json:"filename"`
	Label     string    `json:"label"`
	SavedAt   time.Time `json:"saved_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LapCount  int       `json:"lap_count"`
}

func New(
	h *hub.Hub,
	lapMgr *lap.Manager,
	telem *telemetry.Engine,
	rec *recorder.Recorder,
	fwd *forwarder.Forwarder,
	dataDir string,
) *Server {
	s := &Server{
		mux:        http.NewServeMux(),
		hub:        h,
		lapManager: lapMgr,
		telemetry:  telem,
		recorder:   rec,
		forwarder:  fwd,
		dataDir:    dataDir,
	}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.Handle("GET /api/laps", jsonHandler(s.getLaps))
	s.mux.Handle("GET /api/sessions", jsonHandler(s.getSessions))
	s.mux.Handle("DELETE /api/laps", jsonHandler(s.deleteLaps))
	s.mux.Handle("POST /api/laps/clear", jsonHandler(s.clearLaps))
	s.mux.Handle("GET /api/lap-files", jsonHandler(s.listLapFiles))
	s.mux.Handle("POST /api/lap-files/save", jsonHandler(s.saveLapFile))
	s.mux.Handle("POST /api/lap-files/load", jsonHandler(s.loadLapFile))
	s.mux.Handle("DELETE /api/lap-files", jsonHandler(s.deleteLapFile))
	s.mux.Handle("GET /api/telemetry/last", jsonHandler(s.getLastTelemetry))
	s.mux.Handle("GET /api/engine/status", jsonHandler(s.getEngineStatus))
	s.mux.Handle("POST /api/recording/start", jsonHandler(s.startRecording))
	s.mux.Handle("POST /api/recording/stop", jsonHandler(s.stopRecording))
	s.mux.Handle("GET /api/recording/status", jsonHandler(s.getRecordingStatus))
	s.mux.Handle("POST /api/forwarder/target", jsonHandler(s.setForwarderTarget))
	s.mux.Handle("GET /api/forwarder/status", jsonHandler(s.getForwarderStatus))
	s.mux.Handle("GET /ws", wsHandler(s.hub, s.lapManager, s.telemetry))

	sub, _ := fs.Sub(embeddedWeb, "web")
	fileServer := http.FileServer(http.FS(sub))
	s.mux.Handle("GET /", fileServer)
}

func jsonHandler(fn func(w http.ResponseWriter, r *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := cors(w, r); err != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := fn(w, r); err != nil {
			log.Printf("api error: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		}
	})
}

func cors(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return fmt.Errorf("options")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func (s *Server) getLaps(w http.ResponseWriter, r *http.Request) error {
	return writeJSON(w, s.lapsUpdatedMessage())
}

func (s *Server) lapsUpdatedMessage() models.LapsUpdatedMessage {
	laps := s.lapManager.GetLaps()

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

	return models.LapsUpdatedMessage{
		Type:     "laps_updated",
		Laps:     laps,
		BestTime: bestTime,
	}
}

func (s *Server) broadcastLapsUpdated() {
	s.hub.Broadcast(s.lapsUpdatedMessage())
}

func (s *Server) broadcastCurrentLapCleared() {
	s.hub.Broadcast(map[string]any{"type": "current_lap_cleared"})
}

func (s *Server) getSessions(w http.ResponseWriter, r *http.Request) error {
	return writeJSON(w, s.lapManager.GetSessions())
}

func (s *Server) getLastTelemetry(w http.ResponseWriter, r *http.Request) error {
	snapshot := s.telemetry.GetLastSnapshot()
	if snapshot == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	return writeJSON(w, models.TelemetryMessage{
		Type: "telemetry",
		Data: snapshot,
	})
}

func (s *Server) getEngineStatus(w http.ResponseWriter, r *http.Request) error {
	snapshot := s.telemetry.GetLastSnapshot()
	status := map[string]any{
		"connected": snapshot != nil,
	}
	if snapshot != nil {
		status["sequence_id"] = snapshot.SequenceID
		status["current_lap"] = snapshot.CurrentLap
	}

	replay, _ := s.telemetry.GetClient().IsReplaySource()
	status["is_replay"] = replay

	return writeJSON(w, status)
}

func (s *Server) deleteLaps(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Indices []int `json:"indices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	s.lapManager.DeleteLaps(req.Indices)
	if err := s.lapManager.SaveLaps(); err != nil {
		return fmt.Errorf("save laps: %w", err)
	}
	s.broadcastLapsUpdated()
	return writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) clearLaps(w http.ResponseWriter, r *http.Request) error {
	if err := s.lapManager.ClearAllLapData(); err != nil {
		return err
	}
	if err := s.lapManager.SaveLaps(); err != nil {
		return fmt.Errorf("save laps: %w", err)
	}
	s.broadcastLapsUpdated()
	s.broadcastCurrentLapCleared()
	return writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) listLapFiles(w http.ResponseWriter, r *http.Request) error {
	files, err := s.readLapFileInfos()
	if err != nil {
		return err
	}
	return writeJSON(w, map[string][]lapFileInfo{"files": files})
}

func (s *Server) saveLapFile(w http.ResponseWriter, r *http.Request) error {
	laps := s.lapManager.GetLaps()
	if len(laps) == 0 {
		return writeJSON(w, map[string]string{"status": "empty"})
	}

	archive := lapArchive{
		Version: 1,
		SavedAt: archiveStartTime(laps),
		Label:   buildLapArchiveLabel(laps),
		Laps:    laps,
	}
	filename, err := buildLapArchiveFilename(archive)
	if err != nil {
		return err
	}
	path := filepath.Join(s.lapArchiveDir(), filename)
	if err := writeLapArchive(path, archive); err != nil {
		return err
	}
	if err := s.removeRenamedLapArchives(archive, filename); err != nil {
		log.Printf("remove renamed lap archives: %v", err)
	}

	if err := s.lapManager.ClearAllLapData(); err != nil {
		return err
	}
	if err := s.lapManager.SaveLaps(); err != nil {
		return fmt.Errorf("save cleared laps: %w", err)
	}
	s.broadcastLapsUpdated()
	s.broadcastCurrentLapCleared()

	return writeJSON(w, map[string]string{"status": "ok", "filename": filename})
}

func (s *Server) loadLapFile(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	path, err := s.lapArchivePath(req.Filename)
	if err != nil {
		return err
	}
	archive, err := readLapArchive(path)
	if err != nil {
		return err
	}

	s.lapManager.LoadLaps(archive.Laps, true)
	if err := s.lapManager.SaveLaps(); err != nil {
		return fmt.Errorf("save loaded laps: %w", err)
	}
	s.broadcastLapsUpdated()

	return writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) deleteLapFile(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	path, err := s.lapArchivePath(req.Filename)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return writeJSON(w, map[string]string{"status": "ok"})
		}
		return fmt.Errorf("delete lap file: %w", err)
	}
	return writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) lapArchiveDir() string {
	return filepath.Join(s.dataDir, "lap_sets")
}

func (s *Server) lapArchivePath(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}
	if filename != filepath.Base(filename) || !strings.HasSuffix(filename, ".json") {
		return "", fmt.Errorf("invalid lap file name")
	}
	return filepath.Join(s.lapArchiveDir(), filename), nil
}

func (s *Server) readLapFileInfos() ([]lapFileInfo, error) {
	dir := s.lapArchiveDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []lapFileInfo{}, nil
		}
		return nil, fmt.Errorf("read lap archive dir: %w", err)
	}

	files := make([]lapFileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		archive, err := readLapArchive(path)
		if err != nil {
			log.Printf("skip lap archive %s: %v", entry.Name(), err)
			continue
		}
		info := lapFileInfo{
			Filename: entry.Name(),
			Label:    archive.Label,
			SavedAt:  archive.SavedAt,
			LapCount: len(archive.Laps),
		}
		stat, err := entry.Info()
		if err == nil {
			info.UpdatedAt = stat.ModTime()
			if info.SavedAt.IsZero() {
				info.SavedAt = stat.ModTime()
			}
		}
		files = append(files, info)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].UpdatedAt.After(files[j].UpdatedAt)
	})

	return files, nil
}

func (s *Server) removeRenamedLapArchives(archive lapArchive, keepFilename string) error {
	dir := s.lapArchiveDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read lap archive dir: %w", err)
	}

	archiveTrackName, _ := archiveNameParts(archive.Laps)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == keepFilename || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		existingArchive, err := readLapArchive(path)
		if err != nil {
			continue
		}
		existingTrackName, _ := archiveNameParts(existingArchive.Laps)
		if !existingArchive.SavedAt.Equal(archive.SavedAt) || existingTrackName != archiveTrackName {
			continue
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove renamed lap archive %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func writeLapArchive(path string, archive lapArchive) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create lap archive dir: %w", err)
	}

	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lap archive: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write lap archive temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace lap archive file: %w", err)
	}
	return nil
}

func readLapArchive(path string) (lapArchive, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return lapArchive{}, fmt.Errorf("read lap archive: %w", err)
	}

	var archive lapArchive
	if err := json.Unmarshal(data, &archive); err == nil && archive.Laps != nil {
		archive.Laps, _ = deduplicateArchiveLaps(archive.Laps)
		if archive.Label == "" {
			archive.Label = buildLapArchiveLabel(archive.Laps)
		}
		return archive, nil
	}

	var laps []*models.Lap
	if err := json.Unmarshal(data, &laps); err != nil {
		return lapArchive{}, fmt.Errorf("unmarshal lap archive: %w", err)
	}
	laps, _ = deduplicateArchiveLaps(laps)
	return lapArchive{
		Version: 1,
		Label:   buildLapArchiveLabel(laps),
		Laps:    laps,
	}, nil
}

func buildLapArchiveFilename(archive lapArchive) (string, error) {
	trackName, carName := archiveNameParts(archive.Laps)
	if archive.SavedAt.IsZero() {
		archive.SavedAt = archiveStartTime(archive.Laps)
	}
	timestamp := shortArchiveTimestamp(archive.SavedAt)
	if archive.SavedAt.IsZero() {
		timestamp = "legacy_" + stableArchiveFallbackID(archive.Laps)
	}

	return fmt.Sprintf(
		"%s__%s__%s.json",
		timestamp,
		shortArchiveName(trackName, 16),
		shortArchiveName(carName, 24),
	), nil
}

func shortArchiveTimestamp(value time.Time) string {
	return value.Format("20060102_1504")
}

func archiveStartTime(laps []*models.Lap) time.Time {
	var firstTick time.Time
	for _, lap := range laps {
		if lap == nil || lap.LapStartTimestamp == nil || lap.LapStartTimestamp.IsZero() {
			continue
		}
		if firstTick.IsZero() || lap.LapStartTimestamp.Before(firstTick) {
			firstTick = *lap.LapStartTimestamp
		}
	}
	if !firstTick.IsZero() {
		return firstTick
	}

	for _, lap := range laps {
		if lap == nil || lap.LapEndTimestamp == nil || lap.LapEndTimestamp.IsZero() {
			continue
		}
		if firstTick.IsZero() || lap.LapEndTimestamp.Before(firstTick) {
			firstTick = *lap.LapEndTimestamp
		}
	}
	return firstTick
}

func stableArchiveFallbackID(laps []*models.Lap) string {
	for i := len(laps) - 1; i >= 0; i-- {
		lap := laps[i]
		if lap == nil {
			continue
		}
		return fmt.Sprintf("%d_%d_%d_%s_%d", lap.LapFinishTime, lap.LapTicks, lap.CarID, shortArchiveName(lap.CircuitID, 16), lap.Number)
	}
	return "empty"
}

func buildLapArchiveLabel(laps []*models.Lap) string {
	trackName, carName := archiveNameParts(laps)
	return fmt.Sprintf("%s / %s / %d laps", trackName, carName, len(laps))
}

func archiveNameParts(laps []*models.Lap) (string, string) {
	trackNames := make(map[string]bool)
	carNames := make(map[string]bool)

	for _, lap := range laps {
		if lap == nil {
			continue
		}
		trackName := lap.CircuitName
		if trackName == "" {
			trackName = lap.CircuitID
		}
		if trackName != "" {
			trackNames[trackName] = true
		}
		if lap.CarName != "" {
			carNames[lap.CarName] = true
		}
	}

	return mapSummary(trackNames, "mixed_tracks", "unknown_track"), mapSummary(carNames, "mixed_cars", "unknown_car")
}

func mapSummary(values map[string]bool, mixedValue string, emptyValue string) string {
	if len(values) == 0 {
		return emptyValue
	}
	if len(values) > 1 {
		return mixedValue
	}
	for value := range values {
		return value
	}
	return emptyValue
}

var archiveNameInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeArchiveName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "unknown"
	}
	value = archiveNameInvalidChars.ReplaceAllString(value, "_")
	value = strings.Trim(value, "._-")
	if value == "" {
		return "unknown"
	}
	if len(value) > 48 {
		return value[:48]
	}
	return value
}

func shortArchiveName(value string, maxLength int) string {
	value = sanitizeArchiveName(value)
	switch value {
	case "mixed_tracks":
		return "mixed"
	case "unknown_track":
		return "track"
	case "mixed_cars":
		return "cars"
	case "unknown_car":
		return "car"
	}

	tokens := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	filteredTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token == "" || isArchiveNameFiller(token) {
			continue
		}
		filteredTokens = append(filteredTokens, token)
	}
	if len(filteredTokens) == 0 {
		filteredTokens = tokens
	}

	shortName := strings.Join(filteredTokens, "")
	if shortName == "" {
		shortName = "unknown"
	}
	if len(shortName) <= maxLength {
		return shortName
	}
	return shortName[:maxLength]
}

func isArchiveNameFiller(value string) bool {
	switch strings.ToLower(value) {
	case "the", "of", "and", "circuit", "speedway", "international", "autodrome", "super", "formula":
		return true
	default:
		return false
	}
}

func deduplicateArchiveLaps(laps []*models.Lap) ([]*models.Lap, int) {
	manager := lap.NewManager(nil)
	manager.LoadLaps(laps, true)
	deduplicated := manager.GetLaps()
	return deduplicated, len(laps) - len(deduplicated)
}

func (s *Server) startRecording(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Filename = ""
	}

	var name string
	var err error
	if req.Filename != "" {
		name = req.Filename
		err = s.recorder.Start(name)
	} else {
		name, err = s.recorder.StartTimestamped("recording", "gtz")
	}
	if err != nil {
		return fmt.Errorf("start recording: %w", err)
	}
	return writeJSON(w, map[string]string{"status": "ok", "filename": name})
}

func (s *Server) stopRecording(w http.ResponseWriter, r *http.Request) error {
	if err := s.recorder.Stop(); err != nil {
		return fmt.Errorf("stop recording: %w", err)
	}
	return writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) getRecordingStatus(w http.ResponseWriter, r *http.Request) error {
	return writeJSON(w, map[string]bool{
		"recording": s.recorder.IsRecording(),
	})
}

func (s *Server) setForwarderTarget(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	if err := s.forwarder.SetTarget(req.Target); err != nil {
		return fmt.Errorf("set forwarder target: %w", err)
	}
	return writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) getForwarderStatus(w http.ResponseWriter, r *http.Request) error {
	return writeJSON(w, map[string]any{
		"running": s.forwarder.IsRunning(),
	})
}
