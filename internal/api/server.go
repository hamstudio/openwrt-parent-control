package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"parentcontrol/internal/config"
	"parentcontrol/internal/device"
	"parentcontrol/internal/dpi"
	"parentcontrol/internal/firewall"
	"parentcontrol/internal/models"
	"parentcontrol/internal/quota"
	"parentcontrol/internal/safedns"
)

// Server 统一 API 与 Web 服务
type Server struct {
	engine     *quota.PolicyEngine
	dpiMgr     *dpi.DPIManager
	fwMgr      *firewall.FirewallManager
	dnsMgr     *safedns.SafeDNSManager
	devTracker *device.DeviceTracker
	config     *config.ConfigStore
	staticFS   embed.FS
	startTime  time.Time
}

// NewServer 创建 API 服务实例
func NewServer(
	engine *quota.PolicyEngine,
	dpiMgr *dpi.DPIManager,
	fwMgr *firewall.FirewallManager,
	dnsMgr *safedns.SafeDNSManager,
	devTracker *device.DeviceTracker,
	config *config.ConfigStore,
	staticFS embed.FS,
) *Server {
	return &Server{
		engine:     engine,
		dpiMgr:     dpiMgr,
		fwMgr:      fwMgr,
		dnsMgr:     dnsMgr,
		devTracker: devTracker,
		config:     config,
		staticFS:   staticFS,
		startTime:  time.Now(),
	}
}

// Start 注册路由并启动 HTTP 监听
func (s *Server) Start(port int) error {
	mux := http.NewServeMux()

	// 1. API 路由
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/devices", s.handleDevices)
	mux.HandleFunc("/api/members", s.handleMembers)
	mux.HandleFunc("/api/members/", s.handleMemberActions)
	mux.HandleFunc("/api/apps", s.handleApps)
	mux.HandleFunc("/api/apps/", s.handleAppActions)
	mux.HandleFunc("/api/categories", s.handleCategories)
	mux.HandleFunc("/api/categories/", s.handleCategoryActions)
	mux.HandleFunc("/api/settings", s.handleSettings)

	// 2. 静态文件与前端 WebUI
	staticSub, err := fs.Sub(s.staticFS, "static")
	if err != nil {
		log.Printf("[API] Failed to sub static FS: %v", err)
	} else {
		fileServer := http.FileServer(http.FS(staticSub))
		mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			data, err := fs.ReadFile(staticSub, "index.html")
			if err != nil {
				http.Error(w, "Index not found", 404)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
		})
	}

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("[API] ParentControl Web dashboard and API listening on http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
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
		UptimeSeconds:  int64(time.Since(s.startTime).Seconds()),
		TotalDevices:   len(devices),
		ActiveDevices:  activeCount,
		ManagedMembers: len(members),
		KernelDPIReady: s.dpiMgr.IsReady(),
		AppCount:       len(s.dpiMgr.GetAllApps()),
		ServerTime:     time.Now(),
	}
	s.jsonResponse(w, http.StatusOK, status)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	devices := s.devTracker.ScanDevices()
	s.jsonResponse(w, http.StatusOK, devices)
}

