package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"parentcontrol/internal/dpi"
	"parentcontrol/internal/models"
)

func TestStatsTracker_RecordAndOverview(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parentcontrol_stats_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	statsFile := filepath.Join(tmpDir, "stats.json")
	dpiMgr := dpi.NewDPIManager("")
	tracker := NewStatsTracker(statsFile, dpiMgr)

	mac1 := "00:11:22:33:44:55"
	mac2 := "AA:BB:CC:DD:EE:FF"

	now := time.Date(2026, 8, 28, 14, 30, 0, 0, time.Local)

	// Record minute activity
	tracker.RecordMinuteActivity(mac1, "192.168.1.100", "iPhone-Child", "m_1", 1024*1024, now)
	tracker.RecordMinuteActivity(mac1, "192.168.1.100", "iPhone-Child", "m_1", 2048*1024, now)

	// Record app activities
	// 2001 is 王者荣耀 (game)
	tracker.RecordAppActivity(mac1, 2001, 1024*1024, now)
	// 3001 is 抖音 (video)
	tracker.RecordAppActivity(mac1, 3001, 2048*1024, now)

	// Verify device used minutes
	usedMin := tracker.GetDeviceUsedMinutesToday(mac1)
	if usedMin != 2 {
		t.Errorf("Expected 2 used minutes for mac1, got %d", usedMin)
	}

	devices := []*models.Device{
		{MAC: mac1, IP: "192.168.1.100", Hostname: "iPhone-Child", MemberID: "m_1", Online: true, TotalBytes: 3145728},
		{MAC: mac2, IP: "192.168.1.101", Hostname: "iPad", MemberID: "", Online: false, TotalBytes: 0},
	}
	members := []models.Member{
		{ID: "m_1", Name: "Tom"},
	}

	overview := tracker.GetOverview(devices, members)
	if overview.TotalOnlineMinutes != 2 {
		t.Errorf("Expected TotalOnlineMinutes=2, got %d", overview.TotalOnlineMinutes)
	}
	if len(overview.DeviceRankings) != 2 {
		t.Errorf("Expected 2 devices in rankings, got %d", len(overview.DeviceRankings))
	}
	if len(overview.CategoryBreakdown) != 2 {
		t.Errorf("Expected 2 categories in breakdown (game & video), got %d", len(overview.CategoryBreakdown))
	}

	// Verify detail
	detail := tracker.GetDeviceStatsDetail(mac1, 7, devices[0], &members[0])
	if detail.HourlyActivity[14] != 2 {
		t.Errorf("Expected 2 minutes at hour 14, got %d", detail.HourlyActivity[14])
	}
	if len(detail.TopApps) != 2 {
		t.Errorf("Expected 2 top apps, got %d", len(detail.TopApps))
	}
	if len(detail.History) != 7 {
		t.Errorf("Expected 7 days in history, got %d", len(detail.History))
	}

	// Test Save and Reload
	if err := tracker.Save(); err != nil {
		t.Fatalf("Failed to save stats: %v", err)
	}

	tracker2 := NewStatsTracker(statsFile, dpiMgr)
	if tracker2.GetDeviceUsedMinutesToday(mac1) != 2 {
		t.Errorf("Expected 2 minutes after reload, got %d", tracker2.GetDeviceUsedMinutesToday(mac1))
	}
}

func TestStatsTracker_DailyRollover(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "parentcontrol_stats_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	statsFile := filepath.Join(tmpDir, "stats.json")
	tracker := NewStatsTracker(statsFile, nil)

	mac := "00:11:22:33:44:55"
	day1 := time.Date(2026, 8, 28, 10, 0, 0, 0, time.Local)
	tracker.RecordMinuteActivity(mac, "192.168.1.100", "Device-1", "m_1", 1000, day1)

	if tracker.GetDeviceUsedMinutesToday(mac) != 1 {
		t.Errorf("Expected 1 minute on day1, got %d", tracker.GetDeviceUsedMinutesToday(mac))
	}

	// Move to next day
	day2 := time.Date(2026, 8, 29, 0, 1, 0, 0, time.Local)
	tracker.CheckDailyRollover(day2, 0)

	// After rollover, today's minutes should be 0
	if tracker.GetDeviceUsedMinutesToday(mac) != 0 {
		t.Errorf("Expected 0 minutes after rollover on day2, got %d", tracker.GetDeviceUsedMinutesToday(mac))
	}

	// Detail history should contain 8-28 with 1 min
	detail := tracker.GetDeviceStatsDetail(mac, 7, nil, nil)
	foundPrev := false
	for _, h := range detail.History {
		if h.Date == "2026-08-28" && h.UsedMinutes == 1 {
			foundPrev = true
			break
		}
	}
	if !foundPrev {
		t.Errorf("Expected to find 2026-08-28 with 1 min in historical record")
	}
}
