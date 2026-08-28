package relay

import (
	"encoding/json"
	"testing"
)

func TestMessageQueue_StateAndCommand(t *testing.T) {
	mq := NewMessageQueue()
	secret := "test-secret-123"

	// 1. 测试状态更新
	testState := map[string]interface{}{
		"running": true,
		"devices": 5,
	}
	stateBytes, _ := json.Marshal(testState)
	mq.UpdateState(secret, stateBytes)

	cached := mq.GetLatestState(secret)
	if cached == nil {
		t.Fatalf("expected cached state, got nil")
	}
	if !cached.Online {
		t.Errorf("expected online true, got false")
	}

	// 2. 测试离线指令队列
	cmdPayload := json.RawMessage(`{"action":"lock","mac":"AA:BB:CC:DD:EE:FF"}`)
	delivered, err := mq.DispatchCommand(secret, cmdPayload)
	if err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}
	if delivered {
		t.Errorf("expected offline queueing (delivered=false), got true")
	}

	// 3. 测试路由器上线消费离线指令
	pending := mq.RegisterRouter(secret, &WSConn{})
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending command, got %d", len(pending))
	}
	if string(pending[0]) != string(cmdPayload) {
		t.Errorf("expected cmd %s, got %s", string(cmdPayload), string(pending[0]))
	}

	// 4. 再次获取离线队列应已清空
	if len(mq.pendingCmds[secret]) != 0 {
		t.Errorf("expected pending cmds empty after registration")
	}
}
