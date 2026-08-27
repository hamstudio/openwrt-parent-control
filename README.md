# ParentControl Guard for OpenWrt

[English](README.md) | [简体中文](README_zh.md)

A modern, high-performance, and fine-grained Parental Control & Application Security Management System for OpenWrt, powered by Go and the `kmod-oaf` Deep Packet Inspection (DPI) kernel module. Includes native iOS (SwiftUI) and Android clients sharing a pure Swift core library.

---

## ✨ Features

- **📱 Modern Responsive Web Dashboard**: Mobile-first, single-binary embedded Web UI with zero external runtime dependencies. Instant lock toggles, bonus time rewards, and quota sliders.
- **🍎 Native iOS (SwiftUI) & Android Direct Client**:
  - **Shared Swift Core (`ParentControlCore`)**: Cross-platform business logic, async network engine, dynamic router discovery, and models written in Swift.
  - **iOS App**: Pure SwiftUI app with native animations, haptic feedback, and responsive layout.
  - **Android Interop**: Swift Core exported via C-FFI / JNI (`libParentControlBridge.so`) for Kotlin/Compose reuse.
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
├── cmd/parentcontrold/       # Main daemon entrypoint (Go)
├── internal/                 # Go backend core engine (DPI, Firewall, Quota, API)
├── web/                      # Embedded responsive Web UI source code
├── rootfs/                   # OpenWrt filesystem files (init.d service, LuCI menu & ACL)
├── client/                   # Native mobile clients
│   ├── ParentControlCore/    # Shared cross-platform Swift Package (Models, Client, State, JNI Bridge)
│   ├── iOS/                  # Native iOS SwiftUI App (ParentControlApp)
│   └── Android/              # Android JNI bridge & Kotlin coroutines repository
├── docs/                     # Architectural, API, and Android Swift interop documentation
├── scripts/                  # Automated compilation and deployment scripts
└── Makefile
```

---

## 📚 Documentation

- [Architecture & Design](docs/ARCHITECTURE.md) ([中文版](docs/ARCHITECTURE_zh.md))
- [RESTful API Reference](docs/API.md) ([中文版](docs/API_zh.md))
- [Deployment & Operations Guide](docs/DEPLOYMENT.md) ([中文版](docs/DEPLOYMENT_zh.md))
- [Android Swift Core Interop Guide](docs/ANDROID_SWIFT_INTEROP.md) ([中文版](docs/ANDROID_SWIFT_INTEROP_zh.md))

---

## 🚀 Quick Start & Deployment

### Automated One-Click Router Deployment
```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

### Native Mobile Clients
- **Shared Swift Core Tests**: `cd client/ParentControlCore && swift test`
- **Build iOS Native App**: `cd client/iOS && swift build`

---

## 📄 License

This project is licensed under the MIT License.
