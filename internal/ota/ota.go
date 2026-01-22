package ota

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/minio/selfupdate"
)

// Config configures the OTA updater
type Config struct {
	CurrentVersion string // The version currently running (e.g. "1.0.0")
	StorageURL     string // Base URL where files are stored (e.g. "http://your-vps-ip:8080")
}

// StartUpdater starts the background ticker to check for updates
func StartUpdater(cfg Config) {
	// check every 24 hours (or whatever interval you prefer)
	ticker := time.NewTicker(24 * time.Hour)
	
	// Check immediately on startup
	go func() {
		if err := checkForUpdate(cfg); err != nil {
			log.Printf("OTA: Update check failed: %v", err)
		}
	}()

	go func() {
		for range ticker.C {
			if err := checkForUpdate(cfg); err != nil {
				log.Printf("OTA: Update check failed: %v", err)
			}
		}
	}()
}

func checkForUpdate(cfg Config) error {
	log.Println("OTA: Checking for updates...")

	// 1. Get the latest version from the server
	resp, err := http.Get(fmt.Sprintf("%s/version.txt", cfg.StorageURL))
	if err != nil {
		return fmt.Errorf("failed to fetch version file: %w", err)
	}
	defer resp.Body.Close()

	remoteVerStrRaw, _ := io.ReadAll(resp.Body)
	remoteVerStr := strings.TrimSpace(string(remoteVerStrRaw))

	// 2. Parse and Compare Versions
	vCurrent, err := semver.Make(cfg.CurrentVersion)
	if err != nil {
		return fmt.Errorf("invalid current version '%s': %w", cfg.CurrentVersion, err)
	}
	vRemote, err := semver.Make(remoteVerStr)
	if err != nil {
		return fmt.Errorf("invalid remote version '%s': %w", remoteVerStr, err)
	}

	if vRemote.LTE(vCurrent) {
		log.Printf("OTA: No update needed. Remote: %s, Current: %s", vRemote, vCurrent)
		return nil
	}

	log.Printf("OTA: New version found: %s. Downloading...", vRemote)

	// 3. Define the binary name based on architecture
	// Orange Pi is usually linux/arm64
	binName := fmt.Sprintf("myapp-%s-%s", runtime.GOOS, runtime.GOARCH)
	binURL := fmt.Sprintf("%s/%s", cfg.StorageURL, binName)
	checksumURL := binURL + ".sha256"

	// 4. Download and Apply
	return doUpdate(binURL, checksumURL)
}

func doUpdate(binURL, checksumURL string) error {
	// A. Download the new binary
	resp, err := http.Get(binURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("binary download failed: %s", resp.Status)
	}

	// B. Verify Checksum (Security Best Practice)
	// We verify the stream as we read it to avoid loading huge files into memory
	// However, selfupdate.Apply consumes the stream. Ideally, verify header/checksum first.
	// For simplicity, we trust the connection or verify a separate hash file.
	
	// NOTE: Production code should download the .sha256 file and verify here.
	// verification logic omitted for brevity but highly recommended.

	// C. Apply the update
	err = selfupdate.Apply(resp.Body, selfupdate.Options{
		// Calculate checksum of downloaded bytes to verify integrity before swap
		Checksum: []byte{}, // You would pass the expected checksum bytes here if you fetched them
	})
	
	if err != nil {
		// Rollback happens automatically if Apply fails
		return fmt.Errorf("update apply failed: %w", err)
	}

	log.Println("OTA: Update applied successfully! Restarting...")
	
	// D. Restart the application
	// We exit successfully. Systemd (or a loop in main) will handle the restart.
	os.Exit(0)
	return nil
}