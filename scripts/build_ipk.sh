#!/usr/bin/env bash
# OpenWrt .ipk build and packaging script
set -e

ARCH="${1:-x86_64}"
VERSION="${2:-1.0.0-1}"
PKG_NAME="luci-app-parentcontrol"
OUTPUT_DIR="dist"

echo "=================================================="
echo "  Packaging OpenWrt Package: ${PKG_NAME}_${VERSION}_${ARCH}.ipk"
echo "=================================================="

# 1. Determine Go target architecture
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

# 2. Build Go daemon binary
echo "[1/4] Building Go daemon binary (${GOARCH})..."
CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM} go build -trimpath -ldflags="-s -w" -o "${DATA_DIR}/usr/bin/parentcontrold" ./cmd/parentcontrold

# 3. Copy rootfs resources and LuCI frontend
echo "[2/4] Assembling rootfs files and LuCI interface..."
cp rootfs/etc/init.d/parentcontrol "${DATA_DIR}/etc/init.d/parentcontrol"
chmod 755 "${DATA_DIR}/etc/init.d/parentcontrol"

cp rootfs/usr/share/luci/menu.d/luci-app-parentcontrol.json "${DATA_DIR}/usr/share/luci/menu.d/"
cp rootfs/usr/share/rpcd/acl.d/luci-app-parentcontrol.json "${DATA_DIR}/usr/share/rpcd/acl.d/"
cp rootfs/www/luci-static/resources/view/parentcontrol/overview.js "${DATA_DIR}/www/luci-static/resources/view/parentcontrol/"

# 4. Generate opkg control metadata and installation scripts
echo "[3/4] Generating opkg control metadata..."
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

# 5. Package as standard .ipk (using ustar format compatible with OpenWrt busybox tar)
echo "[4/4] Building .ipk package archive..."
echo "2.0" > "${WORK_DIR}/debian-binary"

export COPYFILE_DISABLE=1
(cd "${DATA_DIR}" && tar --format=ustar -czf "${WORK_DIR}/data.tar.gz" .)
(cd "${CONTROL_DIR}" && tar --format=ustar -czf "${WORK_DIR}/control.tar.gz" .)

ROOT_DIR="$(pwd)"
IPK_FILE="${OUTPUT_DIR}/${PKG_NAME}_${VERSION}_${ARCH}.ipk"
IPK_ABS_PATH="${ROOT_DIR}/${IPK_FILE}"
(cd "${WORK_DIR}" && tar --format=ustar -czf "${IPK_ABS_PATH}" ./debian-binary ./control.tar.gz ./data.tar.gz)

# Cleanup temporary directory
rm -rf "${WORK_DIR}"

echo "=================================================="
echo "✅ Build completed successfully! Output file:"
echo "   -> ${IPK_FILE} ($(du -h "${IPK_FILE}" | awk '{print $1}'))"
echo "=================================================="
echo "To install on router:"
echo "   opkg install ${IPK_FILE}"
echo "=================================================="
