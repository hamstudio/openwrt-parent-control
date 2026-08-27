#!/usr/bin/env bash
# OpenWrt .ipk 一键打包发布脚本
set -e

ARCH="${1:-x86_64}"
VERSION="${2:-1.0.0-1}"
PKG_NAME="luci-app-parentcontrol"
OUTPUT_DIR="dist"

echo "=================================================="
echo "  正在打包 OpenWrt 插件: ${PKG_NAME}_${VERSION}_${ARCH}.ipk"
echo "=================================================="

# 1. 确定 Go 编译目标架构
case "$ARCH" in
  x86_64)
    GOARCH="amd64"
    GOARM=""
    ;;
  aarch64|arm64)
    GOARCH="arm64"
    GOARM=""
    ;;
  arm_cortex-a7_neon-vfpv4|armv7)
    GOARCH="arm"
    GOARM="7"
    ;;
  mips_24kc)
    GOARCH="mips"
    GOARM=""
    ;;
  mipsel_24kc)
    GOARCH="mipsle"
    GOARM=""
    ;;
  *)
    GOARCH="amd64"
    ;;
esac

WORK_DIR=$(mktemp -d)
DATA_DIR="${WORK_DIR}/data"
CONTROL_DIR="${WORK_DIR}/control"

mkdir -p "${DATA_DIR}/usr/bin"
mkdir -p "${DATA_DIR}/etc/init.d"
mkdir -p "${DATA_DIR}/usr/share/luci/menu.d"
mkdir -p "${DATA_DIR}/usr/share/rpcd/acl.d"
mkdir -p "${DATA_DIR}/www/luci-static/resources/view/parentcontrol"
mkdir -p "${CONTROL_DIR}"
mkdir -p "${OUTPUT_DIR}"

# 2. 编译 Go 守护进程
echo "[1/4] 编译 Go 后端守护进程 (${GOARCH})..."
CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM} go build -trimpath -ldflags="-s -w" -o "${DATA_DIR}/usr/bin/parentcontrold" ./cmd/parentcontrold

# 3. 复制 rootfs 资源与 LuCI 前端
echo "[2/4] 组装 rootfs 文件与 LuCI 界面..."
cp rootfs/etc/init.d/parentcontrol "${DATA_DIR}/etc/init.d/parentcontrol"
chmod 755 "${DATA_DIR}/etc/init.d/parentcontrol"

cp rootfs/usr/share/luci/menu.d/luci-app-parentcontrol.json "${DATA_DIR}/usr/share/luci/menu.d/"
cp rootfs/usr/share/rpcd/acl.d/luci-app-parentcontrol.json "${DATA_DIR}/usr/share/rpcd/acl.d/"
cp rootfs/www/luci-static/resources/view/parentcontrol/overview.js "${DATA_DIR}/www/luci-static/resources/view/parentcontrol/"

# 4. 生成 control 元数据与安装控制脚本
echo "[3/4] 生成 opkg control 元数据..."
cat > "${CONTROL_DIR}/control" <<EOF
Package: ${PKG_NAME}
Version: ${VERSION}
Depends: kmod-oaf, iptables, dnsmasq-full
Section: luci
Architecture: ${ARCH}
Maintainer: ParentControl Guard Team
Description: LuCI support for ParentControl Guard with L7 DPI and Cloudflare Worker sync.
EOF

cat > "${CONTROL_DIR}/postinst" <<'EOF'
#!/bin/sh
if [ -z "$IPKG_INSTROOT" ]; then
    chmod +x /usr/bin/parentcontrold
    chmod +x /etc/init.d/parentcontrol
    /etc/init.d/parentcontrol enable
    /etc/init.d/parentcontrol restart
    rm -f /tmp/luci-indexcache* /tmp/luci-modulecache/* 2>/dev/null || true
fi
exit 0
EOF
chmod 755 "${CONTROL_DIR}/postinst"

cat > "${CONTROL_DIR}/prerm" <<'EOF'
#!/bin/sh
if [ -z "$IPKG_INSTROOT" ]; then
    /etc/init.d/parentcontrol stop 2>/dev/null || true
    /etc/init.d/parentcontrol disable 2>/dev/null || true
fi
exit 0
EOF
chmod 755 "${CONTROL_DIR}/prerm"

# 5. 打包成标准 .ipk (使用 ustar 格式兼容 OpenWrt busybox tar)
echo "[4/4] 压制 .ipk 安装包..."
echo "2.0" > "${WORK_DIR}/debian-binary"

export COPYFILE_DISABLE=1
(cd "${DATA_DIR}" && tar --format=ustar -czf "${WORK_DIR}/data.tar.gz" .)
(cd "${CONTROL_DIR}" && tar --format=ustar -czf "${WORK_DIR}/control.tar.gz" .)

IPK_FILE="${OUTPUT_DIR}/${PKG_NAME}_${VERSION}_${ARCH}.ipk"
(cd "${WORK_DIR}" && tar --format=ustar -czf "${OLDPWD}/${IPK_FILE}" ./debian-binary ./control.tar.gz ./data.tar.gz)

# 清理临时目录
rm -rf "${WORK_DIR}"

echo "=================================================="
echo "✅ 打包完成！输出文件:"
echo "   -> ${IPK_FILE} ($(du -h "${IPK_FILE}" | awk '{print $1}'))"
echo "=================================================="
echo "在路由器上安装方式:"
echo "   opkg install ${IPK_FILE}"
echo "=================================================="
