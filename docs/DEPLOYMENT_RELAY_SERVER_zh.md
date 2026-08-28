# 国内 VPS 自建 Go Relay Server (中继服务器) 部署指南

对于国内网络环境下无法稳定访问 Cloudflare Worker 的场景，ParentControl Guard 提供了纯 Go 原生编写的高性能、轻量级云端中继服务 (**Go Relay Server + MQ Socket**)。

---

## 🏗️ 架构优势

1. **⚡ 双向 WebSocket 即时推流**：
   - 路由器主动出站与 VPS 保持单条 TCP/WSS 长连接（**路由器无需公网 IP，无需端口映射**）；
   - 手机 App 下发一键断网、加时等指令时，毫秒级即时推达路由器。
2. **🚀 极简免运维**：
   - 内置轻量级 In-Memory PubSub 消息总线，零外部中间件（无需 Redis/RabbitMQ/Kafka）；
   - 单二进制文件，内存占用仅约 10MB。
3. **📦 离线指令缓冲**：
   - 路由器网络离线时，家长 App 提交的规则修改和指令自动进入 MQ 缓冲队列，路由器重新连上时秒级自动执行。

---

## 🛠️ VPS 部署步骤

### 方法 A: Docker Compose 极速部署 (推荐，10 秒启动)

在您的国内云服务器（阿里云、腾讯云、华为云、UCloud 等）上执行：

```bash
# 1. 下载或克隆仓库
git clone https://github.com/hamguy/parent-control.git
cd parent-control/cloud/relay-server

# 2. 修改 docker-compose.yml 中的 RELAY_SECRET 为您自定义的强密码
# 例如：RELAY_SECRET=my_custom_secret_key_888

# 3. 启动容器
docker compose up -d

# 4. 验证服务运行
curl http://localhost:9000/health
# 返回: {"service":"ParentControl-RelayServer","status":"ok","version":"1.0.0"}
```

---

### 方法 B: 编译为 Linux 独立二进制文件并使用 Systemd 守护

```bash
# 1. 编译
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /usr/local/bin/parentcontrol-relay ./cmd/relay-server

# 2. 创建 systemd 服务文件 /etc/systemd/system/parentcontrol-relay.service
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

# 3. 启动并设置开机自启
systemctl daemon-reload
systemctl enable --now parentcontrol-relay
```

---

## 🔒 推荐：配置 Nginx 反向代理与 HTTPS / WSS 证书

为了获得最佳安全性，建议在 VPS 上通过 Nginx 配置域名与免费 SSL 证书（Let's Encrypt）：

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

## ⚙️ 路由器与手机 App 接入配置

1. 登录路由器管理后台 -> **【家长控制】** -> **【全局安全配置】**；
2. 勾选 **“云端远程管理与公网同步”**；
3. **Relay Server 地址** 填写：
   - 域名模式 (推荐): `wss://relay.yourdomain.com/ws/router`
   - IP 模式: `ws://<vps-ip>:9000/ws/router`
4. **通信共享密钥 (Secret)** 填写 VPS 上配置的 `RELAY_SECRET`；
5. 点击 **【保存并应用设置】**，路由器日志将立即提示 `Connected to Cloud Relay Server successfully via WebSocket!`。
