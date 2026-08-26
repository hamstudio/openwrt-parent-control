# 系统架构与实现原理设计 (Architecture Design)

[English](ARCHITECTURE.md) | [简体中文](ARCHITECTURE_zh.md)

本文档详细描述 OpenWrt 细粒度家长控制系统 (ParentControl Guard) 的整体技术架构、数据流向及底层网络控制原理。

---

## 1. 整体技术分层

系统由 **用户交互层 (Web & LuCI)**、**核心守护进程 (Go Daemon)**、**系统配置层 (Config Store)** 以及 **底层网络执行层 (iptables & kmod-oaf & dnsmasq)** 四层协同构成：

```mermaid
flowchart TB
    subgraph UI_Layer [用户交互层]
        WebUI[📱 现代响应式 Web 控制台 (端口 8088)]
        LuCI[⚙️ LuCI 原生菜单插件]
    end

    subgraph Daemon_Layer [Go 核心守护进程 parentcontrold]
        APIServer[RESTful API 服务]
        DeviceTracker[设备自动发现与流量采集]
        QuotaEngine[活跃时长统计与配额引擎]
        PolicyEngine[策略评估与防火墙/DPI下发]
    end

    subgraph Config_Layer [配置持久化]
        ConfigFile[/etc/parentcontrol/config.json]
    end

    subgraph Kernel_Network [底层网络栈与执行层]
        OAFEngine[kmod-oaf 内核 DPI 模块]
        DevAppfilter[/dev/appfilter 字符设备]
        IPTablesForward[iptables PARENT_CONTROL_FWD]
        IPTablesNat[iptables PARENT_CONTROL_NAT_PRE]
        DnsmasqConf[/tmp/dnsmasq.d/parentcontrol.conf]
    end

    WebUI <--> APIServer
    LuCI <--> WebUI
    APIServer <--> PolicyEngine
    PolicyEngine <--> ConfigFile
    PolicyEngine <--> QuotaEngine
    DeviceTracker --> QuotaEngine

    PolicyEngine --> IPTablesForward
    PolicyEngine --> IPTablesNat
    PolicyEngine --> DevAppfilter
    DevAppfilter --> OAFEngine
    PolicyEngine --> DnsmasqConf
```

---

## 2. 核心网络控制机制

### 2.1 流量拦截流水线 (Forward Chain Pipeline)
当数据包从局域网终端进入路由器转发链 (`FORWARD`) 时：
1. **优先进入自定义链**：`PARENT_CONTROL_FWD` 挂载于 `FORWARD` 链首部。
2. **防绕过审查**：
   - 阻断 DoT (TCP/UDP 853 端口)。
   - 阻断公网知名 DoH IP（如 `1.1.1.1`, `8.8.8.8` 等）。
3. **受管状态判定**：
   - 若设备被**一键断网 (Locked)** -> `DROP`。
   - 若设备**每日配额耗尽**且未处于**加时状态 (Bonus Time)** -> `DROP`。
   - 若设备处于**禁网时间段**内且未处于**加时状态** -> `DROP`。

### 2.2 深度包检测 (L7 DPI) 交互机制
1. 内核模块 `kmod-oaf` 拦截经过网络协议栈的数据包，进行特征码匹配（SNI、协议头、目的 IP/端口）。
2. `parentcontrold` 启动时读取 `/etc/appfilter/feature_cn.cfg`，解析应用 ID 与分类。
3. 动态配置通过向 `/dev/appfilter` 写入 JSON 指令实现：
   - `op: 3` -> 清空现有规则。
   - `op: 1` -> 下发封禁的 App ID 列表 (`{"apps": [2001, 2002, ...]}`)。
   - `op: 4` -> 下发受管设备的 MAC 列表 (`{"mac_list": ["AA:BB:CC:..."]}`)。

### 2.3 SafeSearch 与 DNS 强制劫持
1. **53 端口强制重定向**：
   - 在 `nat` 表的 `PREROUTING` 链挂载 `PARENT_CONTROL_NAT_PRE`，将所有目标为外部 53 端口的请求重定向到本地 53 端口 (`REDIRECT --to-ports 53`)。
2. **搜索引擎安全搜索 (SafeSearch)**：
   - 生成 `/tmp/dnsmasq.d/parentcontrol.conf`：
     - Google: 重写至 `forcesafesearch.google.com` (216.239.38.120)。
     - Bing: 重写至 `strict.bing.com` (204.79.197.220)。
     - YouTube: 重写至 `restrict.youtube.com` (216.239.38.119)。
   - 向 `dnsmasq` 发送 `SIGHUP` 信号实现平滑无缝热重载。
