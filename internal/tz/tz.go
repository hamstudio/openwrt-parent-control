package tz

import (
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // Embed IANA tzdata
)

var (
	mu          sync.RWMutex
	currentLoc  *time.Location
	currentName string
)

func init() {
	DetectAndApplyTimezone()
}

// DetectAndApplyTimezone detects router timezone and updates time.Local
func DetectAndApplyTimezone() *time.Location {
	mu.Lock()
	defer mu.Unlock()

	loc, name := resolveTimezone()
	currentLoc = loc
	currentName = name
	time.Local = loc
	return loc
}

// GetLocation returns current resolved Location
func GetLocation() *time.Location {
	mu.RLock()
	defer mu.RUnlock()
	if currentLoc == nil {
		return time.Local
	}
	return currentLoc
}

// Now returns current time in detected router timezone
func Now() time.Time {
	return time.Now().In(GetLocation())
}

// GetTimezoneInfo returns description of active timezone and offset in seconds
func GetTimezoneInfo() (string, int) {
	mu.RLock()
	defer mu.RUnlock()
	now := Now()
	name, offset := now.Zone()
	return name, offset
}

// GetCurrentZonename returns the resolved zonename or POSIX string
func GetCurrentZonename() string {
	mu.RLock()
	defer mu.RUnlock()
	if currentName != "" {
		return currentName
	}
	return "UTC"
}

func resolveTimezone() (*time.Location, string) {
	// 1. Try UCI zonename (e.g. "Asia/Shanghai")
	if out, err := exec.Command("uci", "get", "system.@system[0].zonename").Output(); err == nil {
		zonename := strings.TrimSpace(string(out))
		if zonename != "" {
			if loc, err := time.LoadLocation(zonename); err == nil {
				log.Printf("[Timezone] Loaded from UCI zonename '%s'", zonename)
				return loc, zonename
			}
		}
	}

	// 2. Try /etc/TZ or UCI timezone
	var tzStr string
	if tzBytes, err := os.ReadFile("/etc/TZ"); err == nil {
		tzStr = strings.TrimSpace(string(tzBytes))
	}
	if tzStr == "" {
		if out, err := exec.Command("uci", "get", "system.@system[0].timezone").Output(); err == nil {
			tzStr = strings.TrimSpace(string(out))
		}
	}
	if tzStr == "" {
		tzStr = os.Getenv("TZ")
	}

	if tzStr != "" {
		// Try to parse POSIX TZ string
		if loc, name := ParsePOSIXTZ(tzStr); loc != nil {
			log.Printf("[Timezone] Parsed POSIX TZ '%s' -> %s (%v)", tzStr, name, loc)
			return loc, name
		}
	}

	// 3. Fallback to /etc/localtime or system local
	if loc, err := time.LoadLocation("Local"); err == nil && loc != nil {
		return loc, loc.String()
	}

	return time.UTC, "UTC"
}

var posixTZRegex = regexp.MustCompile(`^([A-Za-z]+)([+-]?\d+(?::\d+)?(?:[:\d+])?)`)

// ParsePOSIXTZ parses POSIX timezone strings like "CST-8", "EST5EDT", "GMT0", etc.
func ParsePOSIXTZ(tzStr string) (*time.Location, string) {
	tzStr = strings.TrimSpace(tzStr)
	if tzStr == "" {
		return nil, ""
	}

	// Try standard IANA name first if it doesn't look like pure POSIX
	if loc, err := time.LoadLocation(tzStr); err == nil {
		return loc, tzStr
	}

	// Common aliases
	switch tzStr {
	case "CST-8", "GMT-8", "UTC-8":
		if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
			return loc, "Asia/Shanghai"
		}
	case "JST-9", "GMT-9", "UTC-9":
		if loc, err := time.LoadLocation("Asia/Tokyo"); err == nil {
			return loc, "Asia/Tokyo"
		}
	case "GMT0", "UTC0", "UTC":
		return time.UTC, "UTC"
	}

	matches := posixTZRegex.FindStringSubmatch(tzStr)
	if len(matches) >= 3 {
		zoneName := matches[1]
		offsetStr := matches[2]

		// In POSIX: offset is subtracted from local to get UTC.
		// "CST-8" means Local - (-8) = UTC => Local = UTC + 8.
		// "EST5" means Local - (+5) = UTC => Local = UTC - 5.
		sign := 1
		if strings.HasPrefix(offsetStr, "-") {
			sign = -1
			offsetStr = strings.TrimPrefix(offsetStr, "-")
		} else if strings.HasPrefix(offsetStr, "+") {
			sign = 1
			offsetStr = strings.TrimPrefix(offsetStr, "+")
		}

		parts := strings.Split(offsetStr, ":")
		hours, _ := strconv.Atoi(parts[0])
		minutes := 0
		if len(parts) > 1 {
			minutes, _ = strconv.Atoi(parts[1])
		}

		// Positive POSIX offset means WEST of GMT (negative UTC offset)
		// Negative POSIX offset means EAST of GMT (positive UTC offset)
		totalSeconds := -sign * (hours*3600 + minutes*60)
		return time.FixedZone(zoneName, totalSeconds), zoneName
	}

	return nil, ""
}
