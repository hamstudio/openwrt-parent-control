# RESTful API Reference

[English](API.md) | [简体中文](API_zh.md)

ParentControl Daemon embeds a lightweight HTTP RESTful API on port `8088` by default, allowing seamless integration with mobile apps, home automation platforms, and external scripts.

---

## 1. System & Diagnostic Endpoints

### 1.1 Get System Status
- **Method**: `GET`
- **Path**: `/api/status`
- **Response Example**:
```json
{
  "running": true,
  "uptime_seconds": 3600,
  "total_devices": 39,
  "active_devices": 12,
  "managed_members": 2,
  "blocked_count_today": 0,
  "kernel_dpi_ready": true,
  "app_count": 8,
  "server_time": "2026-08-27T07:00:00Z"
}
```

### 1.2 List Discovered LAN Devices
- **Method**: `GET`
- **Path**: `/api/devices`
- **Response Example**:
```json
[
  {
    "mac": "F0:18:98:AA:BB:CC",
    "ip": "192.168.0.150",
    "hostname": "iPhone-14",
    "vendor": "Apple",
    "online": true,
    "rx_rate": 1048576,
    "tx_rate": 524288,
    "total_bytes": 104857600,
    "last_seen": "2026-08-27T07:15:00Z"
  }
]
```

### 1.3 List DPI Application Categories & Signatures
- **Method**: `GET`
- **Path**: `/api/apps`
- **Response Example**:
```json
[
  {
    "class_id": 2,
    "class_name": "game",
    "class_zh": "游戏",
    "icon": "gamepad",
    "apps": [
      { "id": 2001, "name": "王者荣耀", "class_id": 2, "class_zh": "游戏" },
      { "id": 2002, "name": "和平精英", "class_id": 2, "class_zh": "游戏" },
      { "id": 2023, "name": "原神", "class_id": 2, "class_zh": "游戏" }
    ]
  }
]
```

---

## 2. Family Member & Policy Management

### 2.1 List All Managed Members
- **Method**: `GET`
- **Path**: `/api/members`

### 2.2 Create or Update a Member
- **Method**: `POST`
- **Path**: `/api/members`
- **Request Body**:
```json
{
  "id": "m_xiaoming",
  "name": "Tom",
  "avatar": "boy",
  "device_macs": [
    "F0:18:98:AA:BB:CC"
  ],
  "enabled": true,
  "quota_minutes": 120,
  "schedule": {
    "enabled": true,
    "days": [0, 1, 2, 3, 4, 5, 6],
    "time_ranges": [
      { "start_time": "21:30", "end_time": "07:00" }
    ],
    "action": "block"
  },
  "blocked_app_ids": [2001, 2002, 3001],
  "safe_search": true,
  "block_adult": true
}
```

### 2.3 Delete a Member
- **Method**: `DELETE`
- **Path**: `/api/members/{member_id}`

### 2.4 Instant Internet Lock
- **Method**: `POST`
- **Path**: `/api/members/{member_id}/lock`

### 2.5 Resume / Unlock Internet Access
- **Method**: `POST`
- **Path**: `/api/members/{member_id}/unlock`

### 2.6 Reward Bonus Time
- **Method**: `POST`
- **Path**: `/api/members/{member_id}/bonus?minutes=30`

---

## 3. Global Settings Endpoints

### 3.1 Get Global Settings
- **Method**: `GET`
- **Path**: `/api/settings`

### 3.2 Update Global Settings
- **Method**: `POST`
- **Path**: `/api/settings`
- **Request Body**:
```json
{
  "enabled": true,
  "enforce_safe_search": true,
  "block_doh_dot": true,
  "isolate_new_devices": false
}
```
