#!/bin/bash
set -e

VERSION="v1.0.1" 
BINARY_NAME="strct-agent-arm64"
REPO="strct-org/strct"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"

echo "--- Installing Strct Cloud Agent ---"

echo "Installing dependencies..."
apt-get update -qq
apt-get install -y docker.io network-manager curl

echo "Creating directories..."
mkdir -p /mnt/data
mkdir -p /etc/strct

echo "Downloading Agent ($VERSION)..."
curl -L -o /usr/local/bin/cloud-agent "$DOWNLOAD_URL"
chmod +x /usr/local/bin/cloud-agent

echo "Configuring Systemd..."
cat <<EOF > /etc/systemd/system/cloud-agent.service
[Unit]
Description=Strct Agent
After=network-online.target docker.service
Wants=network-online.target docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/etc/strct
ExecStart=/usr/local/bin/cloud-agent
Restart=always
RestartSec=5s
Environment="VPS_IP=157.90.167.157"
Environment="VPS_PORT=7000"
Environment="AUTH_TOKEN=Struct33_Secret_Key_99"
Environment="DOMAIN=strct.org"

[Install]
WantedBy=multi-user.target
EOF

echo "Starting Agent..."
systemctl daemon-reload
systemctl enable cloud-agent
systemctl restart cloud-agent

echo "SUCCESS! Agent is running."
echo "Check status with: systemctl status cloud-agent"
echo "View logs with:  journalctl -u cloud-agent -f"