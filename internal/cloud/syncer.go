package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"parentcontrol/internal/config"
	"parentcontrol/internal/device"
	"parentcontrol/internal/dpi"
	"parentcontrol/internal/models"
	"parentcontrol/internal/quota"
)

// Syncer 负责路由器与 Cloudflare Worker 公网中继之间的出站双向同步
type Syncer struct {
	engine     *quota.PolicyEngine
	devTracker *device.DeviceTracker
	dpiMgr     *dpi.DPIManager
	config     *config.ConfigStore
	httpClient *http.Client
}

// NewSyncer 创建同步器实例
func NewSyncer(
	engine *quota.PolicyEngine,
	devTracker *device.DeviceTracker,
	dpiMgr *dpi.DPIManager,
	config *config.ConfigStore,
) *Syncer {
	return &Syncer{
		engine:     engine,
		devTracker: devTracker,
		dpiMgr:     dpiMgr,
		config:     config,
		httpClient: &http.Client{
			Timeout: 35 * time.Second,
		},
	}
}

// SetHTTPTransport 设置自定义 Transport (用于测试与内存 Mock)
func (s *Syncer) SetHTTPTransport(tr http.RoundTripper) {
	s.httpClient.Transport = tr
}

// Start 启动后台同步与长轮询协程
func (s *Syncer) Start(ctx context.Context) {
	log.Printf("[CloudSyncer] Cloud synchronization daemon started")

	// 1. 周期性全量状态上报协程 (每 15 秒)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		// 启动立即同步一次
		s.syncState()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.syncState()
			}
		}
	}()

	// 2. 长轮询指令监听协程 (秒级实时响应)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				s.pollAndExecuteCommands()
			}
		}
	}()
}

// syncState 上报本地当前状态并处理附带的指令
func (s *Syncer) syncState() {
	settings := s.config.Data.Settings
	if !settings.CloudSyncEnabled || strings.TrimSpace(settings.CloudWorkerURL) == "" {
		return
	}

	workerURL := strings.TrimRight(settings.CloudWorkerURL, "/")
	endpoint := fmt.Sprintf("%s/api/router/sync", workerURL)

	devices := s.devTracker.ScanDevices()
	activeCount := 0
	for _, d := range devices {
		if d.Online {
			activeCount++
		}
	}

	members := s.engine.GetMembers()

	status := models.SystemStatus{
		Running:        true,
		TotalDevices:   len(devices),
		ActiveDevices:  activeCount,
		ManagedMembers: len(members),
		KernelDPIReady: s.dpiMgr.IsReady(),
		AppCount:       len(s.dpiMgr.GetCategories()),
		PinRequired:    settings.PinCode != "",
		ServerTime:     time.Now(),
	}

	payload := map[string]interface{}{
		"status":     status,
		"members":    members,
		"devices":    devices,
		"categories": s.dpiMgr.GetCategories(),
		"settings":   settings,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[CloudSyncer] Failed to marshal sync payload: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		log.Printf("[CloudSyncer] Failed to create sync request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if settings.CloudDeviceSecret != "" {
		req.Header.Set("X-Router-Secret", settings.CloudDeviceSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[CloudSyncer] HTTP error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var res struct {
		Success  bool                  `json:"success"`
		Commands []models.CloudCommand `json:"commands"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && len(res.Commands) > 0 {
		log.Printf("[CloudSyncer] Received %d command(s) during state sync", len(res.Commands))
		s.executeCommands(res.Commands)
	}
}

// pollAndExecuteCommands 挂起长轮询拉取指令
func (s *Syncer) pollAndExecuteCommands() {
	settings := s.config.Data.Settings
	if !settings.CloudSyncEnabled || strings.TrimSpace(settings.CloudWorkerURL) == "" {
		time.Sleep(3 * time.Second)
		return
	}

	workerURL := strings.TrimRight(settings.CloudWorkerURL, "/")
	endpoint := fmt.Sprintf("%s/api/router/poll", workerURL)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		time.Sleep(2 * time.Second)
		return
	}
	if settings.CloudDeviceSecret != "" {
		req.Header.Set("X-Router-Secret", settings.CloudDeviceSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		time.Sleep(3 * time.Second)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		time.Sleep(3 * time.Second)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var res struct {
		Commands []models.CloudCommand `json:"commands"`
	}
	if err := json.Unmarshal(bodyBytes, &res); err == nil && len(res.Commands) > 0 {
		log.Printf("[CloudSyncer] Poll received %d command(s)", len(res.Commands))
		s.executeCommands(res.Commands)
	}
}

// executeCommands 派发指令并在本地 PolicyEngine 中生效
func (s *Syncer) executeCommands(commands []models.CloudCommand) {
	needSaveConfig := false

	for _, cmd := range commands {
		log.Printf("[CloudSyncer] Executing command: %s (Type: %s, Member: %s)", cmd.ID, cmd.Type, cmd.MemberID)

		switch cmd.Type {
		case "LOCK":
			_ = s.engine.LockMember(cmd.MemberID)

		case "UNLOCK":
			_ = s.engine.UnlockMember(cmd.MemberID)

		case "BONUS":
			var payload struct {
				Minutes int `json:"minutes"`
			}
			minutes := 30
			if err := json.Unmarshal(cmd.Payload, &payload); err == nil && payload.Minutes > 0 {
				minutes = payload.Minutes
			}
			_ = s.engine.BonusMember(cmd.MemberID, minutes)

		case "SET_MEMBER":
			var m models.Member
			if err := json.Unmarshal(cmd.Payload, &m); err == nil {
				s.engine.SetMember(m)
				needSaveConfig = true
			}

		case "DELETE_MEMBER":
			s.engine.DeleteMember(cmd.MemberID)
			needSaveConfig = true

		case "UPDATE_SETTINGS":
			var newSettings models.GlobalSettings
			if err := json.Unmarshal(cmd.Payload, &newSettings); err == nil {
				s.engine.UpdateSettings(newSettings)
				s.config.Data.Settings = newSettings
				_ = s.config.Save()
			}

		case "ADD_APP":
			var app models.AppInfo
			if err := json.Unmarshal(cmd.Payload, &app); err == nil {
				_, _ = s.dpiMgr.AddApp(app)
			}

		case "DELETE_APP":
			var payload struct {
				AppID int `json:"app_id"`
			}
			if err := json.Unmarshal(cmd.Payload, &payload); err == nil {
				_ = s.dpiMgr.DeleteApp(payload.AppID)
			}
		}
	}

	if needSaveConfig {
		s.config.Data.Members = s.engine.GetMembers()
		_ = s.config.Save()
	}

	// 立即触发规则重评与内核应用
	s.engine.EvaluateAndApply(time.Now())
}
