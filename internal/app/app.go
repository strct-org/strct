package app

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/strct-org/strct-agent/internal/config"
	"github.com/strct-org/strct-agent/internal/features/cloud"
	"github.com/strct-org/strct-agent/internal/features/monitor"
	"github.com/strct-org/strct-agent/internal/network/dns"
	"github.com/strct-org/strct-agent/internal/network/tunnel"
	"github.com/strct-org/strct-agent/internal/platform/wifi"
	"github.com/strct-org/strct-agent/internal/setup"
)

type Agent struct {
	Config   *config.Config
	Wifi     wifi.Provider
	Services []Service
}

type Service interface {
	Start() error
}

func New(cfg *config.Config) *Agent {
	var wifiMgr wifi.Provider
	if cfg.IsArm64() {
		wifiMgr = &wifi.RealWiFi{Interface: "wlan0"}

	} else {
		wifiMgr = &wifi.MockWiFi{}
	}

	return &Agent{
		Config: cfg,
		Wifi:   wifiMgr,
	}
}

func (a *Agent) Bootstrap() {
	if !a.hasInternet() {
		log.Println("[INIT] No Internet detected. Starting Setup Wizard...")
		a.runSetupWizard()
	} else {
		log.Println("[INIT] Internet detected. Skipping setup.")
	}

	a.Services = []Service{
		cloud.New(a.Config.DataDir, 8080, a.Config.IsDev),
		tunnel.New(a.Config),
		dns.NewAdBlocker(":53"),
		monitor.New(),
	}
}

func (a *Agent) Start() {
	var wg sync.WaitGroup

	log.Println("--- Strct Agent Starting Services ---")

	for _, svc := range a.Services {
		wg.Add(1)
		go func(s Service) {
			defer wg.Done()
			if err := s.Start(); err != nil {
				log.Printf("Service crashed: %v", err)
			}
		}(svc)
	}

	wg.Wait()
}



func (a *Agent) hasInternet() bool {
	client := http.Client{Timeout: 3 * time.Second}
	_, err := client.Get("http://clients3.google.com/generate_204")
	return err == nil
}

func (a *Agent) runSetupWizard() {
	macSuffix := "XXXX" 

	ssid := "Strct-Setup-" + macSuffix
	password := "strct" + macSuffix

	log.Printf("[SETUP] Creating Hotspot. SSID: %s", ssid)

	err := a.Wifi.StartHotspot(ssid, password)
	if err != nil {
		log.Printf("[SETUP] Failed to create hotspot: %v", err)
	}

	done := make(chan bool)
	
	go setup.StartCaptivePortal(a.Wifi, done, a.Config.IsDev)

	log.Println("[SETUP] Waiting for user credentials...")
	<-done 

	a.Wifi.StopHotspot()
	time.Sleep(2 * time.Second)
}