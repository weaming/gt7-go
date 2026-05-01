package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/coder/websocket"

	"github.com/weaming/gt7-go/internal/hub"
	"github.com/weaming/gt7-go/internal/lap"
	"github.com/weaming/gt7-go/internal/models"
	"github.com/weaming/gt7-go/internal/telemetry"
)

var wsIDCounter atomic.Int64

func wsHandler(h *hub.Hub, lapMgr *lap.Manager, engine *telemetry.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("ws accept error: %v", err)
			return
		}

		id := wsIDCounter.Add(1)
		client := &hub.Client{
			ID:   fmt.Sprintf("ws-%d", id),
			Send: make(chan []byte, 256),
		}

		h.Register <- client

		// Send current state to newly connected client (always, even empty, for sync)
		laps := lapMgr.GetLaps()
		var bestTime int64
		for _, l := range laps {
			if l.LapFinishTime > 0 && (bestTime == 0 || l.LapFinishTime < bestTime) {
				bestTime = l.LapFinishTime
			}
		}
		lap.ComputeAllTimeDiffs(laps)
		msg, _ := json.Marshal(models.LapsUpdatedMessage{
			Type:     "laps_updated",
			Laps:     laps,
			BestTime: bestTime,
		})
		if msg != nil {
			select {
			case client.Send <- msg:
			default:
			}
		}

		// Send current in-progress lap so frontend can resume.
		// Skip when there's no accumulated data (e.g. during replay without forceRecord).
		if cur := lapMgr.GetCurrentLapState(); cur != nil && cur.LapTicks > 0 {
			log.Printf("ws: sending current_lap to new client (num=%d ticks=%d)", cur.Number, cur.LapTicks)
			msg, _ := json.Marshal(cur)
			if msg != nil {
				select {
				case client.Send <- msg:
				default:
				}
			}
		}

		// Send current replay record state
		msg, _ = json.Marshal(map[string]any{
			"type":    "replay_record_state",
			"enabled": engine.IsForceRecording(),
		})
		if msg != nil {
			select {
			case client.Send <- msg:
			default:
			}
		}

		go wsWritePump(conn, client)
		wsReadPump(conn, client, h, engine)
	}
}

func wsWritePump(conn *websocket.Conn, client *hub.Client) {
	for msg := range client.Send {
		err := conn.Write(context.Background(), websocket.MessageText, msg)
		if err != nil {
			break
		}
	}
}

func wsReadPump(conn *websocket.Conn, client *hub.Client, h *hub.Hub, engine *telemetry.Engine) {
	defer func() {
		h.Unregister <- client
		_ = conn.Close(websocket.StatusNormalClosure, "connection closed")
	}()

	for {
		_, msg, err := conn.Read(context.Background())
		if err != nil {
			break
		}

		var cmd struct {
			Cmd     string          `json:"cmd"`
			ID      json.RawMessage `json:"id,omitempty"`
			Enabled *bool           `json:"enabled,omitempty"`
		}
		if err := json.Unmarshal(msg, &cmd); err != nil {
			continue
		}

		switch cmd.Cmd {
		case "set_replay_record":
			if cmd.Enabled == nil {
				continue
			}
			engine.SetForceRecord(*cmd.Enabled)
			log.Printf("replay recording: %v", *cmd.Enabled)
			h.Broadcast(map[string]any{
				"type":    "replay_record_state",
				"enabled": *cmd.Enabled,
			})
		default:
			log.Printf("ws command: %s", cmd.Cmd)
		}
	}
}
