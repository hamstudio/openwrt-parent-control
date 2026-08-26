# Deployment & Operations Guide

[English](DEPLOYMENT.md) | [简体中文](DEPLOYMENT_zh.md)

## 1. Prerequisites & System Requirements

- **Operating System**: OpenWrt 21.02 / 22.03 / 23.05+ or derivatives (e.g., iStoreOS).
- **Supported Architectures**: x86_64, aarch64, arm, mips (set `GOARCH` during compilation).
- **Core Dependencies**:
  - `iptables` or `nftables`
  - `dnsmasq-full`
  - `kmod-oaf` (for L7 Deep Packet Inspection application recognition)

---

## 2. Automated One-Click Deployment

1. Ensure Go is installed on your local development machine: `go version`
2. Run the deployment script:
```bash
./scripts/deploy.sh
```

---

## 3. Manual Build & Deployment

### 3.1 Cross-Compile Go Daemon
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/parentcontrold ./cmd/parentcontrold
```

### 3.2 Upload Binary to Router
```bash
scp bin/parentcontrold root@<Router-IP>:/usr/bin/parentcontrold
ssh root@<Router-IP> "chmod +x /usr/bin/parentcontrold"
```

### 3.3 Configure & Start procd Service
```bash
scp rootfs/etc/init.d/parentcontrol root@<Router-IP>:/etc/init.d/parentcontrol
ssh root@<Router-IP> "chmod +x /etc/init.d/parentcontrol && /etc/init.d/parentcontrol enable && /etc/init.d/parentcontrol start"
```

### 3.4 Install Native LuCI Plugin
```bash
scp rootfs/usr/share/luci/menu.d/luci-app-parentcontrol.json root@<Router-IP>:/usr/share/luci/menu.d/
scp rootfs/usr/share/rpcd/acl.d/luci-app-parentcontrol.json root@<Router-IP>:/usr/share/rpcd/acl.d/
scp rootfs/www/luci-static/resources/view/parentcontrol/overview.js root@<Router-IP>:/www/luci-static/resources/view/parentcontrol/
ssh root@<Router-IP> "rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache*"
```

---

## 4. Useful Operations & Troubleshooting Commands

- **Check Service Status**: `/etc/init.d/parentcontrol status`
- **Restart Service**: `/etc/init.d/parentcontrol restart`
- **Stop Service**: `/etc/init.d/parentcontrol stop`
- **Inspect Live Logs**: `logread -e parentcontrold`
- **Inspect Active Netfilter Blocks**: `iptables -t filter -L PARENT_CONTROL_FWD -n -v`
- **Inspect Generated DNS Rules**: `cat /tmp/dnsmasq.d/parentcontrol.conf`
