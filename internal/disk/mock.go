package disk

import (
	"fmt"
	"time"
)

type MockDisk struct {
	VirtualPath string
	IsFormatted bool
}

func (d *MockDisk) GetStatus() (string, error) {
	if d.IsFormatted {
		return "Formatted (Virtual 1TB)", nil
	}
	return "Raw/Unformatted (Virtual 1TB)", nil
}

func (d *MockDisk) Format() error {
	fmt.Printf("[MOCK DISK] Simulating format of %s...\n", d.VirtualPath)
	fmt.Println("[MOCK DISK] Creating GPT Table...")
	time.Sleep(1 * time.Second)
	fmt.Println("[MOCK DISK] Creating Partition...")
	time.Sleep(1 * time.Second)
	fmt.Println("[MOCK DISK] Running mkfs.ext4...")
	time.Sleep(2 * time.Second)
	
	d.IsFormatted = true // Update state in memory
	fmt.Println("[MOCK DISK] Format Complete.")
	return nil
}