#!/bin/bash
set -e

ROUTER_IP="192.168.0.110"
ROUTER_USER="root"

echo "=== 1. 正在编译 Go 二进制 parentcontrold (linux/amd64) ==="
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/parentcontrold ./cmd/parentcontrold

echo "=== 2. 上传文件至路由器 $ROUTER_IP ==="
ssh -o BatchMode=yes $ROUTER_USER@$ROUTER_IP "mkdir -p /etc/parentcontrol /usr/share/luci/menu.d /usr/share/rpcd/acl.d /www/luci-static/resources/view/parentcontrol; killall parentcontrold 2>/dev/null || true"

# 上传二进制
scp bin/parentcontrold $ROUTER_USER@$ROUTER_IP:/usr/bin/parentcontrold
ssh -o BatchMode=yes $ROUTER_USER@$ROUTER_IP "chmod +x /usr/bin/parentcontrold"

# 上传 init 脚本
scp rootfs/etc/init.d/parentcontrol $ROUTER_USER@$ROUTER_IP:/etc/init.d/parentcontrol
ssh -o BatchMode=yes $ROUTER_USER@$ROUTER_IP "chmod +x /etc/init.d/parentcontrol"

# 上传 LuCI 文件
scp rootfs/usr/share/luci/menu.d/luci-app-parentcontrol.json $ROUTER_USER@$ROUTER_IP:/usr/share/luci/menu.d/
scp rootfs/usr/share/rpcd/acl.d/luci-app-parentcontrol.json $ROUTER_USER@$ROUTER_IP:/usr/share/rpcd/acl.d/
scp rootfs/www/luci-static/resources/view/parentcontrol/overview.js $ROUTER_USER@$ROUTER_IP:/www/luci-static/resources/view/parentcontrol/

echo "=== 3. 重载服务与 LuCI 缓存 ==="
ssh -o BatchMode=yes $ROUTER_USER@$ROUTER_IP "/etc/init.d/parentcontrol enable && /etc/init.d/parentcontrol restart && rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache*"

echo "=== 部署完成！==="
echo "Web 控制台地址: http://$ROUTER_IP:8088"
