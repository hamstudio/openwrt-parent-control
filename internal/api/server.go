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
	"parentcontrol/internal/tz"
)

// Server manages unified API and Web services
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

// NewServer creates a new API server instance
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

// Start registers routes and begins listening for HTTP requests
func (s *Server) Start(port int) error {
	mux := http.NewServeMux()

	// 1. Authentication and public routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/auth/login", s.handleAuthLogin)

	// 2. Core protected API routes
	mux.HandleFunc("/api/devices", s.requireAuth(s.handleDevices))
	mux.HandleFunc("/api/devices/", s.requireAuth(s.handleDeviceActions))
	mux.HandleFunc("/api/members", s.requireAuth(s.handleMembers))
	mux.HandleFunc("/api/members/", s.requireAuth(s.handleMemberActions))
	mux.HandleFunc("/api/apps", s.requireAuth(s.handleApps))
	mux.HandleFunc("/api/apps/", s.requireAuth(s.handleAppActions))
	mux.HandleFunc("/api/categories", s.requireAuth(s.handleCategories))
	mux.HandleFunc("/api/settings", s.requireAuth(s.handleSettings))

	// 3. Static files and frontend Web UI
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

	// Global middleware: support iframe embedding and cross-origin debugging
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Pin-Code, X-Router-Secret")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("[API] ParentControl HTTP dashboard listening on http://%s", addr)

	// Smart TLS configuration and HTTPS listener (port + 1, e.g. 8089)
	tlsConfig, err := LoadOrCreateTLSConfig()
	if err == nil && tlsConfig != nil {
		httpsPort := port + 1
		httpsAddr := fmt.Sprintf("0.0.0.0:%d", httpsPort)
		httpsServer := &http.Server{
			Addr:      httpsAddr,
			Handler:   handler,
			TLSConfig: tlsConfig,
		}
		log.Printf("[API] ParentControl HTTPS dashboard listening on https://%s", httpsAddr)
		go func() {
			if err := httpsServer.ListenAndServeTLS("", ""); err != nil {
				log.Printf("[API] HTTPS server error: %v", err)
			}
		}()
	}

	return http.ListenAndServe(addr, handler)
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// requireAuth PIN code validation middleware
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pinRequired := s.config.Data.Settings.PinCode
		if pinRequired != "" {
			// Extract from Header, Bearer token, or Query parameter
			pin := r.Header.Get("X-Pin-Code")
			if pin == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					pin = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}
			if pin == "" {
				pin = r.URL.Query().Get("pin")
			}

			if pin != pinRequired {
				s.jsonResponse(w, http.StatusUnauthorized, map[string]string{
					"error": "PIN code required or invalid",
				})
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Pin string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	configuredPin := s.config.Data.Settings.PinCode
	if configuredPin == "" || req.Pin == configuredPin {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"token":   configuredPin,
		})
	} else {
		s.jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   "Incorrect PIN",
		})
	}
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

	zoneName, offset := tz.GetTimezoneInfo()
	status := models.SystemStatus{
		Running:           true,
		UptimeSeconds:     int64(time.Since(s.startTime).Seconds()),
		TotalDevices:      len(devices),
		ActiveDevices:     activeCount,
		ManagedMembers:    len(members),
		KernelDPIReady:    s.dpiMgr.IsReady(),
		AppCount:          len(s.dpiMgr.GetCategories()),
		PinRequired:       s.config.Data.Settings.PinCode != "",
		ServerTime:        tz.Now(),
		TimezoneName:      zoneName,
		TimezoneOffset:    offset,
	}
	s.jsonResponse(w, http.StatusOK, status)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	devices := s.devTracker.ScanDevices()
	members := s.engine.GetMembers()
	settings := s.engine.GetSettings()

	// Build MAC -> Member mapping and blacklist set
	macToMember := make(map[string]*models.Member)
	for _, m := range members {
		for _, mac := range m.DeviceMACs {
			macToMember[strings.ToLower(mac)] = &m
		}
	}

	blacklistedMACs := make(map[string]bool)
	for _, mac := range settings.CustomBlacklist {
		blacklistedMACs[strings.ToLower(mac)] = true
	}

	for i := range devices {
		normMAC := strings.ToLower(devices[i].MAC)
		if m, ok := macToMember[normMAC]; ok {
			devices[i].MemberID = m.ID
			if m.IsLocked {
				devices[i].IsLocked = true
			}
		}
		if blacklistedMACs[normMAC] {
			devices[i].IsLocked = true
		}
	}

	s.jsonResponse(w, http.StatusOK, devices)
}

