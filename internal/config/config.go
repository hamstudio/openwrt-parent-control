package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"parentcontrol/internal/models"
)

type AppConfig struct {
	Settings         models.GlobalSettings `json:"settings"`
	Members          []models.Member       `json:"members"`
	CustomApps       []models.AppInfo      `json:"custom_apps,omitempty"`
	CustomCategories []models.AppCategory  `json:"custom_categories,omitempty"`
}

type ConfigStore struct {
	mu       sync.Mutex
	filePath string
	Data     AppConfig
}

func NewConfigStore(filePath string) *ConfigStore {
	if filePath == "" {
		filePath = "/etc/parentcontrol/config.json"
	}
	store := &ConfigStore{
		filePath: filePath,
		Data: AppConfig{
			Settings: models.GlobalSettings{
				Enabled:           true,
				WebPort:           8088,
				EnforceSafeSearch: true,
				BlockDoHDoT:       true,
				IsolateNewDevices: false,
				DailyResetHour:    0,
				CustomBlacklist:   []string{},
				CustomWhitelist:   []string{},
			},
			Members:          []models.Member{},
			CustomApps:       []models.AppInfo{},
			CustomCategories: []models.AppCategory{},
		},
	}
	store.Load()
	return store
}

func (cs *ConfigStore) Load() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	data, err := os.ReadFile(cs.filePath)
	if err != nil {
		log.Printf("[Config] Config file %s not found. Creating default config.", cs.filePath)
		cs.saveUnsafe()
		return
	}

	var conf AppConfig
	if err := json.Unmarshal(data, &conf); err != nil {
		log.Printf("[Config] Failed to parse config JSON: %v. Using defaults.", err)
		return
	}

	cs.Data = conf
	log.Printf("[Config] Loaded configuration with %d members from %s.", len(cs.Data.Members), cs.filePath)
}

func (cs *ConfigStore) Save() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.saveUnsafe()
}

func (cs *ConfigStore) saveUnsafe() error {
	dir := filepath.Dir(cs.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cs.Data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cs.filePath, data, 0644)
}
