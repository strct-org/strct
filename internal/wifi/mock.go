package wifi

import "fmt"

type MockWiFi struct{}

func (m *MockWiFi) Scan() ([]Network, error) {
	return []Network{
		{SSID: "Test_Net", Signal: 99, Security: "WPA2"},
	}, nil
}

func (m *MockWiFi) Connect(ssid, password string) error {
	fmt.Printf("[MOCK] Connected to %s\n", ssid)
	return nil
}