# OpenWrt 细粒度家长控制系统 (ParentControl Guard)

[English](README.md) | [简体中文](README_zh.md)

基于 Go 语言与 `kmod-oaf` 内核深度包检测 (DPI) 引擎开发的 OpenWrt 细粒度家长控制与应用安全管控系统。

## ✨ 核心特性

- **📱 现代独立 Web 控制台**：零外部依赖，移动端优先，支持手机浏览器便捷管理、大卡片一键断网、加时奖励及配额滑块。
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

## 📁 目录结构

```text
├── cmd/parentcontrold/       # 核心守护进程入口
├── internal/
│   ├── api/                  # RESTful HTTP API 与 Web 路由
│   ├── config/               # 配置存储与持久化
│   ├── device/               # 局域网设备探测 (ARP/DHCP) 与流量采集
│   ├── dpi/                  # kmod-oaf 内核驱动与特征库解析
│   ├── firewall/             # iptables 动态规则链与防绕过
│   ├── models/               # 数据模型定义
│   ├── quota/                # 活跃时长统计与配额调度引擎
│   └── safedns/              # dnsmasq 协同与 SafeSearch 拦截
├── web/                      # 现代化独立 Web 控制台前端源码 (Go embed 内嵌)
├── rootfs/                   # OpenWrt 部署文件 (init.d 服务、LuCI 菜单与 ACL)
├── scripts/                  # 编译与一键部署脚本
└── Makefile
```

## 📚 文档目录

- [系统架构与原理](docs/ARCHITECTURE_zh.md)
- [RESTful API 规范](docs/API_zh.md)
- [部署与运维指南](docs/DEPLOYMENT_zh.md)

## 🚀 快速上手

```bash
# 赋予部署脚本权限并部署至目标路由器
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

部署完成后，即可通过浏览器访问：`http://<路由器IP>:8088` 进入管理控制台。
