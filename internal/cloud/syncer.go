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

// Syncer handles outbound bidirectional sync between router and Cloudflare Worker
type Syncer struct {
	engine     *quota.PolicyEngine
	devTracker *device.DeviceTracker
	dpiMgr     *dpi.DPIManager
	config     *config.ConfigStore
	httpClient *http.Client
}

// NewSyncer creates a new Syncer instance
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

// SetHTTPTransport sets custom Transport (for testing with mock servers)
func (s *Syncer) SetHTTPTransport(tr http.RoundTripper) {
	s.httpClient.Transport = tr
}

// Start launches background state sync and real-time command listener
func (s *Syncer) Start(ctx context.Context) {
	log.Printf("[CloudSyncer] Cloud synchronization daemon started")

	// 主控分发协程：根据配置 URL 自适应启动 WebSocket 长连接或 HTTP 轮询
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				settings := s.config.Data.Settings
				if !settings.CloudSyncEnabled || strings.TrimSpace(settings.CloudWorkerURL) == "" {
					time.Sleep(3 * time.Second)
					continue
				}

				rawURL := strings.TrimSpace(settings.CloudWorkerURL)
				if strings.HasPrefix(rawURL, "ws://") || strings.HasPrefix(rawURL, "wss://") {
					// 启动 WebSocket 双向实时长连接
					s.runWebSocketLoop(ctx, rawURL, settings.CloudDeviceSecret)
				} else {
					// 启动标准 HTTP 轮询同步
					s.runHTTPLoop(ctx)
				}
			}
		}
	}()
}

// runWebSocketLoop 维持与云端中继的持久化 WebSocket 长连接 (毫秒级双向即时推流)
func (s *Syncer) runWebSocketLoop(ctx context.Context, wsURL string, secret string) {
	// 确保路径指向 /ws/router
	if !strings.Contains(wsURL, "/ws") {
		wsURL = strings.TrimRight(wsURL, "/") + "/ws/router"
	}

	log.Printf("[CloudSyncer] Connecting to Cloud Relay WebSocket: %s", wsURL)
	ws, err := DialWebSocket(wsURL, secret)
	if err != nil {
		log.Printf("[CloudSyncer] WebSocket connection failed: %v, retrying in 5s...", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			return
		}
	}
	defer ws.Close()

	log.Printf("[CloudSyncer] Connected to Cloud Relay Server successfully via WebSocket!")

	// 1. 发送初始状态
	s.sendWSState(ws, secret)

	// 2. 定时上报状态与保活心跳 (每 15 秒)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.sendWSState(ws, secret); err != nil {
					return
				}
			}
		}
	}()

	// 3. 阻塞监听接收来自云端中继的即时指令
	for {
		msgStr, err := ws.ReadMessage()
		if err != nil {
			log.Printf("[CloudSyncer] WebSocket read ended: %v", err)
			break
		}

		var msg struct {
			Type    string                `json:"type"`
			Payload json.RawMessage       `json:"payload"`
		}
		if err := json.Unmarshal([]byte(msgStr), &msg); err == nil {
			if msg.Type == "command" && len(msg.Payload) > 0 {
				var cmd models.CloudCommand
				if err := json.Unmarshal(msg.Payload, &cmd); err == nil && cmd.Type != "" {
					log.Printf("[CloudSyncer] Received instant command via WebSocket: %s (%s)", cmd.ID, cmd.Type)
					s.executeCommands([]models.CloudCommand{cmd})
					// 状态变更后立即上报最新状态
					_ = s.sendWSState(ws, secret)
				}
			}
		}
	}
}

// sendWSState 通过 WebSocket 上报本机状态
func (s *Syncer) sendWSState(ws *WSClientConn, secret string) error {
	devices := s.devTracker.ScanDevices()
	activeCount := 0
	for _, d := range devices {
		if d.Online {
			activeCount++
		}
	}

	members := s.engine.GetMembers()
	settings := s.config.Data.Settings

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

	statePayload := map[string]interface{}{
		"status":     status,
		"members":    members,
		"devices":    devices,
		"categories": s.dpiMgr.GetCategories(),
		"settings":   settings,
	}

	payloadBytes, err := json.Marshal(statePayload)
	if err != nil {
		return err
	}

	msg := map[string]interface{}{
		"type":      "state_update",
		"secret":    secret,
		"timestamp": time.Now().Unix(),
		"payload":   json.RawMessage(payloadBytes),
	}

	msgBytes, _ := json.Marshal(msg)
	return ws.WriteMessage(string(msgBytes))
}

// runHTTPLoop 维持原有的 HTTP 轮询模式
func (s *Syncer) runHTTPLoop(ctx context.Context) {
	// 定期上报
	s.syncState()
	// 轮询长轮询指令
	s.pollAndExecuteCommands()
}

// syncState reports local state and handles returned pending commands
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

// pollAndExecuteCommands performs long-polling to pull pending cloud commands
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

// executeCommands dispatches commands and applies them to the local PolicyEngine
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

	// Trigger immediate rule re-evaluation and kernel application
	s.engine.EvaluateAndApply(time.Now())
}
