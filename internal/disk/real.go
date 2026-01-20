package disk

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type RealDisk struct {
	DevicePath string
}

type lsblkOutput struct {
	Blockdevices []blockDevice `json:"blockdevices"`
}

type blockDevice struct {
	Name     string        `json:"name"`
	Size     string        `json:"size"`
	Type     string        `json:"type"`
	Children []blockDevice `json:"children,omitempty"`
}

func (d *RealDisk) GetStatus() (string, error) {
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,SIZE,TYPE,MOUNTPOINT", d.DevicePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("disk not found or error reading: %v", err)
	}

	var data lsblkOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return "", fmt.Errorf("failed to parse lsblk output: %v", err)
	}

	if len(data.Blockdevices) == 0 {
		return "Not Found", nil
	}

	// Logic: If the device has "Children" (partitions), it is formatted.
	dev := data.Blockdevices[0]
	status := fmt.Sprintf("Raw/Unformatted (&s)", dev.Size)

	if len(dev.Children) > 0 {
		status = fmt.Sprintf("Formatted (%s)", dev.Size)
	}

	return status, nil
}

func (d *RealDisk) Format() error {
	fmt.Printf("[DISK] REAL FORMATTING INITIATED ON %s\n", d.DevicePath)

	// 2. Create Partition
	if err := exec.Command("parted", d.DevicePath, "--script", "mkpart", "primary", "ext4", "0%", "100%").Run(); err != nil {
		return err
	}

	partPath := d.DevicePath + "1"
	if d.DevicePath ==  "/dev/nvme0n1" {
		partPath = d.DevicePath + "p1"
	}

	// Refresh kernel partition table
	exec.Command("partprobe", d.DevicePath).Run()

	if err := exec.Command("mkfs.ext4", "-F", partPath).Run(); err != nil {
		return err
	}

	return nil
}
