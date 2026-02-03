package tunnel

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/strct-org/strct-agent/internal/config"
)

type Service struct {
	GlobalConfig *config.Config
}

type TemplateData struct {
	ServerIP   string
	ServerPort int
	Token      string
	DeviceID   string
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
	data := TemplateData{
		ServerIP:   s.GlobalConfig.VPSIP,
		ServerPort: s.GlobalConfig.VPSPort,
		Token:      s.GlobalConfig.AuthToken,
		DeviceID:   s.GlobalConfig.DeviceID,
		LocalPort:  8080, 
	}

	log.Printf("[TUNNEL] Configuring for Device: %s -> %s:%d", data.DeviceID, data.ServerIP, data.ServerPort)

	file, err := os.Create("frpc.toml")
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	tmpl, err := template.New("frpc").Parse(frpConfigTmpl)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	for {
		log.Println("[TUNNEL] Starting FRP Client...")
		
		cmd := exec.Command("./frpc", "-c", "frpc.toml")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Start()
		if err != nil {
			log.Printf("[TUNNEL] Failed to start binary: %v. Is ./frpc inside the folder?", err)
			time.Sleep(10 * time.Second)
			continue
		}

		err = cmd.Wait()
		log.Printf("[TUNNEL] Process exited: %v. Restarting in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}
}