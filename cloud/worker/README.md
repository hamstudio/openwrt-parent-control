# Cloudflare Worker Relay for ParentControl Guard

[English](README.md) | [简体中文](README_zh.md)

This serverless Cloudflare Worker provides an out-of-home remote synchronization relay for ParentControl Guard. It allows parents to manage household internet policies over cellular networks (4G/5G) without requiring a public IP, dynamic DNS (DDNS), or port forwarding on their home router.

---

## 🏗️ How It Works

1. **Router Long-Poll (`CloudSyncer`)**: The router's Go daemon sends periodic outbound HTTPS requests to the Worker to push status updates and pull pending commands.
2. **State & Command Storage (Cloudflare KV)**:
   - `status:<device_id>`: Stores real-time router status and active member states.
   - `commands:<device_id>`: Queues pending control actions (Lock, Unlock, Bonus Time, Rule changes).
3. **Mobile Client Access**: Native iOS and Android apps can interact directly with the Worker endpoint using a shared secret and PIN.

---

## 🚀 Quick Deployment with Wrangler

### 1. Install Dependencies
```bash
cd cloud/worker
npm install
```

### 2. Create KV Namespace
```bash
# Create KV namespace for production
npx wrangler kv:namespace create PARENT_CONTROL_KV
```

Copy the generated `id` and paste it into `wrangler.toml`:
```toml
[[kv_namespaces]]
binding = "PARENT_CONTROL_KV"
id = "YOUR_KV_NAMESPACE_ID_HERE"
```

### 3. Deploy to Cloudflare
```bash
npx wrangler deploy
```

Once deployed, you will receive a public worker URL such as `https://parentcontrol-worker.<your-subdomain>.workers.dev`.

---

## ⚙️ Router Configuration

1. In the router's Web Console, go to **Global Settings** -> **Cloudflare Worker Remote Sync**.
2. Enable Cloud Sync, enter your Worker URL and optionally set a shared Secret.
3. Click **Save & Apply Settings**.
