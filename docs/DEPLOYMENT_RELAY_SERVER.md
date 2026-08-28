# Standalone Go Cloud Relay Server Deployment Guide (Domestic VPS / Self-Hosted)

For scenarios where Cloudflare Workers are unstable or inaccessible (e.g. within domestic Mainland China networks), ParentControl Guard provides a high-performance, lightweight **Go Relay Server + MQ Socket** backend.

---

## 🏗️ Architecture Highlights

1. **⚡ Real-time Bidirectional WebSocket Streams**:
   - The router initiates a single outbound TCP/WSS connection (**no public IP, DDNS, or port forwarding required**);
   - Commands (instant lock, bonus time) dispatched from mobile apps reach the router in milliseconds.
2. **🚀 Zero-Ops In-Memory PubSub MQ**:
   - Built-in lightweight message queue without external Redis/RabbitMQ dependencies;
   - Single static binary consuming only ~10MB of RAM.
3. **📦 Offline Command Buffering**:
   - If the router is temporarily disconnected, parent commands are queued and instantly executed upon reconnection.

---

## 🛠️ VPS Deployment Options

### Option A: Docker Compose (Recommended, 10-Second Setup)

```bash
# 1. Clone repository
git clone https://github.com/hamguy/parent-control.git
cd parent-control/cloud/relay-server

# 2. Configure your secret key in docker-compose.yml
# e.g., RELAY_SECRET=my_custom_secret_key_888

# 3. Launch container
docker compose up -d

# 4. Verify health
curl http://localhost:9000/health
# Response: {"service":"ParentControl-RelayServer","status":"ok","version":"1.0.0"}
```

---

### Option B: Standalone Binary with Systemd

```bash
# 1. Build binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /usr/local/bin/parentcontrol-relay ./cmd/relay-server

# 2. Create systemd unit /etc/systemd/system/parentcontrol-relay.service
cat <<EOF > /etc/systemd/system/parentcontrol-relay.service
[Unit]
Description=ParentControl Cloud Relay Server
After=network.target

[Service]
Type=simple
Environment="PORT=9000"
Environment="RELAY_SECRET=your_custom_secret_here"
ExecStart=/usr/local/bin/parentcontrol-relay
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 3. Enable and start
systemctl daemon-reload
systemctl enable --now parentcontrol-relay
```

---

## 🔒 Recommended: Nginx Reverse Proxy with TLS/WSS

```nginx
server {
    listen 443 ssl http2;
    server_name relay.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/relay.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/relay.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:9000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

---

## ⚙️ Router & Mobile App Configuration

1. In the router dashboard, go to **Global Security Settings**;
2. Enable **Cloud Remote Management & Sync**;
3. Set **Relay Server URL**:
   - TLS Domain: `wss://relay.yourdomain.com/ws/router`
   - Direct IP: `ws://<vps-ip>:9000/ws/router`
4. Set **Shared Secret Key** to match your server's `RELAY_SECRET`;
5. Save settings to establish the live real-time WebSocket connection.
