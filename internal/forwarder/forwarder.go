package forwarder

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type Forwarder struct {
	mu         sync.RWMutex
	targetAddr *net.UDPAddr
	conn       *net.UDPConn
	running    bool
	done       chan struct{}

	playstationIP string
}

func New(playstationIP string) *Forwarder {
	return &Forwarder{
		playstationIP: playstationIP,
	}
}

func (f *Forwarder) SetTarget(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolve target address: %w", err)
	}
	f.mu.Lock()
	f.targetAddr = udpAddr
	f.mu.Unlock()
	return nil
}

func (f *Forwarder) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return nil
	}

	sourceAddr := "0.0.0.0:33739"
	if f.playstationIP != "" {
		sourceAddr = f.playstationIP + ":33739"
	}

	udpAddr, err := net.ResolveUDPAddr("udp", sourceAddr)
	if err != nil {
		return fmt.Errorf("resolve source address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen UDP: %w", err)
	}
	f.conn = conn
	f.running = true
	f.done = make(chan struct{})

	go f.forwardLoop()
	return nil
}

func (f *Forwarder) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running && f.conn != nil {
		f.conn.Close()
		close(f.done)
		f.running = false
	}
}

func (f *Forwarder) IsRunning() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.running
}

func (f *Forwarder) forwardLoop() {
	buf := make([]byte, 65535)
	for {
		n, _, err := f.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		f.mu.RLock()
		target := f.targetAddr
		f.mu.RUnlock()

		if target != nil {
			_, err := f.conn.WriteTo(buf[:n], target)
			if err != nil {
				log.Printf("forwarder: write error: %v", err)
			}
		}
	}
}
