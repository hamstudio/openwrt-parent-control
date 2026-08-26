# 部署与运维指南 (Deployment & Operations)

## 1. 硬件与系统要求

- **操作系统**：OpenWrt 21.02 / 22.03 / 23.05+ 或基于 OpenWrt 的衍生发行版（如 iStoreOS）。
- **架构支持**：x86_64, aarch64, arm, mips 等（只需在编译时指定对应的 `GOARCH`）。
- **内核依赖**：
  - `iptables` 或 `nftables`
  - `dnsmasq-full`
  - `kmod-oaf` (用于深度包检测 L7 DPI 应用识别)

---

## 2. 自动化一键部署

1. 确认主机的 Go 开发环境已安装：`go version`
2. 执行一键部署脚本：
```bash
./scripts/deploy.sh
```

---

## 3. 手动编译与部署步骤

### 3.1 交叉编译 Go 守护进程
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/parentcontrold ./cmd/parentcontrold
```

### 3.2 上传至路由器并赋予权限
```bash
scp bin/parentcontrold root@<路由器IP>:/usr/bin/parentcontrold
ssh root@<路由器IP> "chmod +x /usr/bin/parentcontrold"
```

### 3.3 配置开机自启并启动服务
```bash
scp rootfs/etc/init.d/parentcontrol root@<路由器IP>:/etc/init.d/parentcontrol
ssh root@<路由器IP> "chmod +x /etc/init.d/parentcontrol && /etc/init.d/parentcontrol enable && /etc/init.d/parentcontrol start"
```

### 3.4 安装 LuCI 控制面板
```bash
scp rootfs/usr/share/luci/menu.d/luci-app-parentcontrol.json root@<路由器IP>:/usr/share/luci/menu.d/
scp rootfs/usr/share/rpcd/acl.d/luci-app-parentcontrol.json root@<路由器IP>:/usr/share/rpcd/acl.d/
scp rootfs/www/luci-static/resources/view/parentcontrol/overview.js root@<路由器IP>:/www/luci-static/resources/view/parentcontrol/
ssh root@<路由器IP> "rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache*"
```

---

## 4. 日常维护命令

- **查看服务状态**：`/etc/init.d/parentcontrol status`
- **重启服务**：`/etc/init.d/parentcontrol restart`
- **停止服务**：`/etc/init.d/parentcontrol stop`
- **查看实时日志**：`logread -e parentcontrold`
- **查看 iptables 阻断规则**：`iptables -t filter -L PARENT_CONTROL_FWD -n -v`
- **查看 DNS 劫持规则**：`cat /tmp/dnsmasq.d/parentcontrol.conf`
