package quota

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"parentcontrol/internal/device"
	"parentcontrol/internal/dpi"
	"parentcontrol/internal/firewall"
	"parentcontrol/internal/models"
	"parentcontrol/internal/stats"
	"parentcontrol/internal/tz"
)

// PolicyEngine coordinates quotas, schedule rules, and firewall enforcement
type PolicyEngine struct {
	mu           sync.RWMutex
	members      map[string]*models.Member
	settings     models.GlobalSettings
	fw           *firewall.FirewallManager
	dpi          *dpi.DPIManager
	tracker      *device.DeviceTracker
	stats        *stats.StatsTracker
	stopChan     chan struct{}
	lastResetDay int
}

// NewPolicyEngine creates a new PolicyEngine instance
func NewPolicyEngine(fw *firewall.FirewallManager, dpiMgr *dpi.DPIManager, dt *device.DeviceTracker) *PolicyEngine {
	engine := &PolicyEngine{
		members:      make(map[string]*models.Member),
		fw:           fw,
		dpi:          dpiMgr,
		tracker:      dt,
		stopChan:     make(chan struct{}),
		lastResetDay: -1,
		settings: models.GlobalSettings{
			Enabled:           true,
			WebPort:           8088,
			EnforceSafeSearch: true,
			BlockDoHDoT:       true,
			IsolateNewDevices: false,
			DailyResetHour:    0,
		},
	}
	return engine
}

// SetStatsTracker sets the statistical tracking engine
func (pe *PolicyEngine) SetStatsTracker(st *stats.StatsTracker) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.stats = st
}

// Start launches the background periodic policy evaluation and quota counting loop
func (pe *PolicyEngine) Start() {
	go pe.runLoop()
	log.Println("[Engine] Policy and quota engine started.")
}

// Stop terminates the policy engine
func (pe *PolicyEngine) Stop() {
	close(pe.stopChan)
	pe.fw.Cleanup()
	if pe.stats != nil {
		_ = pe.stats.Save()
	}
	log.Println("[Engine] Policy engine stopped.")
}

func (pe *PolicyEngine) runLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	minuteCounter := 0

	for {
		select {
		case <-pe.stopChan:
			return
		case <-ticker.C:
			now := tz.Now()
			pe.tracker.ScanDevices()

			// Check if daily quota reset is due
			pe.checkDailyReset(now)

			minuteCounter++
			// Evaluate active usage duration accumulation every 60s (6 x 10s ticks)
			if minuteCounter >= 6 {
				minuteCounter = 0
				pe.evaluateActiveUsage(now)
			}

			// Evaluate and sync firewall & DPI rules on each tick
			pe.EvaluateAndApply(now)
		}
	}
}

// checkDailyReset resets daily active duration at the specified hour
func (pe *PolicyEngine) checkDailyReset(now time.Time) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	resetHour := pe.settings.DailyResetHour
	if now.Hour() == resetHour && now.Day() != pe.lastResetDay {
		log.Printf("[Quota] Daily quota reset triggered at %s for day %d.", now.Format("15:04:05"), now.Day())
		for _, m := range pe.members {
			m.UsedMinutes = 0
		}
		pe.lastResetDay = now.Day()

		if pe.stats != nil {
			pe.stats.CheckDailyRollover(now, resetHour)
		}
	}
}

// evaluateActiveUsage checks if managed devices have network traffic and increments active minutes
func (pe *PolicyEngine) evaluateActiveUsage(now time.Time) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	// 1. Scan and record per-device activity into stats engine
	devices := pe.tracker.ScanDevices()
	for _, dev := range devices {
		if dev.Online && (dev.RxRate > 2048 || dev.TxRate > 2048) {
			if pe.stats != nil {
				pe.stats.RecordMinuteActivity(dev.MAC, dev.IP, dev.Hostname, dev.MemberID, dev.RxRate*60, now)
			}
		}
	}

	// 2. Scan DPI / OAF visits
	if pe.stats != nil {
		pe.stats.ScanOAFVisits(now)
	}

	// 3. Update member used minutes
	for _, member := range pe.members {
		if !member.Enabled {
			continue
		}

		// Check if any bound device has active traffic (downstream + upstream rate > 2KB/s)
		isActive := false
		for _, mac := range member.DeviceMACs {
			dev := pe.tracker.GetDevice(mac)
			if dev != nil && dev.Online && (dev.RxRate > 2048 || dev.TxRate > 2048) {
				isActive = true
				break
			}
		}

		if isActive {
			member.UsedMinutes++
			member.LastActiveTime = now
		}
	}
}

