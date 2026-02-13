package tunnel

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/strct-org/strct-agent/internal/config"
)

type Service struct {
	GlobalConfig *config.Config
}

type TemplateData struct {
	ServerIP   string
	Token      string
	DeviceID   string
	ServerPort int
	LocalPort  int
}

const frpConfigTmpl = `
serverAddr = "{{.ServerIP}}"
serverPort = {{.ServerPort}}
auth.token = "{{.Token}}"

[[proxies]]
name = "web_{{.DeviceID}}"
type = "http"
localPort = {{.LocalPort}}
subdomain = "{{.DeviceID}}"
`

func New(cfg *config.Config) *Service {
	return &Service{
		GlobalConfig: cfg,
	}
}

func (s *Service) Start() error {
	// 1. GET PROJECT ROOT (Current Working Directory)
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %v", err)
	}

	// 2. DEFINE PATHS
	// Binary is in Root (strct/frpc)
	frpcBinaryPath := filepath.Join(projectRoot, "frpc")
	// Config goes into DataDir to keep root clean (strct/data/frpc.toml)
	frpcConfigPath := filepath.Join(s.GlobalConfig.DataDir, "frpc.toml")

	// 3. CHECK IF BINARY EXISTS
	if _, err := os.Stat(frpcBinaryPath); os.IsNotExist(err) {
		log.Printf("===============================================================")
		log.Printf("[TUNNEL] CRITICAL ERROR: 'frpc' binary missing!")
		log.Printf("[TUNNEL] We looked here: %s", frpcBinaryPath)
		log.Printf("[TUNNEL] Please run: wget https://github.com/fatedier/frp/releases/download/v0.54.0/frp_0.54.0_linux_amd64.tar.gz")
		log.Printf("===============================================================")
		// Return nil so we don't crash the whole agent, just this feature fails
		return fmt.Errorf("binary not found at %s", frpcBinaryPath)
	}

	// 4. PREPARE CONFIG DATA
	data := TemplateData{
		ServerIP:   s.GlobalConfig.VPSIP,
		ServerPort: s.GlobalConfig.VPSPort,
		Token:      s.GlobalConfig.AuthToken,
		DeviceID:   s.GlobalConfig.DeviceID,
		LocalPort:  8080, 
	}

	log.Printf("[TUNNEL] Configuring for Device: %s -> %s:%d", data.DeviceID, data.ServerIP, data.ServerPort)

	// 5. WRITE CONFIG FILE
	// Ensure data directory exists first
	if err := os.MkdirAll(s.GlobalConfig.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %v", err)
	}

	file, err := os.Create(frpcConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	
	// Write template
	func() {
		defer file.Close()
		tmpl, err := template.New("frpc").Parse(frpConfigTmpl)
		if err != nil {
			log.Printf("[TUNNEL] Template parsing error: %v", err)
			return
		}
		if err := tmpl.Execute(file, data); err != nil {
			log.Printf("[TUNNEL] Template execution error: %v", err)
		}
	}()

	// 6. ENSURE PERMISSIONS (chmod +x)
	// This fixes "permission denied" errors automatically
	if err := os.Chmod(frpcBinaryPath, 0755); err != nil {
		log.Printf("[TUNNEL] Warning: Could not chmod binary: %v", err)
	}

	// 7. RUN LOOP
	go func() {
		for {
			log.Println("[TUNNEL] Starting FRP Client...")
			
			// Command: ./frpc -c ./data/frpc.toml
			cmd := exec.Command(frpcBinaryPath, "-c", frpcConfigPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Start()
			if err != nil {
				log.Printf("[TUNNEL] Failed to start binary: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			err = cmd.Wait()
			log.Printf("[TUNNEL] Process exited: %v. Restarting in 5 seconds...", err)
			time.Sleep(5 * time.Second)
		}
	}()

	return nil
}

// 
// 
// package tunnel

// import (
// 	"fmt"
// 	"html/template"
// 	"log"
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// 	"time"

// 	"github.com/strct-org/strct-agent/internal/config"
// )

// type Service struct {
// 	GlobalConfig *config.Config
// }

// type TemplateData struct {
// 	ServerIP   string
// 	Token      string
// 	DeviceID   string
// 	ServerPort int
// 	LocalPort  int
// }

// const frpConfigTmpl = `
// serverAddr = "{{.ServerIP}}"
// serverPort = {{.ServerPort}}
// auth.token = "{{.Token}}"

// [[proxies]]
// name = "web_{{.DeviceID}}"
// type = "http"
// localPort = {{.LocalPort}}
// subdomain = "{{.DeviceID}}"
// `

// func New(cfg *config.Config) *Service {
// 	return &Service{
// 		GlobalConfig: cfg,
// 	}
// }

// // func (s *Service) Start() error {
// // 	data := TemplateData{
// // 		ServerIP:   s.GlobalConfig.VPSIP,
// // 		ServerPort: s.GlobalConfig.VPSPort,
// // 		Token:      s.GlobalConfig.AuthToken,
// // 		DeviceID:   s.GlobalConfig.DeviceID,
// // 		LocalPort:  8080, 
// // 	}

// // 	log.Printf("[TUNNEL] Configuring for Device: %s -> %s:%d", data.DeviceID, data.ServerIP, data.ServerPort)

// // 	file, err := os.Create("frpc.toml")
// // 	if err != nil {
// // 		return fmt.Errorf("failed to create config file: %v", err)
// // 	}
// // 	defer file.Close()

// // 	tmpl, err := template.New("frpc").Parse(frpConfigTmpl)
// // 	if err != nil {
// // 		return err
// // 	}

// // 	if err := tmpl.Execute(file, data); err != nil {
// // 		return fmt.Errorf("failed to write config: %v", err)
// // 	}

// // 	for {
// // 		log.Println("[TUNNEL] Starting FRP Client...")
		
// // 		cmd := exec.Command("./frpc", "-c", "frpc.toml")
// // 		cmd.Stdout = os.Stdout
// // 		cmd.Stderr = os.Stderr

// // 		err := cmd.Start()
// // 		if err != nil {
// // 			log.Printf("[TUNNEL] Failed to start binary: %v. Is ./frpc inside the folder?", err)
// // 			time.Sleep(10 * time.Second)
// // 			continue
// // 		}

// // 		err = cmd.Wait()
// // 		log.Printf("[TUNNEL] Process exited: %v. Restarting in 5 seconds...", err)
// // 		time.Sleep(5 * time.Second)
// // 	}
// // }

// func (s *Service) Start() error {
// 	// 1. DYNAMICALLY FIND THE BINARY
// 	// This fixes the issue between 'go run' (Dev) and 'go build' (Prod)
// 	frpcPath, err := findBinary("frpc")
// 	if err != nil {
// 		log.Printf("[TUNNEL] CRITICAL: Could not find 'frpc' binary. Ensure it is in the project root.")
// 		return err
// 	}
	
// 	log.Printf("[TUNNEL] Found binary at: %s", frpcPath)

// 	// 2. Determine where to save the config file
// 	// We use Current Working Directory (CWD) to ensure we have write permissions
// 	cwd, _ := os.Getwd()
// 	configPath := filepath.Join(cwd, "frpc.toml")

// 	data := TemplateData{
// 		ServerIP:   s.GlobalConfig.VPSIP,
// 		ServerPort: s.GlobalConfig.VPSPort,
// 		Token:      s.GlobalConfig.AuthToken,
// 		DeviceID:   s.GlobalConfig.DeviceID,
// 		LocalPort:  8080,
// 	}

// 	log.Printf("[TUNNEL] Configuring for Device: %s -> %s:%d", data.DeviceID, data.ServerIP, data.ServerPort)

// 	// 3. Create Config File
// 	file, err := os.Create(configPath)
// 	if err != nil {
// 		return fmt.Errorf("failed to create config file: %v", err)
// 	}
	
// 	func() {
// 		defer file.Close()
// 		tmpl, err := template.New("frpc").Parse(frpConfigTmpl)
// 		if err != nil {
// 			log.Printf("[TUNNEL] Template error: %v", err)
// 			return
// 		}
// 		if err := tmpl.Execute(file, data); err != nil {
// 			log.Printf("[TUNNEL] Config write error: %v", err)
// 		}
// 	}()

// 	// 4. Ensure Binary is Executable
// 	if err := os.Chmod(frpcPath, 0755); err != nil {
// 		log.Printf("[TUNNEL] Warning: Could not chmod binary: %v", err)
// 	}

// 	// 5. Run Loop
// 	for {
// 		log.Println("[TUNNEL] Starting FRP Client...")

// 		// Use Absolute Paths for everything
// 		cmd := exec.Command(frpcPath, "-c", configPath)
// 		cmd.Stdout = os.Stdout
// 		cmd.Stderr = os.Stderr

// 		err := cmd.Start()
// 		if err != nil {
// 			log.Printf("[TUNNEL] Failed to start: %v", err)
// 			time.Sleep(10 * time.Second)
// 			continue
// 		}

// 		err = cmd.Wait()
// 		log.Printf("[TUNNEL] Process exited: %v. Restarting in 5 seconds...", err)
// 		time.Sleep(5 * time.Second)
// 	}
// }

// // Helper function to find frpc in Dev and Prod environments
// func findBinary(name string) (string, error) {
// 	// Priority 1: Check Current Working Directory (Best for 'go run' or Docker working dir)
// 	cwd, err := os.Getwd()
// 	if err == nil {
// 		path := filepath.Join(cwd, name)
// 		if _, err := os.Stat(path); err == nil {
// 			return path, nil
// 		}
// 	}

// 	// Priority 2: Check relative to the Executable (Best for 'go build' / Systemd)
// 	ex, err := os.Executable()
// 	if err == nil {
// 		exPath := filepath.Join(filepath.Dir(ex), name)
// 		if _, err := os.Stat(exPath); err == nil {
// 			return exPath, nil
// 		}
// 	}

// 	// Priority 3: Check global PATH (If installed via package manager)
// 	path, err := exec.LookPath(name)
// 	if err == nil {
// 		return path, nil
// 	}

// 	return "", fmt.Errorf("binary %s not found in CWD or Executable dir", name)
// }


// //! check future implementation
// // package tunnel

// // import (
// // 	"fmt"
// // 	"log"
// // 	"os"
// // 	"os/exec"
// // 	"text/template" // Used text/template for config files (not html)
// // 	"time"

// // 	"github.com/strct-org/strct-agent/internal/config"
// // )

// // type Service struct {
// // 	GlobalConfig *config.Config
// // }

// // // TemplateData holds values to inject into frpc.toml
// // type TemplateData struct {
// // 	// Server Params
// // 	ServerIP   string
// // 	ServerPort int
// // 	Token      string
	
// // 	// Device Identity
// // 	DeviceID   string

// // 	// Local Ports (On the Raspberry Pi)
// // 	LocalWebPort  int // Your Cloud/UI (8080)
// // 	LocalVPNPort  int // WireGuard (51820)
// // 	LocalSSHPort  int // SSH (22)

// // 	// Remote Ports (On the VPS)
// // 	// These MUST be unique per device for TCP/UDP protocols!
// // 	RemoteVPNPort int 
// // 	RemoteSSHPort int
// // }

// // // -----------------------------------------------------------------------------
// // // FRP CONFIGURATION TEMPLATE
// // // -----------------------------------------------------------------------------
// // const frpConfigTmpl = `
// // # Common Server Settings
// // serverAddr = "{{.ServerIP}}"
// // serverPort = {{.ServerPort}}
// // auth.token = "{{.Token}}"

// // # -----------------------------------------------------------------------------
// // # 1. WEB DASHBOARD (Cloud + AdBlocker UI)
// // # -----------------------------------------------------------------------------
// // # Accessed via: https://{{.DeviceID}}.strct.org
// // # Caddy (VPS) -> FRP VHost Port -> Tunnel -> Pi:{{.LocalWebPort}}
// // [[proxies]]
// // name = "web_{{.DeviceID}}"
// // type = "http"
// // localPort = {{.LocalWebPort}}
// // subdomain = "{{.DeviceID}}"

// // # -----------------------------------------------------------------------------
// // # 2. VPN (WireGuard)
// // # -----------------------------------------------------------------------------
// // # Accessed via: {{.ServerIP}}:{{.RemoteVPNPort}}
// // # This tunnels UDP traffic for VPN connection.
// // [[proxies]]
// // name = "vpn_{{.DeviceID}}"
// // type = "udp"
// // localPort = {{.LocalVPNPort}}
// // remotePort = {{.RemoteVPNPort}}

// // # -----------------------------------------------------------------------------
// // # 3. SSH ACCESS (Remote Maintenance)
// // # -----------------------------------------------------------------------------
// // # Accessed via: ssh pi@{{.ServerIP}} -p {{.RemoteSSHPort}}
// // [[proxies]]
// // name = "ssh_{{.DeviceID}}"
// // type = "tcp"
// // localPort = {{.LocalSSHPort}}
// // remotePort = {{.RemoteSSHPort}}
// // `

// // func New(cfg *config.Config) *Service {
// // 	return &Service{
// // 		GlobalConfig: cfg,
// // 	}
// // }

// // func (s *Service) Start() error {
// // 	// 1. Prepare the Data
// // 	// In a real production scenario, RemoteVPNPort and RemoteSSHPort
// // 	// should come from s.GlobalConfig, assigned by your backend API 
// // 	// when the device registers.
	
// // 	// Example fallback if config is missing specific ports (YOU MUST FIX THIS FOR PROD)
// // 	vpnPort := s.GlobalConfig.AssignedVPNPort
// // 	if vpnPort == 0 {
// // 		vpnPort = 6000 // Default/Fallback (Dangerous in prod: collisions will occur)
// // 	}

// // 	sshPort := s.GlobalConfig.AssignedSSHPort
// // 	if sshPort == 0 {
// // 		sshPort = 2200 // Default/Fallback
// // 	}

// // 	data := TemplateData{
// // 		ServerIP:      s.GlobalConfig.VPSIP,
// // 		ServerPort:    s.GlobalConfig.VPSPort,
// // 		Token:         s.GlobalConfig.AuthToken,
// // 		DeviceID:      s.GlobalConfig.DeviceID,
		
// // 		// Internal Services
// // 		LocalWebPort:  8080,  // Matches your 'cloud' package port
// // 		LocalVPNPort:  51820, // Standard WireGuard port
// // 		LocalSSHPort:  22,    // Standard SSH port

// // 		// External Access
// // 		RemoteVPNPort: vpnPort,
// // 		RemoteSSHPort: sshPort,
// // 	}

// // 	log.Printf("[TUNNEL] Generating config for Device: %s", data.DeviceID)
// // 	log.Printf("[TUNNEL] Proxy Routing: Web(Subdomain) | VPN(:%d) | SSH(:%d)", data.RemoteVPNPort, data.RemoteSSHPort)

// // 	// 2. Write the Config File
// // 	if err := s.writeConfig(data); err != nil {
// // 		return err
// // 	}

// // 	// 3. Run the Binary Loop
// // 	go s.runProcess()

// // 	return nil
// // }

// // func (s *Service) writeConfig(data TemplateData) error {
// // 	file, err := os.Create("frpc.toml")
// // 	if err != nil {
// // 		return fmt.Errorf("failed to create frpc.toml: %v", err)
// // 	}
// // 	defer file.Close()

// // 	tmpl, err := template.New("frpc").Parse(frpConfigTmpl)
// // 	if err != nil {
// // 		return fmt.Errorf("failed to parse template: %v", err)
// // 	}

// // 	if err := tmpl.Execute(file, data); err != nil {
// // 		return fmt.Errorf("failed to write config content: %v", err)
// // 	}
	
// // 	return nil
// // }

// // func (s *Service) runProcess() {
// // 	for {
// // 		log.Println("[TUNNEL] Starting FRP Client...")

// // 		// Ensure ./frpc is executable
// // 		os.Chmod("./frpc", 0755)

// // 		cmd := exec.Command("./frpc", "-c", "frpc.toml")
// // 		cmd.Stdout = os.Stdout
// // 		cmd.Stderr = os.Stderr

// // 		if err := cmd.Start(); err != nil {
// // 			log.Printf("[TUNNEL] Failed to start binary: %v. Is ./frpc present?", err)
// // 			time.Sleep(15 * time.Second)
// // 			continue
// // 		}

// // 		// Wait for the process to exit (it shouldn't, unless network drops)
// // 		err := cmd.Wait()
// // 		log.Printf("[TUNNEL] Connection lost: %v. Restarting in 5s...", err)
// // 		time.Sleep(5 * time.Second)
// // 	}
// // }