# Architecture & System Design

[English](ARCHITECTURE.md) | [简体中文](ARCHITECTURE_zh.md)

This document provides a comprehensive overview of the architecture, packet processing pipeline, and internal mechanisms of ParentControl Guard for OpenWrt.

---

## 1. High-Level Architecture

The system consists of four distinct layers: **User Interface**, **Core Go Daemon**, **Configuration Store**, and the **Kernel/Network Enforcement Layer**.

```mermaid
flowchart TB
    subgraph UI_Layer [User Interface Layer]
        WebUI[📱 Modern Responsive Web Dashboard (Port 8088)]
        LuCI[⚙️ Native LuCI Menu Plugin]
    end

    subgraph Daemon_Layer [Core Go Daemon - parentcontrold]
        APIServer[RESTful API Engine]
        DeviceTracker[Device Discovery & Traffic Collector]
        QuotaEngine[Active Time Tracker & Token Bucket Engine]
        PolicyEngine[Rule Evaluator & Netfilter Dispatcher]
    end

    subgraph Config_Layer [Configuration & Persistence]
        ConfigFile[/etc/parentcontrol/config.json]
    end

    subgraph Kernel_Network [Kernel & Network Enforcement Layer]
        OAFEngine[kmod-oaf Kernel DPI Module]
        DevAppfilter[/dev/appfilter Character Device]
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

## 2. Core Network Enforcement Mechanisms

### 2.1 Packet Filtering Pipeline (`FORWARD` Chain)
When packets arrive from local clients and traverse the Linux routing table (`FORWARD` chain):
1. **Custom Chain Priority**: The custom chain `PARENT_CONTROL_FWD` is prepended to the top of the `FORWARD` chain.
2. **Anti-Bypass Inspection**:
   - Blocks DNS-over-TLS (TCP/UDP port 853).
   - Blocks public DoH provider IPs (e.g., `1.1.1.1`, `8.8.8.8`, etc.) over port 443.
3. **Managed State Evaluation**:
   - If a member is **Locked** -> Drop all traffic.
   - If a member has exhausted their **Daily Quota** (and has no active **Bonus Time**) -> Drop all traffic.
   - If current time matches a **Scheduled Block Window** (and has no active **Bonus Time**) -> Drop all traffic.

### 2.2 Deep Packet Inspection (L7 DPI via `kmod-oaf`)
1. The kernel module `kmod-oaf` hooks into the netfilter architecture to analyze application protocol headers, TLS Server Name Indication (SNI), and destination ports.
2. `parentcontrold` parses signature definitions from `/etc/appfilter/feature_cn.cfg`.
3. Rules are dynamically loaded into kernel space via JSON commands sent to `/dev/appfilter`:
   - `op: 3` -> Clear existing rules.
   - `op: 1` -> Load list of blocked application IDs (`{"apps": [2001, 2002, ...]}`).
   - `op: 4` -> Load list of managed MAC addresses (`{"mac_list": ["AA:BB:CC:..."]}`).

### 2.3 SafeSearch & DNS Redirection
1. **Forced Port 53 Redirection**:
   - An iptables rule in `PARENT_CONTROL_NAT_PRE` redirects all outbound TCP/UDP port 53 traffic to the local router's resolver (`REDIRECT --to-ports 53`).
2. **Search Engine SafeSearch**:
   - Generates `/tmp/dnsmasq.d/parentcontrol.conf`:
     - Google: Redirected to `forcesafesearch.google.com` (216.239.38.120).
     - Bing: Redirected to `strict.bing.com` (204.79.197.220).
     - YouTube: Redirected to `restrict.youtube.com` (216.239.38.119).
   - Reloads dnsmasq gracefully via `SIGHUP`.