// EvaluateAndApply makes policy decisions: determines allow/block state for all devices and applies to kernel
func (pe *PolicyEngine) EvaluateAndApply(now time.Time) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if !pe.settings.Enabled {
		_ = pe.fw.SyncBlockedMACs([]string{})
		_ = pe.dpi.ApplyRules([]int{}, []string{})
		return
	}

	blockedMACs := make([]string, 0)
	allManagedMACs := make([]string, 0)
	allBlockedAppIDs := make(map[int]bool)

	for _, member := range pe.members {
		if !member.Enabled {
			continue
		}

		allManagedMACs = append(allManagedMACs, member.DeviceMACs...)

		// Collect blocked DPI App IDs
		for _, appID := range member.BlockedAppIDs {
			allBlockedAppIDs[appID] = true
		}

		// Determine if member should be completely disconnected
		shouldBlock := pe.shouldBlockMember(member, now)

		if shouldBlock {
			blockedMACs = append(blockedMACs, member.DeviceMACs...)
		}
	}

	// Add blacklisted MAC addresses (single device one-click block)
	for _, mac := range pe.settings.CustomBlacklist {
		if mac != "" {
			blockedMACs = append(blockedMACs, mac)
		}
	}

	// Apply iptables block rules
	_ = pe.fw.SyncBlockedMACs(blockedMACs)

	// Apply kmod-oaf DPI rules
	appIDsSlice := make([]int, 0, len(allBlockedAppIDs))
	for id := range allBlockedAppIDs {
		appIDsSlice = append(appIDsSlice, id)
	}
	_ = pe.dpi.ApplyRules(appIDsSlice, allManagedMACs)
}

// shouldBlockMember determines whether an individual member should have internet access cut off
func (pe *PolicyEngine) shouldBlockMember(m *models.Member, now time.Time) bool {
	// 1. Check one-click lock
	if m.IsLocked {
		return true
	}

	// 2. Check temporary bonus time exemption
	if m.BonusUntil != nil && m.BonusUntil.After(now) {
		return false // Under bonus time, allow access
	}

	// 3. Check daily quota limit
	if m.QuotaMinutes > 0 && m.UsedMinutes >= m.QuotaMinutes {
		return true
	}

	// 4. Check schedule rules
	if m.Schedule.Enabled {
		currentWeekday := int(now.Weekday())
		isMatchedDay := false
		for _, day := range m.Schedule.Days {
			if day == currentWeekday {
				isMatchedDay = true
				break
			}
		}

		if isMatchedDay {
			inRange := false
			currentHM := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

			for _, tr := range m.Schedule.TimeRanges {
				if isTimeInRange(currentHM, tr.StartTime, tr.EndTime) {
					inRange = true
					break
				}
			}

			if m.Schedule.Action == "block" && inRange {
				return true // Matched block interval
			}
			if m.Schedule.Action == "allow" && !inRange {
				return true // Outside allowed interval
			}
		}
	}

	return false
}

// isTimeInRange determines whether current time is between start and end (supports overnight spans like "21:00" to "07:00")
func isTimeInRange(curr, start, end string) bool {
	if start == "" || end == "" {
		return false
	}
	if start <= end {
		return curr >= start && curr <= end
	}
	// Spans midnight
	return curr >= start || curr <= end
}

// SetMember adds or updates a managed member
func (pe *PolicyEngine) SetMember(m models.Member) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.members[m.ID] = &m
}

// DeleteMember removes a managed member
func (pe *PolicyEngine) DeleteMember(id string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	delete(pe.members, id)
}

// GetMembers returns all managed members
func (pe *PolicyEngine) GetMembers() []models.Member {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	list := make([]models.Member, 0, len(pe.members))
	for _, m := range pe.members {
		list = append(list, *m)
	}
	return list
}

// GetMember retrieves an individual member by ID
func (pe *PolicyEngine) GetMember(id string) (*models.Member, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	m, ok := pe.members[id]
	if !ok {
		return nil, false
	}
	cp := *m
	return &cp, true
}

