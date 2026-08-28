package models

import (
	"encoding/json"
	"time"
)

// Device represents a network device detected on the LAN
type Device struct {
	MAC        string    `json:"mac"`
	IP         string    `json:"ip"`
	Hostname   string    `json:"hostname"`
	CustomName string    `json:"custom_name"`
	Vendor     string    `json:"vendor"`
	Online     bool      `json:"online"`
	MemberID   string    `json:"member_id"` // Associated family member ID, empty if unassigned
	IsLocked   bool      `json:"is_locked"` // Whether blocked via one-click lock
	TxRate     uint64    `json:"tx_rate"`   // Real-time upstream rate in bytes/s
	RxRate     uint64    `json:"rx_rate"`   // Real-time downstream rate in bytes/s
	TotalBytes uint64    `json:"total_bytes"`
	LastSeen   time.Time `json:"last_seen"`
}

// TimeRange represents a time interval within a day (24-hour format, e.g. "21:30" to "07:00")
type TimeRange struct {
	StartTime string `json:"start_time"` // "HH:MM"
	EndTime   string `json:"end_time"`   // "HH:MM"
}

// ScheduleRule represents internet schedule access rules
type ScheduleRule struct {
	Enabled    bool        `json:"enabled"`
	Days       []int       `json:"days"` // 0=Sunday, 1=Monday, ..., 6=Saturday
	TimeRanges []TimeRange `json:"time_ranges"`
	Action     string      `json:"action"` // "block" (disallow access) or "allow" (allow access only during ranges)
}

// Member represents a managed family member
type Member struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Avatar         string       `json:"avatar"`           // Avatar identifier
	DeviceMACs     []string     `json:"device_macs"`      // List of bound device MAC addresses
	Enabled        bool         `json:"enabled"`          // Whether management is enabled
	IsLocked       bool         `json:"is_locked"`        // Whether manually locked/blocked
	BonusUntil     *time.Time   `json:"bonus_until"`      // Expiration time of temporary bonus time
	QuotaMinutes   int          `json:"quota_minutes"`    // Daily allowed online duration in minutes (0 for unlimited)
	UsedMinutes    int          `json:"used_minutes"`     // Minutes used today
	LastActiveTime time.Time    `json:"last_active_time"` // Last active timestamp
	Schedule       ScheduleRule `json:"schedule"`         // Internet schedule rule
	BlockedAppIDs  []int        `json:"blocked_app_ids"`  // Blocked DPI App IDs
	SafeSearch     bool         `json:"safe_search"`      // Whether SafeSearch is enforced
	BlockAdult     bool         `json:"block_adult"`      // Whether adult content is blocked
	MaxSpeedDown   int          `json:"max_speed_down"`   // Downlink speed limit (KB/s), 0 for unlimited
	MaxSpeedUp     int          `json:"max_speed_up"`     // Uplink speed limit (KB/s), 0 for unlimited
}

// AppInfo represents a classified application recognized by DPI
type AppInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ClassID     int    `json:"class_id"`
	ClassName   string `json:"class_name"`
	ClassZh     string `json:"class_zh"`
	Description string `json:"description,omitempty"`
	IsCustom    bool   `json:"is_custom,omitempty"`
	Selected    bool   `json:"selected,omitempty"`
}

// AppCategory represents an application category grouping
type AppCategory struct {
	ClassID   int       `json:"class_id"`
	ClassName string    `json:"class_name"`
	ClassZh   string    `json:"class_zh"`
	Icon      string    `json:"icon"`
	IsCustom  bool      `json:"is_custom,omitempty"`
	Apps      []AppInfo `json:"apps"`
}

// GlobalSettings represents system-wide security and operational settings
type GlobalSettings struct {
	Enabled           bool     `json:"enabled"`              // Master switch
	WebPort           int      `json:"web_port"`             // Web console port (default 8088)
	PinCode           string   `json:"pin_code"`             // Optional 4-digit PIN lock, empty if disabled
	CloudSyncEnabled  bool     `json:"cloud_sync_enabled"`   // Whether Cloudflare Worker sync is enabled
	CloudWorkerURL    string   `json:"cloud_worker_url"`     // Cloudflare Worker API URL (e.g. https://xxx.workers.dev)
	CloudDeviceSecret string   `json:"cloud_device_secret"`  // Shared device secret (optional)
	EnforceSafeSearch bool     `json:"enforce_safe_search"`  // Global SafeSearch enforcement
	BlockDoHDoT       bool     `json:"block_doh_dot"`        // Block public DoH/DoT to prevent bypass
	IsolateNewDevices bool     `json:"isolate_new_devices"`  // Isolate unrecognized devices by default (anti-MAC randomization)
	CustomBlacklist   []string `json:"custom_blacklist"`     // Custom blacklisted domains
	CustomWhitelist   []string `json:"custom_whitelist"`     // Custom whitelisted domains
	DailyResetHour    int      `json:"daily_reset_hour"`     // Daily quota reset hour (default 0 / midnight)
}

// CloudCommand represents a command dispatched from the cloud
type CloudCommand struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"` // LOCK, UNLOCK, BONUS, SET_MEMBER, DELETE_MEMBER, UPDATE_SETTINGS, ADD_APP, DELETE_APP
	MemberID  string          `json:"member_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt int64           `json:"created_at"`
}

// SystemStatus represents runtime status of the daemon
type SystemStatus struct {
	Running           bool      `json:"running"`
	UptimeSeconds     int64     `json:"uptime_seconds"`
	TotalDevices      int       `json:"total_devices"`
	ActiveDevices     int       `json:"active_devices"`
	ManagedMembers    int       `json:"managed_members"`
	BlockedCountToday int64     `json:"blocked_count_today"`
	KernelDPIReady    bool      `json:"kernel_dpi_ready"`
	AppCount          int       `json:"app_count"`
	PinRequired       bool      `json:"pin_required"` // Whether 4-digit PIN lock is required
	ServerTime        time.Time `json:"server_time"`
	TimezoneName      string    `json:"timezone_name"`
	TimezoneOffset    int       `json:"timezone_offset"`
}
