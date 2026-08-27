package quota

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"parentcontrol/internal/device"
	"parentcontrol/internal/dpi"
	"parentcontrol/internal/firewall"
	"parentcontrol/internal/models"
)

// PolicyEngine 调度引擎：协调配额、时间表与防火墙执行
type PolicyEngine struct {
	mu           sync.RWMutex
	members      map[string]*models.Member
	settings     models.GlobalSettings
	fw           *firewall.FirewallManager
	dpi          *dpi.DPIManager
	tracker      *device.DeviceTracker
	stopChan     chan struct{}
	lastResetDay int
}

// NewPolicyEngine 创建调度引擎
func NewPolicyEngine(fw *firewall.FirewallManager, dpiMgr *dpi.DPIManager, dt *device.DeviceTracker) *PolicyEngine {
	engine := &PolicyEngine{
		members:      make(map[string]*models.Member),
		fw:           fw,
		dpi:          dpiMgr,
		tracker:      dt,
		stopChan:     make(chan struct{}),
		lastResetDay: time.Now().Day(),
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

// Start 启动后台周期性评估与配额计数循环
func (pe *PolicyEngine) Start() {
	go pe.runLoop()
	log.Println("[Engine] Policy and quota engine started.")
}

// Stop 停止调度引擎
func (pe *PolicyEngine) Stop() {
	close(pe.stopChan)
	pe.fw.Cleanup()
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
		case now := <-ticker.C:
			pe.tracker.ScanDevices()

			// 检查是否跨天重置配额
			pe.checkDailyReset(now)

			minuteCounter++
			// 每 60 秒 (6 次 10s tick) 评估一次时长累加
			if minuteCounter >= 6 {
				minuteCounter = 0
				pe.evaluateActiveUsage(now)
			}

			// 每次 tick 评估并同步防火墙与 DPI 规则
			pe.EvaluateAndApply(now)
		}
	}
}

// checkDailyReset 每天在指定小时重置今日已用时长
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
	}
}

// evaluateActiveUsage 检测受管成员的设备是否有网络活动并累加时长
func (pe *PolicyEngine) evaluateActiveUsage(now time.Time) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	for _, member := range pe.members {
		if !member.Enabled {
			continue
		}

		// 检查名下设备是否有流量 (上行+下行速率 > 2KB/s)
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

// EvaluateAndApply 核心策略决策：判定所有设备的放行/阻断状态并下发底层
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

		// 汇总需要封禁的 DPI App ID
		for _, appID := range member.BlockedAppIDs {
			allBlockedAppIDs[appID] = true
		}

		// 判定该成员是否应当完全切断外网
		shouldBlock := pe.shouldBlockMember(member, now)

		if shouldBlock {
			blockedMACs = append(blockedMACs, member.DeviceMACs...)
		}
	}

	// 汇总黑名单中的 MAC 地址 (单设备一键断网)
	for _, mac := range pe.settings.CustomBlacklist {
		if mac != "" {
			blockedMACs = append(blockedMACs, mac)
		}
	}

	// 下发 iptables 阻断规则
	_ = pe.fw.SyncBlockedMACs(blockedMACs)

	// 下发 kmod-oaf DPI 规则
	appIDsSlice := make([]int, 0, len(allBlockedAppIDs))
	for id := range allBlockedAppIDs {
		appIDsSlice = append(appIDsSlice, id)
	}
	_ = pe.dpi.ApplyRules(appIDsSlice, allManagedMACs)
}

// shouldBlockMember 判定单个成员是否需要被完全切断外网
func (pe *PolicyEngine) shouldBlockMember(m *models.Member, now time.Time) bool {
	// 1. 检查一键断网
	if m.IsLocked {
		return true
	}

	// 2. 检查临时加时 (Bonus Time) 豁免
	if m.BonusUntil != nil && m.BonusUntil.After(now) {
		return false // 加时中，放行
	}

	// 3. 检查每日配额超额
	if m.QuotaMinutes > 0 && m.UsedMinutes >= m.QuotaMinutes {
		return true
	}

	// 4. 检查时间计划表
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
				return true // 命中禁网时间段
			}
			if m.Schedule.Action == "allow" && !inRange {
				return true // 不在允许上网时间段内
			}
		}
	}

	return false
}

