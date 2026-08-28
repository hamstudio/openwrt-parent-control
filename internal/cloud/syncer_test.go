package cloud

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"parentcontrol/internal/config"
	"parentcontrol/internal/device"
	"parentcontrol/internal/dpi"
	"parentcontrol/internal/firewall"
	"parentcontrol/internal/models"
	"parentcontrol/internal/quota"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCloudSyncerExecution(t *testing.T) {
	var receivedSync bool

	mockTransport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/router/sync" {
			receivedSync = true
			cmd := models.CloudCommand{
				ID:        "cmd_test_1",
				Type:      "LOCK",
				MemberID:  "m_test",
				CreatedAt: time.Now().UnixMilli(),
			}
			resBody, _ := json.Marshal(map[string]interface{}{
				"success":  true,
				"commands": []models.CloudCommand{cmd},
			})
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(resBody)),
				Header:     make(http.Header),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
			Header:     make(http.Header),
		}, nil
	})

	// 2. Initialize core components
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.json")
	featFile := filepath.Join(tmpDir, "feature.cfg")
	_ = os.WriteFile(featFile, []byte("#version 1.0\n#class game 1 Games\n101 GameApp: [tcp]\n"), 0644)

	cfgStore := config.NewConfigStore(cfgFile)
	cfgStore.Data.Settings.CloudSyncEnabled = true
	cfgStore.Data.Settings.CloudWorkerURL = "https://mock.worker.dev"
	cfgStore.Data.Members = []models.Member{
		{
			ID:       "m_test",
			Name:     "Test Member",
			Enabled:  true,
			IsLocked: false,
		},
	}
	_ = cfgStore.Save()

	dpiMgr := dpi.NewDPIManager(featFile)
	fwMgr := firewall.NewFirewallManager()
	devTracker := device.NewDeviceTracker()
	engine := quota.NewPolicyEngine(fwMgr, dpiMgr, devTracker)
	engine.UpdateSettings(cfgStore.Data.Settings)
	for _, m := range cfgStore.Data.Members {
		engine.SetMember(m)
	}

	// 3. Execute single sync
	syncer := NewSyncer(engine, devTracker, dpiMgr, cfgStore)
	syncer.SetHTTPTransport(mockTransport)
	syncer.syncState()

	if !receivedSync {
		t.Fatalf("expected state to be synced to mock transport")
	}

	// 4. Verify member was successfully locked by LOCK command
	members := engine.GetMembers()
	var targetMember *models.Member
	for _, m := range members {
		if m.ID == "m_test" {
			targetMember = &m
			break
		}
	}

	if targetMember == nil {
		t.Fatalf("expected member m_test to exist")
	}
	if !targetMember.IsLocked {
		t.Errorf("expected member m_test to be locked by cloud command")
	}
}
