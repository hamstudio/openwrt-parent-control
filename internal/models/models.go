package models

import "time"

// Device 表示局域网内探测到的设备
type Device struct {
	MAC        string    `json:"mac"`
	IP         string    `json:"ip"`
	Hostname   string    `json:"hostname"`
	CustomName string    `json:"custom_name"`
	Vendor     string    `json:"vendor"`
	Online     bool      `json:"online"`
	MemberID   string    `json:"member_id"` // 所属家庭成员 ID，为空表示未分配
	TxRate     uint64    `json:"tx_rate"`   // 实时上行速率 bytes/s
	RxRate     uint64    `json:"rx_rate"`   // 实时下行速率 bytes/s
	TotalBytes uint64    `json:"total_bytes"`
	LastSeen   time.Time `json:"last_seen"`
}

// TimeRange 表示一天内的时间段（24小时制，如 "21:30" 到 "07:00"）
type TimeRange struct {
	StartTime string `json:"start_time"` // "HH:MM"
	EndTime   string `json:"end_time"`   // "HH:MM"
}

// ScheduleRule 时间计划表
type ScheduleRule struct {
	Enabled    bool        `json:"enabled"`
	Days       []int       `json:"days"` // 0=周日, 1=周一, ..., 6=周六
	TimeRanges []TimeRange `json:"time_ranges"`
	Action     string      `json:"action"` // "block" (禁止上网) 或 "allow" (仅允许该时段上网)
}

// Member 表示家庭成员（如“小明”）
type Member struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Avatar         string       `json:"avatar"`           // 头像标识
	DeviceMACs     []string     `json:"device_macs"`      // 绑定的设备 MAC 列表
	Enabled        bool         `json:"enabled"`          // 是否启用管控
	IsLocked       bool         `json:"is_locked"`        // 是否被一键断网
	BonusUntil     *time.Time   `json:"bonus_until"`      // 临时加时到期时间
	QuotaMinutes   int          `json:"quota_minutes"`    // 每日可用上网时长（分钟），0 表示不限
	UsedMinutes    int          `json:"used_minutes"`     // 今日已用时长（分钟）
	LastActiveTime time.Time    `json:"last_active_time"` // 最近活跃时间
	Schedule       ScheduleRule `json:"schedule"`         // 上网时间表
	BlockedAppIDs  []int        `json:"blocked_app_ids"`  // 封禁的 DPI App ID 列表
	SafeSearch     bool         `json:"safe_search"`      // 是否开启 SafeSearch
	BlockAdult     bool         `json:"block_adult"`      // 是否拦截成人网站
	MaxSpeedDown   int          `json:"max_speed_down"`   // 下行限速 (KB/s), 0 表示不限速
	MaxSpeedUp     int          `json:"max_speed_up"`     // 上行限速 (KB/s), 0 表示不限速
}

// AppInfo 表示一个具体的被识别应用
type AppInfo struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ClassID   int    `json:"class_id"`
	ClassName string `json:"class_name"`
	ClassZh   string `json:"class_zh"`
	Selected  bool   `json:"selected,omitempty"`
}

// AppCategory 应用分类
type AppCategory struct {
	ClassID   int       `json:"class_id"`
	ClassName string    `json:"class_name"`
	ClassZh   string    `json:"class_zh"`
	Icon      string    `json:"icon"`
	Apps      []AppInfo `json:"apps"`
}

// GlobalSettings 全局配置
type GlobalSettings struct {
	Enabled           bool     `json:"enabled"`             // 主开关
	WebPort           int      `json:"web_port"`            // 独立 Web 控制台端口 (默认 8088)
	EnforceSafeSearch bool     `json:"enforce_safe_search"` // 全局 SafeSearch
	BlockDoHDoT       bool     `json:"block_doh_dot"`       // 阻断公共 DoH/DoT 防止绕过
	IsolateNewDevices bool     `json:"isolate_new_devices"` // 新设备接入默认隔离（防 MAC 随机化）
	CustomBlacklist   []string `json:"custom_blacklist"`    // 自定义黑名单域名
	CustomWhitelist   []string `json:"custom_whitelist"`    // 自定义白名单域名
	DailyResetHour    int      `json:"daily_reset_hour"`    // 每日配额重置时间（默认 0 点）
}

// SystemStatus 系统运行状态
type SystemStatus struct {
	Running           bool      `json:"running"`
	UptimeSeconds     int64     `json:"uptime_seconds"`
	TotalDevices      int       `json:"total_devices"`
	ActiveDevices     int       `json:"active_devices"`
	ManagedMembers    int       `json:"managed_members"`
	BlockedCountToday int64     `json:"blocked_count_today"`
	KernelDPIReady    bool      `json:"kernel_dpi_ready"`
	AppCount          int       `json:"app_count"`
	ServerTime        time.Time `json:"server_time"`
}
