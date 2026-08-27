include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-parentcontrol
PKG_VERSION:=1.0.0
PKG_RELEASE:=1

PKG_MAINTAINER:=ParentControl Guard Team
PKG_LICENSE:=MIT

LUCI_TITLE:=LuCI support for ParentControl Guard
LUCI_DEPENDS:=+kmod-oaf +iptables +dnsmasq-full
LUCI_PKGARCH:=all

include $(TOPDIR)/feeds/luci/luci.mk

# Call BuildPackage - OpenWrt buildroot signature
$(eval $(call BuildPackage,$(PKG_NAME)))
