# Frequently Asked Questions (FAQ)

This document addresses common questions, troubleshooting tips, and best practices for deploying and operating ParentControl Guard.

---

## Table of Contents
- [1. Web Dashboard & LuCI UI](#1-web-dashboard--luci-ui)
  - [Q1.1: Why does the embedded iframe show "SSL Error / Invalid Certificate" when accessing LuCI via HTTPS?](#q11-why-does-the-embedded-iframe-show-ssl-error--invalid-certificate-when-accessing-luci-via-https)
  - [Q1.2: What should I do if the UI is blank or buttons are unresponsive?](#q12-what-should-i-do-if-the-ui-is-blank-or-buttons-are-unresponsive)
  - [Q1.3: How do I reset or clear a forgotten 4-digit PIN code?](#q13-how-do-i-reset-or-clear-a-forgotten-4-digit-pin-code)
- [2. Policy Control & Scheduling](#2-policy-control--scheduling)
  - [Q2.1: How do I configure multiple custom block time ranges (e.g. lunch break + overnight)?](#q21-how-do-i-configure-multiple-custom-block-time-ranges-eg-lunch-break--overnight)
  - [Q2.2: What is the difference between Device-level Lock and Member-level Lock?](#q22-what-is-the-difference-between-device-level-lock-and-member-level-lock)
  - [Q2.3: How does Bonus Time work?](#q23-how-does-bonus-time-work)
- [3. DPI Signature Engine & App Blocking](#3-dpi-signature-engine--app-blocking)
  - [Q3.1: What does "kmod-oaf Not Loaded (Fallback Mode)" mean?](#q31-what-does-kmod-oaf-not-loaded-fallback-mode-mean)
  - [Q3.2: Why can my child still browse after changing MAC addresses on modern phones?](#q32-why-can-my-child-still-browse-after-changing-mac-addresses-on-modern-phones)
  - [Q3.3: How do I add custom apps not currently in the signature database?](#q33-how-do-i-add-custom-apps-not-currently-in-the-signature-database)
- [4. Remote Cloud Relay & Mobile Sync](#4-remote-cloud-relay--mobile-sync)
  - [Q4.1: How can parents manage the router over 4G/5G mobile networks when away from home?](#q41-how-can-parents-manage-the-router-over-4g5g-mobile-networks-when-away-from-home)
  - [Q4.2: Is outbound cloud synchronization secure? Does it expose internal router ports?](#q42-is-outbound-cloud-synchronization-secure-does-it-expose-internal-router-ports)

---

## 1. Web Dashboard & LuCI UI

### Q1.1: Why does the embedded iframe show "SSL Error / Invalid Certificate" when accessing LuCI via HTTPS?

#### 💡 Root Cause
When accessing OpenWrt LuCI via `https://192.168.0.110/`, your browser only trusts the self-signed certificate on **Port 443**.
Because ParentControl Guard operates on separate dedicated ports (`8089 HTTPS` / `8088 HTTP`), modern browsers (Safari, Chrome, Firefox) treat different ports as distinct security origins. For security reasons, browsers **never prompt certificate acceptance dialogs inside background iframes**, resulting in an immediate SSL certificate error.

#### 🛠️ Solution (Choose one):
- **Option A (Authorize Once, Permanent iframe Embed - Recommended)**:
  Open `https://192.168.0.110:8089` in a new browser tab. Click **"Advanced" -> "Proceed to 192.168.0.110 (unsafe)"**. Once accepted, refresh the LuCI page and the embedded dashboard will load smoothly without further warnings.
- **Option B (Direct Fullscreen Access, Zero Certificate Hassle)**:
  Click the **`[HTTP Direct (:8088)]`** or **`[↗ Open Fullscreen Dashboard]`** buttons at the top of the LuCI page for an unobstructed, full-sized management experience.

---

### Q1.2: What should I do if the UI is blank or buttons are unresponsive?
Press `Ctrl + F5` (Windows) or `Cmd + Shift + R` (macOS) to perform a hard refresh and purge cached assets. The frontend supports 8 dynamic locales with crash-safe fallbacks.

---

### Q1.3: How do I reset or clear a forgotten 4-digit PIN code?
Connect to the router via SSH and run:
```bash
# 1. Clear PIN in configuration file
sed -i 's/"pin_code": "[^"]*"/"pin_code": ""/g' /etc/parentcontrol/config.json

# 2. Restart service
/etc/init.d/parentcontrol restart
```

---

## 2. Policy Control & Scheduling

### Q2.1: How do I configure multiple custom block time ranges (e.g. lunch break + overnight)?
1. Click **⚙️ (Edit Rules)** on the target family member card;
2. Scroll to **Time Limits & Block Schedule** and ensure **Enable Schedule** is checked;
3. Select active days (use presets: *Everyday*, *Workdays*, *Weekend*);
4. Click **`+ Add Time Range`** to add multiple intervals (e.g. `12:30 - 13:30` and overnight `21:30 - 07:00`);
5. Click **Save Rules** to apply immediately.

---

### Q2.2: What is the difference between Device-level Lock and Member-level Lock?
- **Member Lock**: Instantly blocks internet access for **all devices** belonging to that family member (e.g. phone + tablet + laptop).
- **Device Lock**: Instantly isolates a **specific MAC address** on the LAN devices table, regardless of whether it is assigned to a member.

---

### Q2.3: How does Bonus Time work?
When a member exhausts their daily quota or is within a blocked time window, parents can click **`+ Bonus Time`** on the member card and grant `+15m`, `+30m`, `+1h`, or `+2h`.
The engine grants temporary network immunity until the countdown expires.

---

## 3. DPI Signature Engine & App Blocking

### Q3.1: What does "kmod-oaf Not Loaded (Fallback Mode)" mean?
Deep Layer-7 application recognition requires the OpenWrt kernel module `kmod-oaf`.
If your router firmware lacks this module, ParentControl Guard automatically falls back to domain-level (DNS) and IP/Port filtering. Instant lock, quotas, SafeSearch, and DoH blocking remain fully functional.

---

### Q3.2: Why can my child still browse after changing MAC addresses on modern phones?
Modern iOS/Android devices enable Private Wi-Fi Addresses (MAC randomization).
#### 💡 Mitigation:
Navigate to the **Global Security** tab and enable **Randomized MAC Quarantine**. Unregistered devices will be blocked by default until approved by a parent.

---

### Q3.3: How do I add custom apps not currently in the signature database?
1. Open the **Signatures (DPI)** tab;
2. Click **`+ Add App`** in the top right;
3. Enter the app name, select a category, and optionally add description notes;
4. Save to instantly register the signature across all member rule selectors.

---

## 4. Remote Cloud Relay & Mobile Sync

### Q4.1: How can parents manage the router over 4G/5G mobile networks when away from home?
Deploy the included serverless relay on Cloudflare Workers:
1. Follow [Cloudflare Worker Deployment Guide](../cloud/worker/README.md) to deploy your free worker;
2. Enter the Worker URL and Secret in the router's **Global Security** settings;
3. Enter the same URL in the mobile app to manage parental rules from anywhere.

---

### Q4.2: Is outbound cloud synchronization secure? Does it expose internal router ports?
**Yes, 100% secure**. The `CloudSyncer` daemon initiates **outbound-only** HTTPS / WSS requests to the relay server. **No public IP, DDNS, or port forwarding is required**, leaving your router completely invisible to external port scanners.

---

### Q4.3: Why deploy a standalone Go Relay Server on a VPS?
For domestic networks where Cloudflare Workers experience connection latency or throttling:
1. **Millisecond Response**: Built-in in-memory PubSub MQ and bidirectional WebSocket push;
2. **Zero Dependencies**: Single binary with ~10MB memory usage, 10-second Docker Compose startup;
3. **Offline Buffering**: Rules and commands are safely queued if the router temporarily disconnects.