// isTimeInRange 判定 current 是否在 start 和 end 之间 (支持跨夜如 21:00 到 07:00)
func isTimeInRange(curr, start, end string) bool {
	if start == "" || end == "" {
		return false
	}
	if start <= end {
		return curr >= start && curr <= end
	}
	// 跨午夜
	return curr >= start || curr <= end
}

// SetMember 添加或更新成员
func (pe *PolicyEngine) SetMember(m models.Member) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.members[m.ID] = &m
}

// DeleteMember 删除成员
func (pe *PolicyEngine) DeleteMember(id string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	delete(pe.members, id)
}

// GetMembers 获取所有成员
func (pe *PolicyEngine) GetMembers() []models.Member {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	list := make([]models.Member, 0, len(pe.members))
	for _, m := range pe.members {
		list = append(list, *m)
	}
	return list
}

// GetMember 获取单个成员
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

// LockMember 一键断网
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

// UnlockMember 解除断网
func (pe *PolicyEngine) UnlockMember(id string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	m, ok := pe.members[id]
	if !ok {
		return fmt.Errorf("member not found")
	}
	m.IsLocked = false
	return nil
}

// BonusMember 奖励加时
func (pe *PolicyEngine) BonusMember(id string, minutes int) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	m, ok := pe.members[id]
	if !ok {
		return fmt.Errorf("member not found")
	}
	bonusUntil := time.Now().Add(time.Duration(minutes) * time.Minute)
	m.BonusUntil = &bonusUntil
	return nil
}

// UpdateSettings 更新全局设置
func (pe *PolicyEngine) UpdateSettings(s models.GlobalSettings) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.settings = s
}

// GetSettings 获取全局设置
func (pe *PolicyEngine) GetSettings() models.GlobalSettings {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.settings
}

// LockDevice 对单个设备执行一键断网
func (pe *PolicyEngine) LockDevice(mac string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	mac = strings.ToLower(strings.TrimSpace(mac))
	if mac == "" {
		return
	}

	// 检查是否已经在黑名单中
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

	// 如果该设备属于某个成员，同时检查成员状态
	for _, m := range pe.members {
		for _, dmac := range m.DeviceMACs {
			if strings.ToLower(dmac) == mac {
				m.IsLocked = true
				break
			}
		}
	}
}

// UnlockDevice 解除单个设备的一键断网
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

	// 如果属于某个成员，解除该成员的锁定
	for _, m := range pe.members {
		for _, dmac := range m.DeviceMACs {
			if strings.ToLower(dmac) == mac {
				m.IsLocked = false
				break
			}
		}
	}
}

// AssignDeviceToMember 快速将设备分配给指定成员（若 memberID 为空则解绑）
func (pe *PolicyEngine) AssignDeviceToMember(mac string, memberID string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	mac = strings.ToLower(strings.TrimSpace(mac))
	if mac == "" {
		return
	}

	// 1. 先从所有现有成员中移除该 MAC，避免重复绑定
	for _, m := range pe.members {
		newMACs := make([]string, 0, len(m.DeviceMACs))
		for _, dmac := range m.DeviceMACs {
			if strings.ToLower(dmac) != mac {
				newMACs = append(newMACs, dmac)
			}
		}
		m.DeviceMACs = newMACs
	}

	// 2. 如果指定了目标 memberID，则加入目标成员
	if memberID != "" {
		if target, ok := pe.members[memberID]; ok {
			target.DeviceMACs = append(target.DeviceMACs, mac)
		}
	}
}

// ParseTimeString 格式化辅助
func ParseTimeString(s string) int {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}
