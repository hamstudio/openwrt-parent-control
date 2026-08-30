package stats

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"parentcontrol/internal/dpi"
	"parentcontrol/internal/models"
	"parentcontrol/internal/tz"
)

type statsPersistSchema struct {
	CurrentDate   string                                            `json:"current_date"`
	TodayStats    map[string]*models.DeviceDayStats                 `json:"today_stats"`
	History       map[string]map[string]*models.HistoricalDayRecord `json:"history"`        // MAC -> Date -> Record
	GlobalHistory map[string]*models.HistoricalDayRecord            `json:"global_history"` // Date -> Record
}

// StatsTracker manages device usage minutes, hourly distribution, DPI category profiling and history
type StatsTracker struct {
	mu            sync.RWMutex
	filePath      string
	dpiMgr        *dpi.DPIManager
	todayStats    map[string]*models.DeviceDayStats                 // key: MAC (uppercase)
	history       map[string]map[string]*models.HistoricalDayRecord // key: MAC -> Date -> Record
	globalHistory map[string]*models.HistoricalDayRecord            // key: Date -> Record
	currentDate   string
	lastSave      time.Time
	lastRollover  int
}

// NewStatsTracker creates a new StatsTracker instance
func NewStatsTracker(filePath string, dpiMgr *dpi.DPIManager) *StatsTracker {
	if filePath == "" {
		filePath = "/etc/parentcontrol/stats.json"
	}

	todayStr := tz.Now().Format("2006-01-02")
	st := &StatsTracker{
		filePath:      filePath,
		dpiMgr:        dpiMgr,
		todayStats:    make(map[string]*models.DeviceDayStats),
		history:       make(map[string]map[string]*models.HistoricalDayRecord),
		globalHistory: make(map[string]*models.HistoricalDayRecord),
		currentDate:   todayStr,
		lastSave:      time.Now(),
		lastRollover:  -1,
	}

	st.Load()
	return st
}

// Load reads existing statistics from the persistent file
func (st *StatsTracker) Load() {
	st.mu.Lock()
	defer st.mu.Unlock()

	data, err := os.ReadFile(st.filePath)
	if err != nil {
		log.Printf("[Stats] No existing stats file at %s, starting fresh.", st.filePath)
		return
	}

	var schema statsPersistSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		log.Printf("[Stats] Failed to parse stats JSON: %v. Starting fresh.", err)
		return
	}

	if schema.History != nil {
		st.history = schema.History
	}
	if schema.GlobalHistory != nil {
		st.globalHistory = schema.GlobalHistory
	}

	todayStr := tz.Now().Format("2006-01-02")
	if schema.CurrentDate == todayStr && schema.TodayStats != nil {
		st.todayStats = schema.TodayStats
		st.currentDate = todayStr
	} else if schema.CurrentDate != "" && schema.TodayStats != nil {
		// Archive previous un-archived today's data into history
		st.archiveDayLocked(schema.CurrentDate, schema.TodayStats)
		st.todayStats = make(map[string]*models.DeviceDayStats)
		st.currentDate = todayStr
	}

	log.Printf("[Stats] Loaded stats database: %d devices in today, %d historical devices.", len(st.todayStats), len(st.history))
}

// Save persists the current statistics to file
func (st *StatsTracker) Save() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.saveUnsafe()
}

