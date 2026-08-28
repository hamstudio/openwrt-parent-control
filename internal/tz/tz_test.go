package tz

import (
	"testing"
	"time"
)

func TestParsePOSIXTZ(t *testing.T) {
	tests := []struct {
		tzStr          string
		expectedOffset int
	}{
		{"CST-8", 8 * 3600},
		{"JST-9", 9 * 3600},
		{"EST5", -5 * 3600},
		{"EST5EDT", -5 * 3600},
		{"GMT0", 0},
		{"UTC", 0},
		{"Asia/Shanghai", 8 * 3600},
		{"Asia/Tokyo", 9 * 3600},
		{"Europe/London", 0}, // or BST in summer
	}

	for _, tt := range tests {
		loc, name := ParsePOSIXTZ(tt.tzStr)
		if loc == nil {
			t.Errorf("ParsePOSIXTZ(%q) returned nil location", tt.tzStr)
			continue
		}
		if name == "" {
			t.Errorf("ParsePOSIXTZ(%q) returned empty name", tt.tzStr)
		}
		// Test offset at a fixed winter timestamp (to avoid DST discrepancy during tests)
		testTime := time.Date(2026, 1, 15, 12, 0, 0, 0, loc)
		_, offset := testTime.Zone()
		if tt.expectedOffset != 0 && offset != tt.expectedOffset {
			t.Errorf("ParsePOSIXTZ(%q) offset = %d, expected %d", tt.tzStr, offset, tt.expectedOffset)
		}
	}
}

func TestNowAndGetTimezoneInfo(t *testing.T) {
	loc := DetectAndApplyTimezone()
	if loc == nil {
		t.Fatal("DetectAndApplyTimezone returned nil")
	}

	now := Now()
	if now.IsZero() {
		t.Fatal("Now() returned zero time")
	}

	name, offset := GetTimezoneInfo()
	t.Logf("Detected timezone name: %s, offset: %d seconds, local time: %s", name, offset, now.Format("2006-01-02 15:04:05 MST"))
}
