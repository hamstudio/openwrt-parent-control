package dpi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"parentcontrol/internal/models"
)

// DPIManager 管理深度应用识别与拦截
type DPIManager struct {
	mu             sync.RWMutex
	featurePath    string
	categories     []models.AppCategory
	appMap         map[int]models.AppInfo
	blockedAppIDs  map[int]bool
	managedMACs    map[string]bool
	isOAFInstalled bool
}

// NewDPIManager 初始化 DPI 管理器
func NewDPIManager(featurePath string) *DPIManager {
	if featurePath == "" {
		featurePath = "/etc/appfilter/feature_cn.cfg"
	}
	mgr := &DPIManager{
		featurePath:   featurePath,
		categories:    make([]models.AppCategory, 0),
		appMap:        make(map[int]models.AppInfo),
		blockedAppIDs: make(map[int]bool),
		managedMACs:   make(map[string]bool),
	}
	mgr.Init()
	return mgr
}

// Init 初始化内核模块与特征库
func (m *DPIManager) Init() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 检查并尝试加载 kmod-oaf
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

	// 2. 加载特征库
	m.loadFeatures()
}

// IsReady 检查 DPI 模块是否就绪
func (m *DPIManager) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isOAFInstalled
}

// loadFeatures 从配置文件解析应用特征与分类
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

		// 分类行: #class chat 1 聊天
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

		// 应用行: 1001 QQ:[tcp;;;;;00:02|-1:03...]
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
				}
				m.appMap[appID] = app
				if currentCat != nil {
					currentCat.Apps = append(currentCat.Apps, app)
				}
			}
		}
	}

	// 整理分类切片
	m.categories = make([]models.AppCategory, 0, len(categoriesMap))
	for _, cat := range categoriesMap {
		m.categories = append(m.categories, *cat)
	}

	log.Printf("[DPI] Successfully loaded %d categories and %d apps from %s", len(m.categories), len(m.appMap), m.featurePath)
}

func getCategoryIcon(className string) string {
	switch className {
	case "chat":
		return "chat"
	case "game":
		return "gamepad"
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

// loadFallbackFeatures 内置降级基础特征
func (m *DPIManager) loadFallbackFeatures() {
	gameCat := models.AppCategory{
		ClassID:   2,
		ClassName: "game",
		ClassZh:   "游戏",
		Icon:      "gamepad",
		Apps: []models.AppInfo{
			{ID: 2001, Name: "王者荣耀", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2002, Name: "和平精英", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2023, Name: "原神", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2015, Name: "我的世界", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
			{ID: 2035, Name: "英雄联盟", ClassID: 2, ClassName: "game", ClassZh: "游戏"},
		},
	}
	videoCat := models.AppCategory{
		ClassID:   3,
		ClassName: "video",
		ClassZh:   "视频",
		Icon:      "play-circle",
		Apps: []models.AppInfo{
			{ID: 3001, Name: "抖音", ClassID: 3, ClassName: "video", ClassZh: "视频"},
			{ID: 3009, Name: "快手", ClassID: 3, ClassName: "video", ClassZh: "视频"},
			{ID: 3010, Name: "小红书", ClassID: 3, ClassName: "video", ClassZh: "视频"},
			{ID: 3014, Name: "哔哩哔哩", ClassID: 3, ClassName: "video", ClassZh: "视频"},
			{ID: 3004, Name: "爱奇艺", ClassID: 3, ClassName: "video", ClassZh: "视频"},
			{ID: 3003, Name: "腾讯视频", ClassID: 3, ClassName: "video", ClassZh: "视频"},
		},
	}
	chatCat := models.AppCategory{
		ClassID:   1,
		ClassName: "chat",
		ClassZh:   "聊天",
		Icon:      "chat",
		Apps: []models.AppInfo{
			{ID: 1001, Name: "QQ", ClassID: 1, ClassName: "chat", ClassZh: "聊天"},
			{ID: 1002, Name: "微信", ClassID: 1, ClassName: "chat", ClassZh: "聊天"},
		},
	}

	m.categories = []models.AppCategory{chatCat, gameCat, videoCat}
	for _, cat := range m.categories {
		for _, app := range cat.Apps {
			m.appMap[app.ID] = app
		}
	}
}

// GetCategories 获取所有分类与 App
func (m *DPIManager) GetCategories() []models.AppCategory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.categories
}

// ApplyRules 应用封禁应用 ID 列表与受管 MAC 列表至内核
func (m *DPIManager) ApplyRules(appIDs []int, macList []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isOAFInstalled {
		log.Println("[DPI] OAF module not installed, skipping kernel rules apply.")
		return nil
	}

	// 1. 清空旧规则: op 3
	if err := m.sendOAFCommand(map[string]interface{}{
		"op":   3,
		"data": map[string]interface{}{},
	}); err != nil {
		log.Printf("[DPI] Failed to clean OAF rules: %v", err)
	}

	// 如果没有要封禁的应用或受管 MAC，直接返回
	if len(appIDs) == 0 || len(macList) == 0 {
		log.Println("[DPI] No apps or MACs to block.")
		return nil
	}

	// 2. 加载 App ID 规则: op 1
	if err := m.sendOAFCommand(map[string]interface{}{
		"op": 1,
		"data": map[string]interface{}{
			"apps": appIDs,
		},
	}); err != nil {
		return fmt.Errorf("failed to load app rules: %w", err)
	}

	// 3. 加载 MAC 列表: op 4
	if err := m.sendOAFCommand(map[string]interface{}{
		"op": 4,
		"data": map[string]interface{}{
			"mac_list": macList,
		},
	}); err != nil {
		return fmt.Errorf("failed to load mac list: %w", err)
	}

	// 设置工作模式为 0 (黑名单模式)
	_ = os.WriteFile("/proc/sys/oaf/work_mode", []byte("0\n"), 0644)

	log.Printf("[DPI] Applied %d blocked apps on %d devices via kmod-oaf", len(appIDs), len(macList))
	return nil
}

// sendOAFCommand 向 /dev/appfilter 发送 JSON 命令
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