func (st *StatsTracker) saveUnsafe() error {
	dir := filepath.Dir(st.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	schema := statsPersistSchema{
		CurrentDate:   st.currentDate,
		TodayStats:    st.todayStats,
		History:       st.history,
		GlobalHistory: st.globalHistory,
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	st.lastSave = time.Now()
	return os.WriteFile(st.filePath, data, 0644)
}

// CheckDailyRollover verifies if date changed or daily reset hour was reached
func (st *StatsTracker) CheckDailyRollover(now time.Time, resetHour int) {
	st.mu.Lock()
	defer st.mu.Unlock()

	todayStr := now.Format("2006-01-02")
	if st.currentDate != todayStr && now.Hour() >= resetHour && st.lastRollover != now.Day() {
		log.Printf("[Stats] Rolling over daily stats from %s to %s at hour %d", st.currentDate, todayStr, now.Hour())
		st.archiveDayLocked(st.currentDate, st.todayStats)
		st.todayStats = make(map[string]*models.DeviceDayStats)
		st.currentDate = todayStr
		st.lastRollover = now.Day()
		_ = st.saveUnsafe()
	}
}

func (st *StatsTracker) archiveDayLocked(date string, dayStats map[string]*models.DeviceDayStats) {
	if date == "" || len(dayStats) == 0 {
		return
	}

	var globalTotalMinutes int
	var globalRx uint64
	var globalTx uint64

	cutoffDate := tz.Now().AddDate(0, 0, -31).Format("2006-01-02")

	for mac, d := range dayStats {
		mac = strings.ToUpper(mac)
		if _, exists := st.history[mac]; !exists {
			st.history[mac] = make(map[string]*models.HistoricalDayRecord)
		}

		st.history[mac][date] = &models.HistoricalDayRecord{
			Date:        date,
			UsedMinutes: d.UsedMinutes,
			RxBytes:     d.RxBytes,
			TxBytes:     d.TxBytes,
		}

		// Prune records older than 30 days
		for recDate := range st.history[mac] {
			if recDate < cutoffDate {
				delete(st.history[mac], recDate)
			}
		}

		globalTotalMinutes += d.UsedMinutes
		globalRx += d.RxBytes
		globalTx += d.TxBytes
	}

	st.globalHistory[date] = &models.HistoricalDayRecord{
		Date:        date,
		UsedMinutes: globalTotalMinutes,
		RxBytes:     globalRx,
		TxBytes:     globalTx,
	}

	for recDate := range st.globalHistory {
		if recDate < cutoffDate {
			delete(st.globalHistory, recDate)
		}
	}
}

// RecordMinuteActivity records active usage for a device during the current minute
func (st *StatsTracker) RecordMinuteActivity(mac, ip, hostname, memberID string, diffBytes uint64, now time.Time) {
	st.mu.Lock()
	defer st.mu.Unlock()

	mac = strings.ToUpper(strings.TrimSpace(mac))
	if mac == "" {
		return
	}

	todayStr := now.Format("2006-01-02")
	if st.currentDate != todayStr {
		st.archiveDayLocked(st.currentDate, st.todayStats)
		st.todayStats = make(map[string]*models.DeviceDayStats)
		st.currentDate = todayStr
	}

	devStat, exists := st.todayStats[mac]
	if !exists {
		devStat = &models.DeviceDayStats{
			Date:       todayStr,
			MAC:        mac,
			IP:         ip,
			Hostname:   hostname,
			MemberID:   memberID,
			Categories: make(map[string]*models.CategoryUsageStat),
			TopApps:    make([]*models.AppUsageStat, 0),
		}
		st.todayStats[mac] = devStat
	}

	if ip != "" {
		devStat.IP = ip
	}
	if hostname != "" && (devStat.Hostname == "" || devStat.Hostname == "Unknown-Device") {
		devStat.Hostname = hostname
	}
	if memberID != "" {
		devStat.MemberID = memberID
	}

	localNow := now.In(tz.GetLocation())
	devStat.UsedMinutes++
	hour := localNow.Hour()
	if hour >= 0 && hour < 24 {
		devStat.HourlyMinutes[hour]++
	}
	devStat.RxBytes += diffBytes

	// Auto save periodically (every 5 minutes)
	if time.Since(st.lastSave) > 5*time.Minute {
		_ = st.saveUnsafe()
	}
}

// RecordAppActivity records an application usage event for a device
func (st *StatsTracker) RecordAppActivity(mac string, appID int, bytes uint64, now time.Time) {
	st.mu.Lock()
	defer st.mu.Unlock()

	mac = strings.ToUpper(strings.TrimSpace(mac))
	if mac == "" || appID <= 0 {
		return
	}

	todayStr := now.Format("2006-01-02")
	devStat, exists := st.todayStats[mac]
	if !exists {
		devStat = &models.DeviceDayStats{
			Date:       todayStr,
			MAC:        mac,
			Categories: make(map[string]*models.CategoryUsageStat),
			TopApps:    make([]*models.AppUsageStat, 0),
		}
		st.todayStats[mac] = devStat
	}

	var appName, className, classZh string
	var classID int
	if st.dpiMgr != nil {
		if app, ok := st.dpiMgr.GetApp(appID); ok {
			appName = app.Name
			classID = app.ClassID
			className = app.ClassName
			classZh = app.ClassZh
		}
	}

	if appName == "" {
		appName = fmt.Sprintf("App-%d", appID)
		classID = 99
		className = "other"
		classZh = "其他"
	}
	if className == "" {
		className = "other"
		classZh = "其他"
	}

	// 1. Update Category stats
	catStat, exists := devStat.Categories[className]
	if !exists {
		catStat = &models.CategoryUsageStat{
			ClassID:   classID,
			ClassName: className,
			ClassZh:   classZh,
			Icon:      getCategoryIconName(className),
		}
		devStat.Categories[className] = catStat
	}
	catStat.Minutes++
	catStat.Bytes += bytes

	// 2. Update TopApps
	var foundApp *models.AppUsageStat
	for _, a := range devStat.TopApps {
		if a.AppID == appID {
			foundApp = a
			break
		}
	}
	if foundApp == nil {
		foundApp = &models.AppUsageStat{
			AppID:     appID,
			AppName:   appName,
			ClassID:   classID,
			ClassName: className,
			ClassZh:   classZh,
		}
		devStat.TopApps = append(devStat.TopApps, foundApp)
	}
	foundApp.Visits++
	foundApp.Minutes++
	foundApp.Bytes += bytes
	foundApp.LastActive = now

	// Sort TopApps by minutes / visits descending
	sort.Slice(devStat.TopApps, func(i, j int) bool {
		return devStat.TopApps[i].Minutes > devStat.TopApps[j].Minutes
	})
	if len(devStat.TopApps) > 20 {
		devStat.TopApps = devStat.TopApps[:20]
	}
}

// ScanOAFVisits parses /proc/sys/oaf/dev_visit_stat or /proc/sys/oaf/visit_list if present
func (st *StatsTracker) ScanOAFVisits(now time.Time) {
	paths := []string{
		"/proc/sys/oaf/dev_visit_stat",
		"/proc/sys/oaf/visit_list",
		"/proc/sys/oaf/dev_visit_info",
	}

	for _, path := range paths {
		if file, err := os.Open(path); err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "MAC") {
					continue
				}
				// Typical format: MAC IP APP_ID TOTAL_NUM DROP_NUM LATEST_TIME
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					mac := strings.ToUpper(fields[0])
					if strings.Count(mac, ":") == 5 {
						appID, err := strconv.Atoi(fields[2])
						if err == nil && appID > 0 {
							var totalNum uint64
							if len(fields) >= 4 {
								totalNum, _ = strconv.ParseUint(fields[3], 10, 64)
							}
							st.RecordAppActivity(mac, appID, totalNum*512, now)
						}
					}
				}
			}
			file.Close()
			break
		}
	}
}

