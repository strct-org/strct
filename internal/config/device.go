package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func getOrGenerateDeviceID(isDev bool) string {
	var filePath string

	if isDev {
		filePath = "device-id.lock"
	} else {
		filePath = "/etc/strct/device-id.lock"
	}

	content, err := os.ReadFile(filePath)
	if err == nil {
		return strings.TrimSpace(string(content))
	}

	newID := "device-" + uuid.New().String()
	log.Printf("[INIT] New Device ID generated: %s", newID)

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[WARN] Could not create directory %s: %v", dir, err)
		return newID 
	}

	err = os.WriteFile(filePath, []byte(newID), 0644)
	if err != nil {
		log.Printf("[WARN] Could not save device ID to disk at %s: %v", filePath, err)
	} else {
		log.Printf("[INIT] Device ID saved to %s", filePath)
	}

	return newID
}