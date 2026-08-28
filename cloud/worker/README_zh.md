# ParentControl Guard - Cloudflare Worker 远程同步服务

[English](README.md) | [简体中文](README_zh.md)

本模块是基于 Cloudflare Workers Serverless 架构构建的公网远程同步中继服务。它让家长能够在 4G/5G 移动网络下，随时随地远程控制家中的路由器规则，**无需公网 IP、无需动态域名解析 (DDNS)，也无需在路由器上做任何端口映射**。

---

## 🏗️ 运作原理

1. **路由器主动长轮询 (`CloudSyncer`)**：路由器的 Go 守护进程周期性向 Worker 发起出站 HTTPS 请求，同步路由器运行状态并拉取待执行指令。
2. **状态与指令暂存 (Cloudflare KV)**：
   - `status:<device_id>`：保存路由器实时状态与成员在线/锁定情况。
   - `commands:<device_id>`：队列化暂存待执行的操作指令（一键断网、恢复、临时加时、规则调整等）。
3. **手机 App 跨网通信**：iOS 与 Android 客户端通过 Worker 提供的 HTTP 接口下发控制指令，安全便捷。

---

## 🚀 使用 Wrangler 快速部署

### 1. 安装依赖
```bash
cd cloud/worker
npm install
```

### 2. 创建 Cloudflare KV 命名空间
```bash
# 创建生产环境 KV 命名空间
npx wrangler kv:namespace create PARENT_CONTROL_KV
```

将控制台输出的 `id` 填入 `wrangler.toml` 文件中：
```toml
[[kv_namespaces]]
binding = "PARENT_CONTROL_KV"
id = "填入你的_KV_NAMESPACE_ID"
```

### 3. 一键部署到 Cloudflare
```bash
npx wrangler deploy
```

部署完成后，控制台将输出专属访问地址，例如：`https://parentcontrol-worker.<你的子域>.workers.dev`。

---

## ⚙️ 路由器端接入配置

1. 打开路由器 Web 控制台，进入 **全局安全配置** -> **Cloudflare Worker 公网远程同步**。
2. 开启同步开关，填入部署好的 Worker API 地址及通信共享密钥（可选）。
3. 点击 **保存并应用设置** 即可完成接入。
