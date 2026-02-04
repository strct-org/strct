package setup

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/strct-org/strct-agent/internal/platform/wifi"
)

type Credentials struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

func StartCaptivePortal(wifiMgr wifi.Provider, done chan<- bool, devMode bool) {
	mux := http.NewServeMux()

	mux.HandleFunc("/scan", func(w http.ResponseWriter, r *http.Request) {
		networks, err := wifiMgr.Scan()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(networks)
	})

	mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}
		log.Printf("[SETUP] Received credentials for %s", creds.SSID)
		
		err := wifiMgr.Connect(creds.SSID, creds.Password)
		if err != nil {
			http.Error(w, "Failed to connect: "+err.Error(), 500)
			return
		}
		w.Write([]byte("Connected! Rebooting..."))
		done <- true
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlPage)
	})

	port := ":80"
	if devMode {
		port = ":8082"
	}

	log.Printf("[SETUP] Web Server listening on %s", port)
	
	if !devMode {
		dnsServer := StartDNSServer("10.42.0.1") 
		defer dnsServer.Shutdown() 
	}

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Printf("[SETUP] HTTP Server Error: %v", err)
	}
}

const htmlPage = `
<!DOCTYPE html>
<html>
<body>
<h2>StructIO Setup</h2>
<button onclick="scan()">Scan Networks</button>
<ul id="list"></ul>
<div id="form" style="display:none">
    <input id="ssid" placeholder="SSID"><br>
    <input id="pass" placeholder="Password"><br>
    <button onclick="connect()">Connect</button>
</div>
<script>
async function scan() {
    let res = await fetch('/scan');
    let nets = await res.json();
    let list = document.getElementById('list');
    list.innerHTML = '';
    nets.forEach(n => {
        let li = document.createElement('li');
        li.innerText = n.SSID + ' (' + n.Signal + '%)';
        li.onclick = () => {
            document.getElementById('ssid').value = n.SSID;
            document.getElementById('form').style.display = 'block';
        };
        list.appendChild(li);
    });
}
async function connect() {
    let ssid = document.getElementById('ssid').value;
    let pass = document.getElementById('pass').value;
    await fetch('/connect', {
        method: 'POST',
        body: JSON.stringify({ssid, password: pass})
    });
    alert('Device connecting... check LED status.');
}
</script>
</body>
</html>
`