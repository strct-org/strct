package wifi

// Network represents a visible WiFi Access Point
type Network struct {
	SSID     string
	Signal   int
	Security string
}

type Provider interface {
	Scan() ([]Network, error)
	Connect(ssid, password string) error
}