// LockMember locks/blocks a member via one-click lock
func (pe *PolicyEngine) LockMember(id string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	m, ok := pe.members[id]
	if !ok {
		return fmt.Errorf("member not found")
	}
	m.IsLocked = true
	return nil
}

// UnlockMember restores internet access for a locked member
func (pe *PolicyEngine) UnlockMember(id string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	m, ok := pe.members[id]
	if !ok {
		return fmt.Errorf("member not found")
	}
	m.IsLocked = false

	// Clean up any member devices from CustomBlacklist if present
	macMap := make(map[string]bool)
	for _, dmac := range m.DeviceMACs {
		macMap[strings.ToLower(strings.TrimSpace(dmac))] = true
	}
	newList := make([]string, 0, len(pe.settings.CustomBlacklist))
	for _, bmac := range pe.settings.CustomBlacklist {
		if !macMap[strings.ToLower(strings.TrimSpace(bmac))] {
			newList = append(newList, bmac)
		}
	}
	pe.settings.CustomBlacklist = newList
	return nil
}

// BonusMember grants temporary bonus internet time in minutes
func (pe *PolicyEngine) BonusMember(id string, minutes int) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	m, ok := pe.members[id]
	if !ok {
		return fmt.Errorf("member not found")
	}
	bonusUntil := tz.Now().Add(time.Duration(minutes) * time.Minute)
	m.BonusUntil = &bonusUntil
	return nil
}

// UpdateSettings updates system-wide settings
func (pe *PolicyEngine) UpdateSettings(s models.GlobalSettings) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.settings = s
}

// GetSettings retrieves current system-wide settings
func (pe *PolicyEngine) GetSettings() models.GlobalSettings {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.settings
}

// LockDevice locks/blocks an individual device by MAC
func (pe *PolicyEngine) LockDevice(mac string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	mac = strings.ToLower(strings.TrimSpace(mac))
	if mac == "" {
		return
	}

	// Check if already present in blacklist
	exists := false
	for _, m := range pe.settings.CustomBlacklist {
		if strings.ToLower(m) == mac {
			exists = true
			break
		}
	}
	if !exists {
		pe.settings.CustomBlacklist = append(pe.settings.CustomBlacklist, mac)
	}

	// If device belongs to a member, check and update member lock state
	for _, m := range pe.members {
		for _, dmac := range m.DeviceMACs {
			if strings.ToLower(dmac) == mac {
				m.IsLocked = true
				break
			}
		}
	}
}

// UnlockDevice removes single-device one-click block by MAC
func (pe *PolicyEngine) UnlockDevice(mac string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	mac = strings.ToLower(strings.TrimSpace(mac))
	newList := make([]string, 0, len(pe.settings.CustomBlacklist))
	for _, m := range pe.settings.CustomBlacklist {
		if strings.ToLower(m) != mac {
			newList = append(newList, m)
		}
	}
	pe.settings.CustomBlacklist = newList

	// If device belongs to a member, unlock that member as well
	for _, m := range pe.members {
		for _, dmac := range m.DeviceMACs {
			if strings.ToLower(dmac) == mac {
				m.IsLocked = false
				break
			}
		}
	}
}

// AssignDeviceToMember assigns a device to a specified member (unbinds if memberID is empty)
func (pe *PolicyEngine) AssignDeviceToMember(mac string, memberID string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	mac = strings.ToLower(strings.TrimSpace(mac))
	if mac == "" {
		return
	}

	// 1. Remove MAC from all existing members to prevent duplicate bindings
	for _, m := range pe.members {
		newMACs := make([]string, 0, len(m.DeviceMACs))
		for _, dmac := range m.DeviceMACs {
			if strings.ToLower(dmac) != mac {
				newMACs = append(newMACs, dmac)
			}
		}
		m.DeviceMACs = newMACs
	}

	// 2. If target memberID is specified, bind to target member
	if memberID != "" {
		if target, ok := pe.members[memberID]; ok {
			target.DeviceMACs = append(target.DeviceMACs, mac)
		}
	}
}

// ParseTimeString helper to parse "HH:MM" into total minutes from midnight
func ParseTimeString(s string) int {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}
