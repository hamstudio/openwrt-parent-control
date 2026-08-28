package dpi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"

	"parentcontrol/internal/models"
)

// DPIManager manages Deep Packet Inspection (DPI) app identification and blocking
type DPIManager struct {
	mu             sync.RWMutex
	featurePath    string
	categories     []models.AppCategory
	appMap         map[int]models.AppInfo
	customApps     map[int]models.AppInfo
	customCats     map[int]models.AppCategory
	blockedAppIDs  map[int]bool
	managedMACs    map[string]bool
	isOAFInstalled bool
}

// NewDPIManager initializes a new DPIManager instance
func NewDPIManager(featurePath string) *DPIManager {
	if featurePath == "" {
		featurePath = "/etc/appfilter/feature_cn.cfg"
	}
	mgr := &DPIManager{
		featurePath:   featurePath,
		categories:    make([]models.AppCategory, 0),
		appMap:        make(map[int]models.AppInfo),
		customApps:    make(map[int]models.AppInfo),
		customCats:    make(map[int]models.AppCategory),
		blockedAppIDs: make(map[int]bool),
		managedMACs:   make(map[string]bool),
	}
	mgr.Init()
	return mgr
}

// Init initializes the kernel module and loads the feature database
func (m *DPIManager) Init() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Check and attempt to load kmod-oaf
	if _, err := os.Stat("/dev/appfilter"); os.IsNotExist(err) {
		log.Println("[DPI] /dev/appfilter not found, preparing feature file and loading oaf module...")
		_ = os.Remove("/tmp/feature.cfg")
		_ = exec.Command("ln", "-sf", m.featurePath, "/tmp/feature.cfg").Run()
		if _, err := os.Stat("/usr/libexec/oaf/gen_class.sh"); err == nil {
			_ = exec.Command("/usr/libexec/oaf/gen_class.sh", "/tmp/feature.cfg").Run()
		}
		_ = exec.Command("insmod", "oaf").Run()
	}

	if _, err := os.Stat("/dev/appfilter"); err == nil {
		m.isOAFInstalled = true
		log.Println("[DPI] kmod-oaf kernel module is active.")
	} else {
		m.isOAFInstalled = false
		log.Printf("[DPI] WARNING: kmod-oaf device /dev/appfilter not accessible: %v", err)
	}

	// 2. Load feature database
	m.loadFeatures()
}

// IsReady checks if the DPI kernel module is ready
func (m *DPIManager) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isOAFInstalled
}

// loadFeatures parses application signatures and categories from feature configuration
func (m *DPIManager) loadFeatures() {
	file, err := os.Open(m.featurePath)
	if err != nil {
		log.Printf("[DPI] Failed to open feature file %s: %v. Using fallback definitions.", m.featurePath, err)
		m.loadFallbackFeatures()
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentCat *models.AppCategory
	categoriesMap := make(map[int]*models.AppCategory)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#version") || strings.HasPrefix(line, "#format") || strings.HasPrefix(line, "#id") {
			continue
		}

		// Category line: #class chat 1 Chat
		if strings.HasPrefix(line, "#class") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				className := fields[1]
				classID, _ := strconv.Atoi(fields[2])
				classZh := fields[3]
				cat := models.AppCategory{
					ClassID:   classID,
					ClassName: className,
					ClassZh:   classZh,
					Icon:      getCategoryIcon(className),
					Apps:      make([]models.AppInfo, 0),
				}
				categoriesMap[classID] = &cat
				currentCat = &cat
			}
			continue
		}

		// App signature line: 1001 QQ:[tcp;;;;;00:02|-1:03...]
		parts := strings.SplitN(line, ":", 2)
		if len(parts) >= 1 {
			idName := strings.TrimSpace(parts[0])
			subParts := strings.Fields(idName)
			if len(subParts) >= 2 {
				appID, err := strconv.Atoi(subParts[0])
				if err != nil {
					continue
				}
				appName := subParts[1]
				classID := 0
				className := ""
				classZh := ""
				if currentCat != nil {
					classID = currentCat.ClassID
					className = currentCat.ClassName
					classZh = currentCat.ClassZh
				}

				app := models.AppInfo{
					ID:        appID,
					Name:      appName,
					ClassID:   classID,
					ClassName: className,
					ClassZh:   classZh,
					IsCustom:  false,
				}
				m.appMap[appID] = app
				if currentCat != nil {
					currentCat.Apps = append(currentCat.Apps, app)
				}
			}
		}
	}

	// Organize categories slice
	m.categories = make([]models.AppCategory, 0, len(categoriesMap))
	for _, cat := range categoriesMap {
		m.categories = append(m.categories, *cat)
	}

	log.Printf("[DPI] Successfully loaded %d categories and %d apps from %s", len(m.categories), len(m.appMap), m.featurePath)
}

