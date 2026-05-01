package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/weaming/gt7-go/internal/forwarder"
	"github.com/weaming/gt7-go/internal/hub"
	"github.com/weaming/gt7-go/internal/lap"
	"github.com/weaming/gt7-go/internal/models"
	"github.com/weaming/gt7-go/internal/recorder"
	"github.com/weaming/gt7-go/internal/telemetry"
)

type Server struct {
	mux        *http.ServeMux
	hub        *hub.Hub
	lapManager *lap.Manager
	telemetry  *telemetry.Engine
	recorder   *recorder.Recorder
	forwarder  *forwarder.Forwarder
	webDir     string
	dataDir    string
}

func New(
	h *hub.Hub,
	lapMgr *lap.Manager,
	telem *telemetry.Engine,
	rec *recorder.Recorder,
	fwd *forwarder.Forwarder,
	webDir, dataDir string,
) *Server {
	s := &Server{
		mux:        http.NewServeMux(),
		hub:        h,
		lapManager: lapMgr,
		telemetry:  telem,
		recorder:   rec,
		forwarder:  fwd,
		webDir:     webDir,
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
	s.mux.Handle("GET /api/telemetry/last", jsonHandler(s.getLastTelemetry))
	s.mux.Handle("GET /api/engine/status", jsonHandler(s.getEngineStatus))
	s.mux.Handle("POST /api/recording/start", jsonHandler(s.startRecording))
	s.mux.Handle("POST /api/recording/stop", jsonHandler(s.stopRecording))
	s.mux.Handle("GET /api/recording/status", jsonHandler(s.getRecordingStatus))
	s.mux.Handle("POST /api/forwarder/target", jsonHandler(s.setForwarderTarget))
	s.mux.Handle("GET /api/forwarder/status", jsonHandler(s.getForwarderStatus))
	s.mux.Handle("GET /ws", wsHandler(s.hub, s.lapManager))

	fileServer := http.FileServer(http.Dir(s.webDir))
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

func writeJSON(w http.ResponseWriter, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func (s *Server) getLaps(w http.ResponseWriter, r *http.Request) error {
	laps := s.lapManager.GetLaps()

	var bestTime int64 = 1<<63 - 1
	for _, l := range laps {
		if l.LapFinishTime > 0 && l.LapFinishTime < bestTime {
			bestTime = l.LapFinishTime
		}
	}
	if bestTime == 1<<63-1 {
		bestTime = 0
	}

	return writeJSON(w, models.LapsUpdatedMessage{
		Type:     "laps_updated",
		Laps:     laps,
		BestTime: bestTime,
	})
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
	status := map[string]interface{}{
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
	return writeJSON(w, map[string]string{"status": "ok"})
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
	return writeJSON(w, map[string]interface{}{
		"running": s.forwarder.IsRunning(),
	})
}