func (s *Server) handleDeviceActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Device MAC required"})
		return
	}

	mac := parts[0]

	// 1. POST /api/devices/:mac/lock (One-click block single device)
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "lock" {
		s.engine.LockDevice(mac)
		s.config.Data.Settings = s.engine.GetSettings()
		s.config.Data.Members = s.engine.GetMembers()
		_ = s.config.Save()
		s.engine.EvaluateAndApply(time.Now())
		s.jsonResponse(w, http.StatusOK, map[string]string{"status": "locked", "mac": mac})
		return
	}

	// 2. POST /api/devices/:mac/unlock (Restore internet access for single device)
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "unlock" {
		s.engine.UnlockDevice(mac)
		s.config.Data.Settings = s.engine.GetSettings()
		s.config.Data.Members = s.engine.GetMembers()
		_ = s.config.Save()
		s.engine.EvaluateAndApply(time.Now())
		s.jsonResponse(w, http.StatusOK, map[string]string{"status": "unlocked", "mac": mac})
		return
	}

	// 3. POST /api/devices/:mac/assign (Assign device to member or unbind)
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "assign" {
		var req struct {
			MemberID string `json:"member_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		s.engine.AssignDeviceToMember(mac, req.MemberID)
		s.config.Data.Members = s.engine.GetMembers()
		_ = s.config.Save()
		s.engine.EvaluateAndApply(time.Now())
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"status":    "assigned",
			"mac":       mac,
			"member_id": req.MemberID,
		})
		return
	}

	s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
}

func (s *Server) handleApps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		categories := s.dpiMgr.GetCategories()
		s.jsonResponse(w, http.StatusOK, categories)
	case http.MethodPost:
		var app models.AppInfo
		if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid payload"})
			return
		}
		added, err := s.dpiMgr.AddApp(app)
		if err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.jsonResponse(w, http.StatusOK, added)
	default:
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func (s *Server) handleAppActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/apps/")
	appID, err := strconv.Atoi(path)
	if err != nil {
		s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid app ID"})
		return
	}

	if r.Method == http.MethodDelete {
		if err := s.dpiMgr.DeleteApp(appID); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}
	s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		categories := s.dpiMgr.GetCategories()
		s.jsonResponse(w, http.StatusOK, categories)
	case http.MethodPost:
		var cat models.AppCategory
		if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid payload"})
			return
		}
		added, err := s.dpiMgr.AddCategory(cat)
		if err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.jsonResponse(w, http.StatusOK, added)
	default:
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
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

		// Persist changes
		s.config.Data.Members = s.engine.GetMembers()
		_ = s.config.Save()

		// Trigger immediate rule evaluation
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
		s.config.Data.Members = s.engine.GetMembers()
		s.config.Data.Settings = s.engine.GetSettings()
		_ = s.config.Save()
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
		s.config.Data.Members = s.engine.GetMembers()
		s.config.Data.Settings = s.engine.GetSettings()
		_ = s.config.Save()
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
		s.config.Data.Members = s.engine.GetMembers()
		s.config.Data.Settings = s.engine.GetSettings()
		_ = s.config.Save()
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

		// Re-apply DNS and SafeSearch rules
		_ = s.dnsMgr.ApplyConfig(req.EnforceSafeSearch, true, req.CustomBlacklist, req.CustomWhitelist)

		// Trigger immediate rule evaluation
		s.engine.EvaluateAndApply(time.Now())

		s.jsonResponse(w, http.StatusOK, req)
	default:
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}
