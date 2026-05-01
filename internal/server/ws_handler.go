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
)

var wsIDCounter atomic.Int64

func wsHandler(h *hub.Hub, lapMgr *lap.Manager) http.HandlerFunc {
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

		// Send current in-progress lap so frontend can resume
		if cur := lapMgr.GetCurrentLapState(); cur != nil {
			msg, _ := json.Marshal(cur)
			if msg != nil {
				select {
				case client.Send <- msg:
				default:
				}
			}
		}

		go wsWritePump(conn, client)
		wsReadPump(conn, client, h)
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

func wsReadPump(conn *websocket.Conn, client *hub.Client, h *hub.Hub) {
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
			Cmd string          `json:"cmd"`
			ID  json.RawMessage `json:"id,omitempty"`
		}
		if err := json.Unmarshal(msg, &cmd); err != nil {
			continue
		}

		log.Printf("ws command: %s", cmd.Cmd)
	}
}