func getCategoryIcon(className string) string {
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

// loadFallbackFeatures provides built-in fallback signatures when feature file is missing
func (m *DPIManager) loadFallbackFeatures() {
	gameCat := models.AppCategory{
		ClassID:   2,
		ClassName: "game",
		ClassZh:   "游戏",
		Icon:      "gamepad-2",
		Apps: []models.AppInfo{
			{ID: 2001, Name: "王者荣耀", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2002, Name: "和平精英", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2023, Name: "原神", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2015, Name: "我的世界", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2035, Name: "英雄联盟", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2040, Name: "蛋仔派对", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2050, Name: "Roblox", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2060, Name: "Steam", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
		},
	}
	videoCat := models.AppCategory{
		ClassID:   3,
		ClassName: "video",
		ClassZh:   "短视频/影视",
		Icon:      "play-circle",
		Apps: []models.AppInfo{
			{ID: 3001, Name: "抖音", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
			{ID: 3009, Name: "快手", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
			{ID: 3010, Name: "小红书", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
			{ID: 3014, Name: "哔哩哔哩", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
			{ID: 3004, Name: "爱奇艺", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
			{ID: 3003, Name: "腾讯视频", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
			{ID: 3005, Name: "优酷", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
			{ID: 3020, Name: "YouTube", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
			{ID: 3025, Name: "TikTok", ClassID: 3, ClassName: "video", ClassZh: "短视频/影视"},
		},
	}
	chatCat := models.AppCategory{
		ClassID:   1,
		ClassName: "chat",
		ClassZh:   "社交/聊天",
		Icon:      "message-square",
		Apps: []models.AppInfo{
			{ID: 1001, Name: "QQ", ClassID: 1, ClassName: "chat", ClassZh: "社交/聊天"},
			{ID: 1002, Name: "微信", ClassID: 1, ClassName: "chat", ClassZh: "社交/聊天"},
			{ID: 1005, Name: "微博", ClassID: 1, ClassName: "chat", ClassZh: "社交/聊天"},
			{ID: 1010, Name: "Discord", ClassID: 1, ClassName: "chat", ClassZh: "社交/聊天"},
		},
	}

	m.categories = []models.AppCategory{chatCat, gameCat, videoCat}
	for _, cat := range m.categories {
		for _, app := range cat.Apps {
			m.appMap[app.ID] = app
		}
	}
}

// LoadCustomData loads persisted custom apps and categories
func (m *DPIManager) LoadCustomData(apps []models.AppInfo, cats []models.AppCategory) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.customApps = make(map[int]models.AppInfo)
	m.customCats = make(map[int]models.AppCategory)

	for _, c := range cats {
		c.IsCustom = true
		if c.Icon == "" {
			c.Icon = getCategoryIcon(c.ClassName)
		}
		m.customCats[c.ClassID] = c
	}

	for _, a := range apps {
		a.IsCustom = true
		m.customApps[a.ID] = a
		m.appMap[a.ID] = a
	}

	m.rebuildCategoriesLocked()
}

// GetCustomData retrieves user-defined custom apps and categories
func (m *DPIManager) GetCustomData() ([]models.AppInfo, []models.AppCategory) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	apps := make([]models.AppInfo, 0, len(m.customApps))
	for _, a := range m.customApps {
		apps = append(apps, a)
	}

	cats := make([]models.AppCategory, 0, len(m.customCats))
	for _, c := range m.customCats {
		cats = append(cats, c)
	}
	return apps, cats
}

// rebuildCategoriesLocked rebuilds the categories slice (must be called with lock held)
func (m *DPIManager) rebuildCategoriesLocked() {
	catMap := make(map[int]*models.AppCategory)

	// 1. Add base categories
	for i := range m.categories {
		cat := m.categories[i]
		cat.Apps = make([]models.AppInfo, 0)
		catMap[cat.ClassID] = &cat
	}

	// 2. Add custom categories
	for _, c := range m.customCats {
		if _, exists := catMap[c.ClassID]; !exists {
			cp := c
			cp.Apps = make([]models.AppInfo, 0)
			catMap[c.ClassID] = &cp
		}
	}

	// 3. Classify all apps into categories
	for _, app := range m.appMap {
		cat, ok := catMap[app.ClassID]
		if !ok {
			// Fallback to "other" category if not found
			otherCat, exists := catMap[99]
			if !exists {
				other := models.AppCategory{
					ClassID:   99,
					ClassName: "other",
					ClassZh:   "其他/自定义",
					Icon:      "grid",
					Apps:      make([]models.AppInfo, 0),
				}
				catMap[99] = &other
				otherCat = &other
			}
			app.ClassID = otherCat.ClassID
			app.ClassName = otherCat.ClassName
			app.ClassZh = otherCat.ClassZh
			cat = otherCat
		} else {
			app.ClassName = cat.ClassName
			app.ClassZh = cat.ClassZh
		}
		cat.Apps = append(cat.Apps, app)
	}

	// 4. Sort categories and their apps
	result := make([]models.AppCategory, 0, len(catMap))
	for _, cat := range catMap {
		sort.Slice(cat.Apps, func(i, j int) bool {
			return cat.Apps[i].ID < cat.Apps[j].ID
		})
		result = append(result, *cat)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ClassID < result[j].ClassID
	})

	m.categories = result
}

// GetCategories returns all categories and apps
func (m *DPIManager) GetCategories() []models.AppCategory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.categories
}

// GetAllApps returns a flattened list of all apps
func (m *DPIManager) GetAllApps() []models.AppInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	apps := make([]models.AppInfo, 0, len(m.appMap))
	for _, a := range m.appMap {
		apps = append(apps, a)
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].ID < apps[j].ID
	})
	return apps
}

// GetApp returns a single app by ID
func (m *DPIManager) GetApp(id int) (models.AppInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	app, ok := m.appMap[id]
	return app, ok
}

// AddApp adds a new managed app
func (m *DPIManager) AddApp(app models.AppInfo) (models.AppInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.TrimSpace(app.Name) == "" {
		return app, fmt.Errorf("app name cannot be empty")
	}

	// If ID not specified or conflicts, generate an auto ID (> 5000)
	if app.ID <= 0 || m.appMap[app.ID].ID != 0 {
		maxID := 5000
		for id := range m.appMap {
			if id >= maxID {
				maxID = id + 1
			}
		}
		app.ID = maxID
	}

	// Lookup category name
	if app.ClassID <= 0 {
		app.ClassID = 2 // Default game
	}
	for _, cat := range m.categories {
		if cat.ClassID == app.ClassID {
			app.ClassName = cat.ClassName
			app.ClassZh = cat.ClassZh
			break
		}
	}
	if app.ClassZh == "" {
		app.ClassZh = "Custom"
		app.ClassName = "custom"
	}

	app.IsCustom = true
	m.appMap[app.ID] = app
	m.customApps[app.ID] = app
	m.rebuildCategoriesLocked()

	log.Printf("[DPI] Added custom app: [%d] %s (%s)", app.ID, app.Name, app.ClassZh)
	return app, nil
}

// UpdateApp updates existing app information
func (m *DPIManager) UpdateApp(id int, updated models.AppInfo) (models.AppInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.appMap[id]
	if !ok {
		return updated, fmt.Errorf("app with ID %d not found", id)
	}

	if strings.TrimSpace(updated.Name) != "" {
		existing.Name = strings.TrimSpace(updated.Name)
	}
	if updated.ClassID > 0 {
		existing.ClassID = updated.ClassID
		for _, cat := range m.categories {
			if cat.ClassID == existing.ClassID {
				existing.ClassName = cat.ClassName
				existing.ClassZh = cat.ClassZh
				break
			}
		}
	}
	if updated.Description != "" {
		existing.Description = updated.Description
	}

	existing.IsCustom = true
	m.appMap[id] = existing
	m.customApps[id] = existing
	m.rebuildCategoriesLocked()

	log.Printf("[DPI] Updated app: [%d] %s (%s)", existing.ID, existing.Name, existing.ClassZh)
	return existing, nil
}

// DeleteApp removes an app by ID
func (m *DPIManager) DeleteApp(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.appMap[id]; !ok {
		return fmt.Errorf("app with ID %d not found", id)
	}

	delete(m.appMap, id)
	delete(m.customApps, id)
	m.rebuildCategoriesLocked()

	log.Printf("[DPI] Deleted app: [%d]", id)
	return nil
}

// AddCategory adds a new app category
func (m *DPIManager) AddCategory(cat models.AppCategory) (models.AppCategory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.TrimSpace(cat.ClassZh) == "" && strings.TrimSpace(cat.ClassName) == "" {
		return cat, fmt.Errorf("category name cannot be empty")
	}
	if strings.TrimSpace(cat.ClassZh) == "" {
		cat.ClassZh = cat.ClassName
	}

	if cat.ClassID <= 0 {
		maxClassID := 10
		for _, c := range m.categories {
			if c.ClassID >= maxClassID {
				maxClassID = c.ClassID + 1
			}
		}
		cat.ClassID = maxClassID
	}

	if cat.ClassName == "" {
		cat.ClassName = fmt.Sprintf("cat_%d", cat.ClassID)
	}
	if cat.Icon == "" {
		cat.Icon = getCategoryIcon(cat.ClassName)
	}
	cat.IsCustom = true
	cat.Apps = make([]models.AppInfo, 0)

	m.customCats[cat.ClassID] = cat
	m.rebuildCategoriesLocked()

	log.Printf("[DPI] Added custom category: [%d] %s", cat.ClassID, cat.ClassZh)
	return cat, nil
}

// DeleteCategory removes an app category by ID
func (m *DPIManager) DeleteCategory(classID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.customCats, classID)
	// Reassign apps belonging to this category to "other"
	for id, app := range m.appMap {
		if app.ClassID == classID {
			app.ClassID = 99
			app.ClassName = "other"
			app.ClassZh = "其他/自定义"
			m.appMap[id] = app
			if _, isCustom := m.customApps[id]; isCustom {
				m.customApps[id] = app
			}
		}
	}

	m.rebuildCategoriesLocked()
	log.Printf("[DPI] Deleted category: [%d]", classID)
	return nil
}

// ApplyRules applies blocked App IDs and managed MACs to the kernel
func (m *DPIManager) ApplyRules(appIDs []int, macList []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isOAFInstalled {
		log.Println("[DPI] OAF module not installed, skipping kernel rules apply.")
		return nil
	}

	// 1. Clear previous rules: op 3
	if err := m.sendOAFCommand(map[string]interface{}{
		"op":   3,
		"data": map[string]interface{}{},
	}); err != nil {
		log.Printf("[DPI] Failed to clean OAF rules: %v", err)
	}

	// If no apps or MACs to block, return early
	if len(appIDs) == 0 || len(macList) == 0 {
		log.Println("[DPI] No apps or MACs to block.")
		return nil
	}

	// 2. Load App ID rules: op 1
	if err := m.sendOAFCommand(map[string]interface{}{
		"op": 1,
		"data": map[string]interface{}{
			"apps": appIDs,
		},
	}); err != nil {
		return fmt.Errorf("failed to load app rules: %w", err)
	}

	// 3. Load MAC list: op 4
	if err := m.sendOAFCommand(map[string]interface{}{
		"op": 4,
		"data": map[string]interface{}{
			"mac_list": macList,
		},
	}); err != nil {
		return fmt.Errorf("failed to load mac list: %w", err)
	}

	// Set operational mode to 0 (blacklist mode)
	_ = os.WriteFile("/proc/sys/oaf/work_mode", []byte("0\n"), 0644)

	log.Printf("[DPI] Applied %d blocked apps on %d devices via kmod-oaf", len(appIDs), len(macList))
	return nil
}

// sendOAFCommand sends JSON control command to /dev/appfilter
func (m *DPIManager) sendOAFCommand(payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	dev, err := os.OpenFile("/dev/appfilter", os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer dev.Close()

	_, err = dev.Write(data)
	return err
}
