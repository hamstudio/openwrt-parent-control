package relay

import (
	"encoding/json"
	"sync"
	"time"
)

// Message 统一中继消息载荷
type Message struct {
	Type      string          `json:"type"` // "state_update", "command", "ack", "ping", "pong"
	DeviceID  string          `json:"device_id,omitempty"`
	Secret    string          `json:"secret,omitempty"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// RouterState 路由器最新状态缓存
type RouterState struct {
	Secret      string          `json:"secret"`
	LastSeen    time.Time       `json:"last_seen"`
	Data        json.RawMessage `json:"data"`
	Online      bool            `json:"online"`
}

// MessageQueue 内存 PubSub 消息总线
type MessageQueue struct {
	mu            sync.RWMutex
	routerConns   map[string]*WSConn             // secret -> router websocket connection
	clientConns   map[string]map[*WSConn]bool     // secret -> set of app clients
	states        map[string]*RouterState         // secret -> latest state
	pendingCmds   map[string][]json.RawMessage    // secret -> offline queued commands
}

// NewMessageQueue 创建消息总线
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		routerConns: make(map[string]*WSConn),
		clientConns: make(map[string]map[*WSConn]bool),
		states:      make(map[string]*RouterState),
		pendingCmds: make(map[string][]json.RawMessage),
	}
}

// RegisterRouter 注册路由器连接
func (mq *MessageQueue) RegisterRouter(secret string, ws *WSConn) []json.RawMessage {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	// 关闭旧连接
	if old, exists := mq.routerConns[secret]; exists && old != ws {
		_ = old.Close()
	}
	mq.routerConns[secret] = ws

	if state, ok := mq.states[secret]; ok {
		state.Online = true
		state.LastSeen = time.Now()
	} else {
		mq.states[secret] = &RouterState{
			Secret:   secret,
			LastSeen: time.Now(),
			Online:   true,
		}
	}

	// 消费离线指令
	pending := mq.pendingCmds[secret]
	mq.pendingCmds[secret] = nil
	return pending
}

// UnregisterRouter 取消路由器注册
func (mq *MessageQueue) UnregisterRouter(secret string, ws *WSConn) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if cur, ok := mq.routerConns[secret]; ok && cur == ws {
		delete(mq.routerConns, secret)
		if state, ok := mq.states[secret]; ok {
			state.Online = false
		}
	}
}

// RegisterClient 注册手机 App 客户端连接
func (mq *MessageQueue) RegisterClient(secret string, ws *WSConn) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if _, ok := mq.clientConns[secret]; !ok {
		mq.clientConns[secret] = make(map[*WSConn]bool)
	}
	mq.clientConns[secret][ws] = true
}

// UnregisterClient 移除手机 App 客户端连接
func (mq *MessageQueue) UnregisterClient(secret string, ws *WSConn) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if clients, ok := mq.clientConns[secret]; ok {
		delete(clients, ws)
		if len(clients) == 0 {
			delete(mq.clientConns, secret)
		}
	}
}

// UpdateState 更新并广播路由器状态
func (mq *MessageQueue) UpdateState(secret string, stateData json.RawMessage) {
	mq.mu.Lock()
	state, ok := mq.states[secret]
	if !ok {
		state = &RouterState{Secret: secret}
		mq.states[secret] = state
	}
	state.Data = stateData
	state.LastSeen = time.Now()
	state.Online = true

	// 复制客户端列表
	var targets []*WSConn
	if clients, ok := mq.clientConns[secret]; ok {
		for c := range clients {
			targets = append(targets, c)
		}
	}
	mq.mu.Unlock()

	// 异步向所有订阅的 App 广播
	msg := Message{
		Type:      "state_update",
		Secret:    secret,
		Timestamp: time.Now().Unix(),
		Payload:   stateData,
	}
	msgBytes, _ := json.Marshal(msg)
	msgStr := string(msgBytes)

	for _, ws := range targets {
		go func(conn *WSConn) {
			_ = conn.WriteMessage(msgStr)
		}(ws)
	}
}

// GetLatestState 获取最新的路由器状态缓存
func (mq *MessageQueue) GetLatestState(secret string) *RouterState {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	if state, ok := mq.states[secret]; ok {
		return &RouterState{
			Secret:   state.Secret,
			LastSeen: state.LastSeen,
			Data:     state.Data,
			Online:   state.Online,
		}
	}
	return nil
}

// DispatchCommand 将指令下发给路由器（在线直发，离线进入缓冲队列）
func (mq *MessageQueue) DispatchCommand(secret string, cmdPayload json.RawMessage) (bool, error) {
	mq.mu.Lock()
	routerWS, online := mq.routerConns[secret]
	if !online {
		// 暂存离线指令队列 (最多暂存 50 条)
		if len(mq.pendingCmds[secret]) < 50 {
			mq.pendingCmds[secret] = append(mq.pendingCmds[secret], cmdPayload)
		}
		mq.mu.Unlock()
		return false, nil
	}
	mq.mu.Unlock()

	msg := Message{
		Type:      "command",
		Secret:    secret,
		Timestamp: time.Now().Unix(),
		Payload:   cmdPayload,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return false, err
	}

	err = routerWS.WriteMessage(string(msgBytes))
	if err != nil {
		mq.UnregisterRouter(secret, routerWS)
		// 发送失败放入重试队列
		mq.mu.Lock()
		mq.pendingCmds[secret] = append(mq.pendingCmds[secret], cmdPayload)
		mq.mu.Unlock()
		return false, err
	}

	return true, nil
}
