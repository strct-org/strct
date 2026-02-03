package monitor

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	ping "github.com/prometheus-community/pro-bing"
)

type NetworkMonitor struct {
	Target string
}

type MonitorStats struct {
	Latency   *float64 `json:"latency,omitempty"`   // ms
	Loss      *float64 `json:"loss,omitempty"`      // %
	IsDown    *bool    `json:"is_down,omitempty"`  
	Bandwidth *float64 `json:"bandwidth,omitempty"` // Pointer to Mbps
}

func New() *NetworkMonitor {
	return &NetworkMonitor{
		Target: "8.8.8.8",
	}
}

func (m *NetworkMonitor) Start() error {
	log.Printf("[MONITOR] Starting Network Health Monitor (Target: %s, Interval: 30s)", m.Target)

	latencyTicker := time.NewTicker(30 * time.Second)
	// bandwidthTicker := time.NewTicker(30 * time.Second)
	bandwidthTicker := time.NewTicker(2 * time.Hour)

	defer latencyTicker.Stop()
	defer bandwidthTicker.Stop()

	for {
		select {
		case <-latencyTicker.C:
			stats, err := m.pingTarget()
			if err != nil {
				log.Printf("[MONITOR] Ping Execution Failed: %v", err)
				continue
			}

			//! send 'stats' to backend.
			// Because Bandwidth is nil, it won't overwrite DB data with 0.
			if stats.IsDown != nil && *stats.IsDown {
				log.Printf("[MONITOR] CRITICAL: Target is DOWN (Loss: %.2f%%)", *stats.Loss)
			} else if stats.Latency != nil && *stats.Latency > 100.0 {
				log.Printf("[MONITOR] High Latency: %.2f ms", *stats.Latency)
			} else {
				log.Printf("[MONITOR] Health OK: %.2f ms", *stats.Latency)
			}

		case <-bandwidthTicker.C:
			stats, err := m.getBandwidth()
			if err != nil {
				log.Printf("[MONITOR] Bandwidth Test Failed: %v", err)
				continue
			}

			//! send 'stats' to backend.
			// Latency/Loss are nil, so they won't mess up the DB.

			if stats.Bandwidth != nil {
				log.Printf("[MONITOR] Bandwidth: %.2f Mbps", *stats.Bandwidth)
			}
		}
	}
}

func (m *NetworkMonitor) pingTarget() (*MonitorStats, error) {
	pinger, err := ping.NewPinger(m.Target)
	if err != nil {
		return nil, err
	}

	pinger.SetPrivileged(true)
	pinger.Count = 3
	pinger.Timeout = 2 * time.Second

	err = pinger.Run()
	if err != nil {
		return nil, err
	}

	pStats := pinger.Statistics()

	latVal := float64(pStats.AvgRtt.Microseconds()) / 1000.0
	lossVal := pStats.PacketLoss
	isDownVal := pStats.PacketLoss >= 100.0

	return &MonitorStats{
		Latency:   &latVal,
		Loss:      &lossVal,
		IsDown:    &isDownVal,
		Bandwidth: nil,
	}, nil
}

func (m *NetworkMonitor) getBandwidth() (*MonitorStats, error) {
	testURL := "http://speedtest.tele2.net/10MB.zip"

	start := time.Now()

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(testURL)
	if err != nil {
		return nil, fmt.Errorf("download start failed: %w", err)
	}
	defer resp.Body.Close()

	written, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download interrupted: %w", err)
	}

	duration := time.Since(start)

	bits := float64(written) * 8
	mbpsVal := (bits / 1_000_000) / duration.Seconds()

	return &MonitorStats{
		Latency:   nil,
		Loss:      nil,
		IsDown:    nil,
		Bandwidth: &mbpsVal,
	}, nil
}
