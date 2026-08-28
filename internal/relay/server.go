package relay

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Server 云端中继服务器
type Server struct {
	mq         *MessageQueue
	authSecret string // 可选全局 Secret，若为空则支持多租户基于客户端传入的 Secret 路由
}

// NewServer 创建中继服务器
func NewServer(authSecret string) *Server {
	return &Server{
		mq:         NewMessageQueue(),
		authSecret: authSecret,
	}
}

// Start 启动中继 HTTP & WebSocket 服务
func (s *Server) Start(port int) error {
	mux := http.NewServeMux()

	// 1. WebSocket 路由
	mux.HandleFunc("/ws/router", s.handleRouterWS)
	mux.HandleFunc("/ws/client", s.handleClientWS)

	// 2. HTTP REST 路由 (与 CF Worker 100% 格式对齐)
	mux.HandleFunc("/api/router/sync", s.handleRouterSyncHTTP)
	mux.HandleFunc("/api/client/status", s.handleClientStatusHTTP)
	mux.HandleFunc("/api/client/command", s.handleClientCommandHTTP)
	mux.HandleFunc("/health", s.handleHealth)

	// 全局中间件（CORS & 日志）
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Router-Secret")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("[RelayServer] ParentControl Cloud Relay Server running on http://%s", addr)
	return http.ListenAndServe(addr, handler)
}

// extractSecret 提取通信鉴权 Secret
func (s *Server) extractSecret(r *http.Request) string {
	secret := r.Header.Get("X-Router-Secret")
	if secret == "" {
		secret = r.URL.Query().Get("secret")
	}
	if secret == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			secret = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if secret == "" && s.authSecret != "" {
		return ""
	}
	if s.authSecret != "" && secret != s.authSecret {
		return ""
	}
	if secret == "" {
		secret = "default"
	}
	return secret
}

// handleRouterWS 路由器 WebSocket 长连接处理
func (s *Server) handleRouterWS(w http.ResponseWriter, r *http.Request) {
	secret := s.extractSecret(r)
	if secret == "" {
		http.Error(w, "Unauthorized: invalid secret", http.StatusUnauthorized)
		return
	}

	ws, err := Upgrade(w, r)
	if err != nil {
		log.Printf("[RelayServer] Router WS upgrade error: %v", err)
		return
	}
	defer ws.Close()

	log.Printf("[RelayServer] Router connected via WebSocket [Secret: %s...]", maskSecret(secret))
	pending := s.mq.RegisterRouter(secret, ws)
	defer s.mq.UnregisterRouter(secret, ws)

	// 立即下发离线期间积压的指令
	for _, cmd := range pending {
		_ = s.mq.routerConns[secret].WriteMessage(string(cmd))
	}

	// 启动心跳 Ping 定时器 (每 20 秒)
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := ws.WritePing(); err != nil {
				return
			}
		}
	}()

	// 循环接收路由器上报的数据
	for {
		msgStr, err := ws.ReadMessage()
		if err != nil {
			log.Printf("[RelayServer] Router disconnected [Secret: %s...]: %v", maskSecret(secret), err)
			break
		}

		var msg Message
		if err := json.Unmarshal([]byte(msgStr), &msg); err == nil {
			if msg.Type == "state_update" && len(msg.Payload) > 0 {
				s.mq.UpdateState(secret, msg.Payload)
			}
		}
	}
}

// handleClientWS 手机 App / Web 控制端 WebSocket 长连接
func (s *Server) handleClientWS(w http.ResponseWriter, r *http.Request) {
	secret := s.extractSecret(r)
	if secret == "" {
		http.Error(w, "Unauthorized: invalid secret", http.StatusUnauthorized)
		return
	}

	ws, err := Upgrade(w, r)
	if err != nil {
		log.Printf("[RelayServer] Client WS upgrade error: %v", err)
		return
	}
	defer ws.Close()

	log.Printf("[RelayServer] Client App connected via WebSocket [Secret: %s...]", maskSecret(secret))
	s.mq.RegisterClient(secret, ws)
	defer s.mq.UnregisterClient(secret, ws)

	// 连接建立时立即推送一次路由器最新状态
	if state := s.mq.GetLatestState(secret); state != nil && len(state.Data) > 0 {
		initialMsg := Message{
			Type:      "state_update",
			Secret:    secret,
			Timestamp: time.Now().Unix(),
			Payload:   state.Data,
		}
		initialBytes, _ := json.Marshal(initialMsg)
		_ = ws.WriteMessage(string(initialBytes))
	}

	for {
		msgStr, err := ws.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal([]byte(msgStr), &msg); err == nil {
			if msg.Type == "command" && len(msg.Payload) > 0 {
				log.Printf("[RelayServer] Client dispatched command to router [Secret: %s...]", maskSecret(secret))
				_, _ = s.mq.DispatchCommand(secret, msg.Payload)
			}
		}
	}
}

// handleRouterSyncHTTP 供路由器 HTTP 轮询同步与拉取指令
func (s *Server) handleRouterSyncHTTP(w http.ResponseWriter, r *http.Request) {
	secret := s.extractSecret(r)
	if secret == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if len(body) > 0 {
		s.mq.UpdateState(secret, body)
	}

	// 消费离线指令并返回
	s.mq.mu.Lock()
	pending := s.mq.pendingCmds[secret]
	s.mq.pendingCmds[secret] = nil
	s.mq.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":             0,
		"pending_commands": pending,
		"server_time":      time.Now().Unix(),
	})
}

// handleClientStatusHTTP 供客户端通过 REST API 查询最新状态
func (s *Server) handleClientStatusHTTP(w http.ResponseWriter, r *http.Request) {
	secret := s.extractSecret(r)
	if secret == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	state := s.mq.GetLatestState(secret)
	w.Header().Set("Content-Type", "application/json")
	if state == nil || len(state.Data) == 0 {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"online":  false,
			"message": "Router has not synced state yet",
		})
		return
	}

	var stateObj interface{}
	_ = json.Unmarshal(state.Data, &stateObj)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"online":    state.Online,
		"last_seen": state.LastSeen.Format(time.RFC3339),
		"state":     stateObj,
	})
}

// handleClientCommandHTTP 供客户端通过 REST API 下发指令
func (s *Server) handleClientCommandHTTP(w http.ResponseWriter, r *http.Request) {
	secret := s.extractSecret(r)
	if secret == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Invalid command payload", http.StatusBadRequest)
		return
	}

	delivered, err := s.mq.DispatchCommand(secret, body)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":      0,
		"delivered": delivered, // true: 实时推达, false: 进入待收队列
		"message":   "Command accepted",
	})
}

// handleHealth 探针
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
		"service": "ParentControl-RelayServer",
		"time":    time.Now().Unix(),
	})
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return "***"
	}
	return s[:3] + "***"
}
