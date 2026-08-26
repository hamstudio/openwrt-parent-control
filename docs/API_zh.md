# RESTful API 规范文档

[English](API.md) | [简体中文](API_zh.md)

ParentControl Daemon 内置轻量级 HTTP API 服务，支持外部系统、移动 App 或第三方自动化调用。默认服务监听端口为 `8088`。

---

## 1. 系统与状态接口

### 1.1 获取系统运行状态
- **请求方法**：`GET`
- **接口路径**：`/api/status`
- **响应示例**：
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

### 1.2 获取局域网已探测设备列表
- **请求方法**：`GET`
- **接口路径**：`/api/devices`
- **响应示例**：
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

### 1.3 获取 DPI 应用分类与特征列表
- **请求方法**：`GET`
- **接口路径**：`/api/apps`
- **响应示例**：
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

## 2. 成员管控接口

### 2.1 获取所有受管成员列表
- **请求方法**：`GET`
- **接口路径**：`/api/members`

### 2.2 创建或更新受管成员
- **请求方法**：`POST`
- **接口路径**：`/api/members`
- **请求体格式**：
```json
{
  "id": "m_xiaoming",
  "name": "小明",
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

### 2.3 删除受管成员
- **请求方法**：`DELETE`
- **接口路径**：`/api/members/{member_id}`

### 2.4 一键断网
- **请求方法**：`POST`
- **接口路径**：`/api/members/{member_id}/lock`

### 2.5 恢复上网
- **请求方法**：`POST`
- **接口路径**：`/api/members/{member_id}/unlock`

### 2.6 奖励加时
- **请求方法**：`POST`
- **接口路径**：`/api/members/{member_id}/bonus?minutes=30`

---

## 3. 全局设置接口

### 3.1 获取全局安全设置
- **请求方法**：`GET`
- **接口路径**：`/api/settings`

### 3.2 保存全局安全设置
- **请求方法**：`POST`
- **接口路径**：`/api/settings`
- **请求体格式**：
```json
{
  "enabled": true,
  "enforce_safe_search": true,
  "block_doh_dot": true,
  "isolate_new_devices": false
}
```
