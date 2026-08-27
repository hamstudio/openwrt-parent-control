package quota

import (
	"testing"
	"time"

	"parentcontrol/internal/models"
)

func TestIsTimeInRange(t *testing.T) {
	// Same-day range: 12:00 to 14:00
	if !isTimeInRange("12:30", "12:00", "14:00") {
		t.Errorf("expected 12:30 to be in 12:00-14:00")
	}
	if isTimeInRange("11:59", "12:00", "14:00") {
		t.Errorf("expected 11:59 NOT to be in 12:00-14:00")
	}
	if isTimeInRange("14:01", "12:00", "14:00") {
		t.Errorf("expected 14:01 NOT to be in 12:00-14:00")
	}

	// Overnight range: 21:30 to 07:00
	if !isTimeInRange("22:00", "21:30", "07:00") {
		t.Errorf("expected 22:00 to be in 21:30-07:00")
	}
	if !isTimeInRange("06:30", "21:30", "07:00") {
		t.Errorf("expected 06:30 to be in 21:30-07:00")
	}
	if isTimeInRange("12:00", "21:30", "07:00") {
		t.Errorf("expected 12:00 NOT to be in 21:30-07:00")
	}
}

func TestShouldBlockMember(t *testing.T) {
	pe := &PolicyEngine{
		members: make(map[string]*models.Member),
	}

	testTime, _ := time.Parse("2006-01-02 15:04:05", "2026-08-27 22:00:00") // Thursday, weekday 4

	// Case 1: Member locked
	mLocked := &models.Member{
		ID:       "m1",
		IsLocked: true,
	}
	if !pe.shouldBlockMember(mLocked, testTime) {
		t.Errorf("expected locked member to be blocked")
	}

	// Case 2: Multi-time-range Block mode
	mBlockSchedule := &models.Member{
		ID:      "m2",
		Enabled: true,
		Schedule: models.ScheduleRule{
			Enabled: true,
			Action:  "block",
			Days:    []int{1, 2, 3, 4, 5}, // weekdays
			TimeRanges: []models.TimeRange{
				{StartTime: "12:00", EndTime: "14:00"},
				{StartTime: "21:30", EndTime: "07:00"},
			},
		},
	}
	// At 22:00 on Thursday -> should block
	if !pe.shouldBlockMember(mBlockSchedule, testTime) {
		t.Errorf("expected member in night block range to be blocked")
	}

	// At 15:00 on Thursday -> should NOT block
	afternoonTime, _ := time.Parse("2006-01-02 15:04:05", "2026-08-27 15:00:00")
	if pe.shouldBlockMember(mBlockSchedule, afternoonTime) {
		t.Errorf("expected member at 15:00 to be allowed")
	}

	// At 13:00 on Thursday -> should block (noon slot)
	noonTime, _ := time.Parse("2006-01-02 15:04:05", "2026-08-27 13:00:00")
	if !pe.shouldBlockMember(mBlockSchedule, noonTime) {
		t.Errorf("expected member in noon slot to be blocked")
	}

	// Case 3: Allow mode (only allow in ranges)
	mAllowSchedule := &models.Member{
		ID:      "m3",
		Enabled: true,
		Schedule: models.ScheduleRule{
			Enabled: true,
			Action:  "allow",
			Days:    []int{1, 2, 3, 4, 5},
			TimeRanges: []models.TimeRange{
				{StartTime: "18:00", EndTime: "20:00"},
			},
		},
	}
	// At 15:00 -> not in allow range -> should block
	if !pe.shouldBlockMember(mAllowSchedule, afternoonTime) {
		t.Errorf("expected member outside allow range to be blocked")
	}

	// At 19:00 -> in allow range -> should NOT block
	eveningTime, _ := time.Parse("2006-01-02 15:04:05", "2026-08-27 19:00:00")
	if pe.shouldBlockMember(mAllowSchedule, eveningTime) {
		t.Errorf("expected member in allow range to be permitted")
	}

	// Case 4: Bonus exemption overrides block schedule
	bonusExpiry := eveningTime.Add(30 * time.Minute)
	mBlockSchedule.BonusUntil = &bonusExpiry
	if pe.shouldBlockMember(mBlockSchedule, testTime.Add(10*time.Minute)) {
		// Wait, testTime was 22:00, let's make bonus valid for testTime
		bonusForNight := testTime.Add(30 * time.Minute)
		mBlockSchedule.BonusUntil = &bonusForNight
		if pe.shouldBlockMember(mBlockSchedule, testTime) {
			t.Errorf("expected bonus time to override block schedule")
		}
	}
}
