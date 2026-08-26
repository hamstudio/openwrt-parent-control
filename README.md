# ParentControl Guard for OpenWrt

[English](README.md) | [简体中文](README_zh.md)

A modern, high-performance, and fine-grained Parental Control & Application Security Management System for OpenWrt, powered by Go and the `kmod-oaf` Deep Packet Inspection (DPI) kernel module.

---

## ✨ Features

- **📱 Modern Responsive Web Dashboard**: Mobile-first, single-binary embedded Web UI with zero external runtime dependencies. Instant lock toggles, bonus time rewards, and quota sliders.
- **🎮 Deep L7 Application Blocking (DPI)**: Powered by kernel-level packet inspection to accurately identify and restrict hundreds of popular apps across categories (e.g., Honor of Kings, Genshin Impact, Steam, TikTok, YouTube, Discord).
- **⏱️ Fine-Grained Time & Quota Management**:
  - Flexible schedule rules by day and time window (with overnight span support).
  - Daily active-traffic-driven time quota (token bucket).
  - Instant actions: One-click Internet Lock & temporary Bonus Time allowance (e.g., +30 mins).
- **🛡️ Content Filtering & Anti-Bypass System**:
  - Enforced SafeSearch on Google, Bing, Baidu, and YouTube.
  - Forced local port 53 DNS redirection and blocked public DoH/DoT servers to prevent DNS evasion.
  - New device quarantine mode to defend against MAC address randomization bypasses.
- **⚙️ Native LuCI Integration**: Seamless integration into the OpenWrt administration interface via `luci-app-parentcontrol`.

---

## 📁 Repository Structure

```text
├── cmd/parentcontrold/       # Main daemon entrypoint
├── internal/
│   ├── api/                  # RESTful HTTP API and static file server
│   ├── config/               # Configuration management and persistence
│   ├── device/               # LAN device discovery (ARP/DHCP) & traffic metrics
│   ├── dpi/                  # kmod-oaf kernel driver & signature parser
│   ├── firewall/             # iptables rule management & anti-bypass filters
│   ├── models/               # Core data structures and models
│   ├── quota/                # Time tracking and quota policy engine
│   └── safedns/              # dnsmasq configuration & SafeSearch enforcement
├── web/                      # Responsive Web UI source code (embedded via Go embed)
├── rootfs/                   # OpenWrt filesystem files (init.d service, LuCI menu & ACL)
├── scripts/                  # Cross-compilation and automated deployment scripts
├── docs/                     # Detailed architectural, API, and deployment documentation
└── Makefile
```

---

## 📚 Documentation

- [Architecture & Design](docs/ARCHITECTURE.md) ([中文版](docs/ARCHITECTURE_zh.md))
- [RESTful API Reference](docs/API.md) ([中文版](docs/API_zh.md))
- [Deployment & Operations Guide](docs/DEPLOYMENT.md) ([中文版](docs/DEPLOYMENT_zh.md))

---

## 🚀 Quick Start & Deployment

### Automated One-Click Deployment
Ensure your local machine has Go installed, then execute:
```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

Once deployment completes, open your browser and navigate to:
```text
http://<Router-IP>:8088
```

---

## 📄 License

This project is licensed under the MIT License.
