# Security Policy

[English](SECURITY.md) | [简体中文](SECURITY_zh.md)

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

---

## 🔒 Reporting a Vulnerability

If you discover a security vulnerability in ParentControl Guard, please **DO NOT** open a public issue.

Instead, please send an email to the maintainers or report via GitHub Private Vulnerability Reporting if enabled.

Please include:
1. Description of the vulnerability.
2. Steps to reproduce or a Proof of Concept (PoC).
3. Potential impact on router security or network traffic isolation.

We will acknowledge your report within 48 hours and work with you on a coordinated disclosure and patch release.

---

## 🛡️ Security Best Practices for Deployment

1. **PIN Protection**: Always set a 4-digit PIN in the Web Console or mobile app to prevent unauthorized changes to filtering rules.
2. **Local Access Isolation**: The Web Console (`:8088` / `:8089`) listens on all local interfaces by default. Ensure your OpenWrt firewall does not expose these ports to the WAN interface.
3. **Cloudflare Worker Secret**: When enabling remote management, configure a strong `cloud_device_secret` to authenticate communications between the router and the Cloudflare Worker.