// GetDeviceUsedMinutesToday returns today's active minutes for a specific MAC
func (st *StatsTracker) GetDeviceUsedMinutesToday(mac string) int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	mac = strings.ToUpper(strings.TrimSpace(mac))
	if d, ok := st.todayStats[mac]; ok {
		return d.UsedMinutes
	}
	return 0
}

// GetMemberUsedMinutesToday aggregates today's active minutes across all devices bound to a member
func (st *StatsTracker) GetMemberUsedMinutesToday(macs []string) int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	total := 0
	for _, mac := range macs {
		norm := strings.ToUpper(strings.TrimSpace(mac))
		if d, ok := st.todayStats[norm]; ok {
			total += d.UsedMinutes
		}
	}
	return total
}

// GetOverview aggregates family router usage metrics for today and 7-day trend
func (st *StatsTracker) GetOverview(devices []*models.Device, members []models.Member) *models.StatsOverview {
	st.mu.RLock()
	defer st.mu.RUnlock()

	todayStr := st.currentDate
	if todayStr == "" {
		todayStr = tz.Now().Format("2006-01-02")
	}

	overview := &models.StatsOverview{
		Date:                todayStr,
		CategoryBreakdown:   make([]*models.CategoryUsageStat, 0),
		DeviceRankings:      make([]models.DeviceDaySummary, 0),
		HistoricalTrend7Day: make([]*models.HistoricalDayRecord, 0),
	}

	memberMap := make(map[string]string)
	for _, m := range members {
		memberMap[m.ID] = m.Name
	}

	devMap := make(map[string]*models.Device)
	for _, d := range devices {
		devMap[strings.ToUpper(d.MAC)] = d
	}

	categoryAgg := make(map[string]*models.CategoryUsageStat)
	totalCategoryMinutes := 0

	for _, d := range devices {
		normMAC := strings.ToUpper(d.MAC)
		usedMin := 0
		var totalBytes uint64 = d.TotalBytes
		topCat := "其他"

		if dayStat, ok := st.todayStats[normMAC]; ok {
			usedMin = dayStat.UsedMinutes
			if dayStat.RxBytes+dayStat.TxBytes > totalBytes {
				totalBytes = dayStat.RxBytes + dayStat.TxBytes
			}

			// Aggregate categories
			var maxCatMin int
			for catName, cat := range dayStat.Categories {
				if _, exists := categoryAgg[catName]; !exists {
					categoryAgg[catName] = &models.CategoryUsageStat{
						ClassID:   cat.ClassID,
						ClassName: cat.ClassName,
						ClassZh:   cat.ClassZh,
						Icon:      cat.Icon,
					}
				}
				categoryAgg[catName].Minutes += cat.Minutes
				categoryAgg[catName].Bytes += cat.Bytes
				totalCategoryMinutes += cat.Minutes

				if cat.Minutes > maxCatMin {
					maxCatMin = cat.Minutes
					topCat = cat.ClassZh
				}
			}
		}

		if d.Online || usedMin > 0 {
			overview.ActiveDeviceCount++
		}
		overview.TotalOnlineMinutes += usedMin
		overview.TotalBytes += totalBytes

		memberName := ""
		if d.MemberID != "" {
			memberName = memberMap[d.MemberID]
		}

		overview.DeviceRankings = append(overview.DeviceRankings, models.DeviceDaySummary{
			MAC:         d.MAC,
			Hostname:    d.Hostname,
			Vendor:      d.Vendor,
			MemberID:    d.MemberID,
			MemberName:  memberName,
			Online:      d.Online,
			UsedMinutes: usedMin,
			TotalBytes:  totalBytes,
			TopCategory: topCat,
		})
	}

	// Sort device rankings by used minutes descending
	sort.Slice(overview.DeviceRankings, func(i, j int) bool {
		return overview.DeviceRankings[i].UsedMinutes > overview.DeviceRankings[j].UsedMinutes
	})

	// Process category percentages
	var topCatName string
	var topCatMinutes int
	for _, cat := range categoryAgg {
		if totalCategoryMinutes > 0 {
			cat.Percentage = float64(cat.Minutes) / float64(totalCategoryMinutes) * 100.0
		}
		if cat.Minutes > topCatMinutes {
			topCatMinutes = cat.Minutes
			topCatName = cat.ClassZh
		}
		overview.CategoryBreakdown = append(overview.CategoryBreakdown, cat)
	}
	sort.Slice(overview.CategoryBreakdown, func(i, j int) bool {
		return overview.CategoryBreakdown[i].Minutes > overview.CategoryBreakdown[j].Minutes
	})

	overview.TopCategory = topCatName
	overview.TopCategoryMinutes = topCatMinutes

	// Fill 7-day historical trend
	baseDate, err := time.ParseInLocation("2006-01-02", todayStr, tz.Now().Location())
	if err != nil {
		baseDate = tz.Now()
	}

	for i := 6; i >= 0; i-- {
		date := baseDate.AddDate(0, 0, -i).Format("2006-01-02")
		if date == todayStr {
			overview.HistoricalTrend7Day = append(overview.HistoricalTrend7Day, &models.HistoricalDayRecord{
				Date:        date,
				UsedMinutes: overview.TotalOnlineMinutes,
				RxBytes:     overview.TotalBytes,
			})
		} else if rec, ok := st.globalHistory[date]; ok {
			overview.HistoricalTrend7Day = append(overview.HistoricalTrend7Day, rec)
		} else {
			overview.HistoricalTrend7Day = append(overview.HistoricalTrend7Day, &models.HistoricalDayRecord{
				Date:        date,
				UsedMinutes: 0,
				RxBytes:     0,
			})
		}
	}

	return overview
}