func (s *Server) handleApps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		categories := s.dpiMgr.GetCategories()
		s.jsonResponse(w, http.StatusOK, categories)
	case http.MethodPost:
		var app models.AppInfo
		if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid app payload"})
			return
		}
		created, err := s.dpiMgr.AddApp(app)
		if err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		// 持久化自定义数据
		customApps, customCats := s.dpiMgr.GetCustomData()
		s.config.Data.CustomApps = customApps
		s.config.Data.CustomCategories = customCats
		_ = s.config.Save()

		s.jsonResponse(w, http.StatusCreated, created)
	default:
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func (s *Server) handleAppActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/apps/")
	if path == "" {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "App ID required"})
		return
	}
	appID, err := strconv.Atoi(path)
	if err != nil {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid app ID"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		app, ok := s.dpiMgr.GetApp(appID)
		if !ok {
			s.jsonResponse(w, http.StatusNotFound, map[string]string{"error": "App not found"})
			return
		}
		s.jsonResponse(w, http.StatusOK, app)
	case http.MethodPut, http.MethodPost:
		var app models.AppInfo
		if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid app payload"})
			return
		}
		updated, err := s.dpiMgr.UpdateApp(appID, app)
		if err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		customApps, customCats := s.dpiMgr.GetCustomData()
		s.config.Data.CustomApps = customApps
		s.config.Data.CustomCategories = customCats
		_ = s.config.Save()

		s.jsonResponse(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := s.dpiMgr.DeleteApp(appID); err != nil {
			s.jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}

		// 同步从成员封禁列表中剔除被删除的 App ID
		members := s.engine.GetMembers()
		for _, m := range members {
			newBlocked := make([]int, 0, len(m.BlockedAppIDs))
			changed := false
			for _, bid := range m.BlockedAppIDs {
				if bid == appID {
					changed = true
				} else {
					newBlocked = append(newBlocked, bid)
				}
			}
			if changed {
				m.BlockedAppIDs = newBlocked
				s.engine.SetMember(m)
			}
		}

		customApps, customCats := s.dpiMgr.GetCustomData()
		s.config.Data.CustomApps = customApps
		s.config.Data.CustomCategories = customCats
		s.config.Data.Members = s.engine.GetMembers()
		_ = s.config.Save()

		s.engine.EvaluateAndApply(time.Now())
		s.jsonResponse(w, http.StatusOK, map[string]string{"result": "deleted", "id": strconv.Itoa(appID)})
	default:
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.jsonResponse(w, http.StatusOK, s.dpiMgr.GetCategories())
	case http.MethodPost:
		var cat models.AppCategory
		if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid category payload"})
			return
		}
		created, err := s.dpiMgr.AddCategory(cat)
		if err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		customApps, customCats := s.dpiMgr.GetCustomData()
		s.config.Data.CustomApps = customApps
		s.config.Data.CustomCategories = customCats
		_ = s.config.Save()

		s.jsonResponse(w, http.StatusCreated, created)
	default:
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func (s *Server) handleCategoryActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/categories/")
	if path == "" {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Category ID required"})
		return
	}
	catID, err := strconv.Atoi(path)
	if err != nil {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid category ID"})
		return
	}

	if r.Method == http.MethodDelete {
		if err := s.dpiMgr.DeleteCategory(catID); err != nil {
			s.jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		customApps, customCats := s.dpiMgr.GetCustomData()
		s.config.Data.CustomApps = customApps
		s.config.Data.CustomCategories = customCats
		_ = s.config.Save()

		s.jsonResponse(w, http.StatusOK, map[string]string{"result": "deleted", "id": strconv.Itoa(catID)})
		return
	}

	s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
}

func (s *Server) handleMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		members := s.engine.GetMembers()
		s.jsonResponse(w, http.StatusOK, members)
	case http.MethodPost:
		var m models.Member
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid payload"})
			return
		}
		if m.ID == "" {
			m.ID = fmt.Sprintf("m_%d", time.Now().UnixNano())
		}
		s.engine.SetMember(m)

		// 持久化保存
		s.config.Data.Members = s.engine.GetMembers()
		_ = s.config.Save()

		// 立即触发规则同步
		s.engine.EvaluateAndApply(time.Now())

		s.jsonResponse(w, http.StatusOK, m)
	default:
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func (s *Server) handleMemberActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/members/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Member ID required"})
		return
	}

	memberID := parts[0]

	// 1. DELETE /api/members/:id
	if r.Method == http.MethodDelete && len(parts) == 1 {
		s.engine.DeleteMember(memberID)
		s.config.Data.Members = s.engine.GetMembers()
		_ = s.config.Save()
		s.engine.EvaluateAndApply(time.Now())
		s.jsonResponse(w, http.StatusOK, map[string]string{"result": "deleted"})
		return
	}

	// 2. POST /api/members/:id/lock
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "lock" {
		if err := s.engine.LockMember(memberID); err != nil {
			s.jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		s.engine.EvaluateAndApply(time.Now())
		s.jsonResponse(w, http.StatusOK, map[string]string{"status": "locked"})
		return
	}

	// 3. POST /api/members/:id/unlock
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "unlock" {
		if err := s.engine.UnlockMember(memberID); err != nil {
			s.jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		s.engine.EvaluateAndApply(time.Now())
		s.jsonResponse(w, http.StatusOK, map[string]string{"status": "unlocked"})
		return
	}

	// 4. POST /api/members/:id/bonus?minutes=30
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "bonus" {
		minutes, _ := strconv.Atoi(r.URL.Query().Get("minutes"))
		if minutes <= 0 {
			minutes = 30
		}
		if err := s.engine.BonusMember(memberID, minutes); err != nil {
			s.jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		s.engine.EvaluateAndApply(time.Now())
		s.jsonResponse(w, http.StatusOK, map[string]string{"status": "bonus_applied", "minutes": strconv.Itoa(minutes)})
		return
	}

	s.jsonResponse(w, http.StatusNotFound, map[string]string{"error": "Unknown member action"})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings := s.engine.GetSettings()
		s.jsonResponse(w, http.StatusOK, settings)
	case http.MethodPost:
		var req models.GlobalSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid payload"})
			return
		}

		s.engine.UpdateSettings(req)
		s.config.Data.Settings = req
		_ = s.config.Save()

		// 重新应用 DNS 与 SafeSearch
		_ = s.dnsMgr.ApplyConfig(req.EnforceSafeSearch, true, req.CustomBlacklist, req.CustomWhitelist)

		// 立即评估规则
		s.engine.EvaluateAndApply(time.Now())

		s.jsonResponse(w, http.StatusOK, req)
	default:
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}
