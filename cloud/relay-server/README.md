# ParentControl Go Cloud Relay Server (国内 VPS 独立中继后端)

一套基于 Go 语言原生开发的高性能、轻量级云端双向中继服务，内置 **In-Memory PubSub MQ 消息总线** 与 **WebSocket 毫秒级双向即时推流**，专为部署在国内阿里云/腾讯云/华为云/私有 VPS 设计，彻底摆脱海外 Cloudflare Worker 的网络波动与限制。

---

## ✨ 核心特性

- **🚀 零外部依赖 & 极致轻量**：无须安装 Redis/RabbitMQ，单二进制文件开箱即用，内存常驻仅 ~10MB。
- **⚡ 双向 WebSocket 实时长连接**：
  - 路由器端（出站长连接，无需公网 IP / 端口映射）；
  - 移动端 App（毫秒级状态同步与指令直达）。
- **📦 离线指令缓冲队列**：路由器短暂断线期间，App 下发的断网/加时指令自动进入 MQ 待收队列，上线秒级执行。
- **🛡️ 多租户隔离与安全认证**：基于 `X-Router-Secret` 自定义密钥隔离不同家庭与路由器设备。

---

## 🚀 快速部署 (VPS 推荐)

### 方法 1: 使用 Docker Compose (推荐，10 秒启动)

```bash
# 1. 克隆代码或上传本目录到 VPS
git clone https://github.com/hamguy/parent-control.git
cd parent-control/cloud/relay-server

# 2. 启动服务
docker compose up -d

# 3. 检查服务健康状态
curl http://localhost:9000/health
```

### 方法 2: 直接编译为独立二进制文件运行

```bash
# 编译二进制文件
CGO_ENABLED=0 go build -ldflags="-s -w" -o relay-server ./cmd/relay-server

# 启动 (自定义端口与密钥)
PORT=9000 RELAY_SECRET="my_secure_secret_888" ./relay-server
```

---

## ⚙️ 路由器与手机 App 配置

1. 打开路由器 Web 控制台 -> **【全局安全配置】**；
2. 开启 **“云端远程管理与公网同步”**；
3. 将 **Worker / Relay API 地址** 填入您的 VPS 地址：
   - 格式 1 (WebSocket): `ws://<vps-ip>:9000/ws/router`
   - 格式 2 (HTTP/HTTPS): `http://<vps-ip>:9000`
4. 填入 **通信共享密钥 (Secret)**；
5. 点击保存后，路由器将自动建立出站长连接并开始实时双向推流。
