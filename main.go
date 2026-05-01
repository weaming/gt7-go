package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/weaming/gt7-go/internal/forwarder"
	"github.com/weaming/gt7-go/internal/hub"
	"github.com/weaming/gt7-go/internal/lap"
	"github.com/weaming/gt7-go/internal/models"
	"github.com/weaming/gt7-go/internal/recorder"
	"github.com/weaming/gt7-go/internal/server"
	"github.com/weaming/gt7-go/internal/telemetry"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	psIP := flag.String("ps", "", "PlayStation 5 IP address (omit for auto-discovery)")
	forward := flag.String("forward", "", "UDP forwarding target address (e.g., 192.168.1.100:33739)")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("get home dir: %v", err)
	}
	defaultData := filepath.Join(homeDir, ".gt7", "data")
	dataDir := flag.String("data", defaultData, "Data directory for recordings and cache")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("create data directory: %v", err)
	}

	h := hub.New()
	go h.Run()

	lapMgr := lap.NewManager(func(l *models.Lap) {
		log.Printf("lap completed: %s", l.Title)
	})

	lapsPath := filepath.Join(*dataDir, "laps.json")
	if err := lapMgr.LoadLapsFromFile(lapsPath); err != nil {
		log.Printf("load laps: %v", err)
	}
	lapMgr.SetSavePath(lapsPath)

	currentLapPath := filepath.Join(*dataDir, "current_lap.jsonl")
	lapMgr.SetCurrentLapSavePath(currentLapPath)
	log.Printf("loading current lap from %s", currentLapPath)
	if err := lapMgr.LoadCurrentLapFromFile(currentLapPath); err != nil {
		log.Printf("load current lap: %v", err)
	}
	if lapMgr.IsCurrentLapActive() {
		log.Printf("current lap loaded: ticks=%d", lapMgr.CurrentLapTicks())
	} else {
		log.Printf("current lap NOT loaded (file may not exist)")
	}

	telem := telemetry.New(h, lapMgr, *psIP)

	fwd := forwarder.New(*psIP)
	if *forward != "" {
		if err := fwd.SetTarget(*forward); err != nil {
			log.Printf("set forwarder target: %v", err)
		}
		if err := fwd.Start(); err != nil {
			log.Printf("start forwarder: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := telem.Start(ctx); err != nil {
		log.Fatalf("start telemetry engine: %v", err)
	}

	rec := recorder.New(telem.GetClient(), *dataDir)

	var srv server.ServerInterface = server.New(h, lapMgr, telem, rec, fwd, *dataDir)

	httpServer := &http.Server{
		Addr:    *addr,
		Handler: srv,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		if err := lapMgr.SaveCurrentLap(); err != nil {
			log.Printf("save current lap on shutdown: %v", err)
		}
		telem.Stop()
		fwd.Stop()
		httpServer.Close()
		cancel()
	}()

	log.Printf("GT7 Dashboard starting on %s", *addr)
	if *psIP != "" {
		log.Printf("PS5 IP: %s", *psIP)
	}
	if *forward != "" {
		log.Printf("UDP forwarding to: %s", *forward)
	}

	fmt.Printf("Open http://localhost%s in your browser\n", *addr)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}
