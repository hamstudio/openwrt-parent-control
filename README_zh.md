# OpenWrt 细粒度家长控制系统 (ParentControl Guard)

[English](README.md) | [简体中文](README_zh.md)

基于 Go 语言与 `kmod-oaf` 内核深度包检测 (DPI) 引擎开发的 OpenWrt 细粒度家长控制与应用安全管控系统。提供完整的 iOS (SwiftUI) 与 Android 跨平台原生直连控制客户端（共享纯 Swift 核心逻辑）。

---

## ✨ 核心特性

- **📱 现代独立 Web 控制台**：零外部依赖，移动端优先，支持手机浏览器便捷管理、大卡片一键断网、加时奖励及配额滑块。
- **🍎 原生 iOS (SwiftUI) 与 Android 直连控制端**：
  - **通用 Swift 业务核心 (`ParentControlCore`)**：跨平台数据模型、异步网络引擎、路由器动态自动探测均由纯 Swift 编写。
  - **iOS App**：纯 SwiftUI 构建，原生流式排版、触觉震动反馈与响应式交互。
  - **Android 复用**：Swift Core 导出为 C-FFI / JNI 动态库 (`libParentControlBridge.so`)，Kotlin 协程无缝消费。
- **🎮 深度应用识别与封禁 (L7 DPI)**：基于内核级 DPI 模块与特征库，精准识别和封禁游戏（王者荣耀、和平精英、原神、Steam）、短视频（抖音、快手、小红书）、社交、直播等数十个大类、数百款具体 App。
- **⏱️ 细粒度时间与配额控制**：
  - 支持按周天/时间区间自定义禁网计划（支持跨夜）。
  - 基于真实流量活跃度的每日时长限额（Quota）。
  - 支持一键断网 (Instant Lock) 与临时加时奖励 (Bonus Time)。
- **🛡️ 域名安全与防绕过体系**：
  - 强制锁定 Google、Bing、Baidu、YouTube 青少年安全搜索 (SafeSearch)。
  - 53 端口强制重定向本地，封锁外部公共 DoH/DoT，杜绝私自修改 DNS 绕过。
  - 新设备隔离模式，防御 MAC 随机化逃避监管。
- **⚙️ LuCI 原生支持**：提供 `luci-app-parentcontrol` 插件，无缝集成至 OpenWrt 系统菜单。

---

## 📁 目录结构

```text
├── cmd/parentcontrold/       # 核心守护进程入口 (Go)
├── internal/                 # Go 后端核心引擎 (DPI, Firewall, Quota, API)
├── web/                      # 现代化独立 Web 控制台前端源码 (Go embed 内嵌)
├── rootfs/                   # OpenWrt 部署文件 (init.d 服务、LuCI 菜单与 ACL)
├── client/                   # 原生移动端工程
│   ├── ParentControlCore/    # 通用 Swift 跨平台包 (Models, Client, State, JNI Bridge)
│   ├── iOS/                  # iOS SwiftUI 原生应用 (ParentControlApp)
│   └── Android/              # Android JNI 桥接与 Kotlin 协程封装
├── docs/                     # 架构、API、双语及 Android 跨平台复用文档
├── scripts/                  # 编译与一键部署脚本
└── Makefile
```

---

## 📚 文档目录

- [系统架构与原理](docs/ARCHITECTURE_zh.md) ([English](docs/ARCHITECTURE.md))
- [RESTful API 规范](docs/API_zh.md) ([English](docs/API.md))
- [部署与运维指南](docs/DEPLOYMENT_zh.md) ([English](docs/DEPLOYMENT.md))
- [常见问题解答 (FAQ)](docs/FAQ_zh.md) ([English](docs/FAQ.md))
- [Android 端复用 Swift 核心逻辑指南](docs/ANDROID_SWIFT_INTEROP_zh.md) ([English](docs/ANDROID_SWIFT_INTEROP.md))

---

## 🚀 快速上手

### 路由器端一键部署
```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

### 原生客户端编译与测试
- **运行 Swift 核心跨平台单元测试**：`cd client/ParentControlCore && swift test`
- **构建 iOS 原生应用**：`cd client/iOS && swift build`