// GetDeviceStatsDetail returns detailed analytics for a single device
func (st *StatsTracker) GetDeviceStatsDetail(mac string, days int, dev *models.Device, member *models.Member) *models.DeviceStatsDetail {
	st.mu.RLock()
	defer st.mu.RUnlock()

	normMAC := strings.ToUpper(strings.TrimSpace(mac))
	todayStr := st.currentDate
	if todayStr == "" {
		todayStr = tz.Now().Format("2006-01-02")
	}

	detail := &models.DeviceStatsDetail{
		MAC:               normMAC,
		History:           make([]*models.HistoricalDayRecord, 0),
		CategoryBreakdown: make([]*models.CategoryUsageStat, 0),
		TopApps:           make([]*models.AppUsageStat, 0),
	}

	if dev != nil {
		detail.Hostname = dev.Hostname
		detail.Vendor = dev.Vendor
		detail.Online = dev.Online
		detail.MemberID = dev.MemberID
	}
	if member != nil {
		detail.MemberName = member.Name
	}

	if dayStat, ok := st.todayStats[normMAC]; ok {
		detail.TodayStats = dayStat
		detail.HourlyActivity = dayStat.HourlyMinutes
		detail.TopApps = dayStat.TopApps

		totalCatMin := 0
		for _, cat := range dayStat.Categories {
			totalCatMin += cat.Minutes
		}
		for _, cat := range dayStat.Categories {
			cp := *cat
			if totalCatMin > 0 {
				cp.Percentage = float64(cp.Minutes) / float64(totalCatMin) * 100.0
			}
			detail.CategoryBreakdown = append(detail.CategoryBreakdown, &cp)
		}
		sort.Slice(detail.CategoryBreakdown, func(i, j int) bool {
			return detail.CategoryBreakdown[i].Minutes > detail.CategoryBreakdown[j].Minutes
		})
	} else {
		detail.TodayStats = &models.DeviceDayStats{
			Date:       todayStr,
			MAC:        normMAC,
			Categories: make(map[string]*models.CategoryUsageStat),
			TopApps:    make([]*models.AppUsageStat, 0),
		}
	}

	// Build historical trend
	if days <= 0 {
		days = 7
	}
	if days > 30 {
		days = 30
	}

	baseDate, err := time.ParseInLocation("2006-01-02", todayStr, tz.Now().Location())
	if err != nil {
		baseDate = tz.Now()
	}
	devHistory := st.history[normMAC]

	for i := days - 1; i >= 0; i-- {
		date := baseDate.AddDate(0, 0, -i).Format("2006-01-02")
		if date == todayStr {
			detail.History = append(detail.History, &models.HistoricalDayRecord{
				Date:        date,
				UsedMinutes: detail.TodayStats.UsedMinutes,
				RxBytes:     detail.TodayStats.RxBytes,
				TxBytes:     detail.TodayStats.TxBytes,
			})
		} else if devHistory != nil && devHistory[date] != nil {
			detail.History = append(detail.History, devHistory[date])
		} else {
			detail.History = append(detail.History, &models.HistoricalDayRecord{
				Date:        date,
				UsedMinutes: 0,
				RxBytes:     0,
				TxBytes:     0,
			})
		}
	}

	return detail
}

func getCategoryIconName(className string) string {
	switch className {
	case "chat":
		return "message-square"
	case "game":
		return "gamepad-2"
	case "video":
		return "play-circle"
	case "music":
		return "music"
	case "download":
		return "download"
	case "shop":
		return "shopping-bag"
	case "finance":
		return "dollar-sign"
	case "work":
		return "briefcase"
	default:
		return "grid"
	}
}
