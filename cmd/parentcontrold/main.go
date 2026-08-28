package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"parentcontrol/internal/api"
	"parentcontrol/internal/cloud"
	"parentcontrol/internal/config"
	"parentcontrol/internal/device"
	"parentcontrol/internal/dpi"
	"parentcontrol/internal/firewall"
	"parentcontrol/internal/quota"
	"parentcontrol/internal/safedns"
	"parentcontrol/internal/stats"
	"parentcontrol/internal/tz"
	"parentcontrol/web"
)

func main() {
	configPath := flag.String("config", "/etc/parentcontrol/config.json", "Path to config file")
	statsPath := flag.String("stats", "/etc/parentcontrol/stats.json", "Path to stats database file")
	featurePath := flag.String("feature", "/etc/appfilter/feature_cn.cfg", "Path to OAF feature file")
	port := flag.Int("port", 8088, "HTTP Web/API port")
	flag.Parse()

	log.Println("==================================================")
	log.Println("  ParentControl Daemon (Go + kmod-oaf DPI Engine)")
	log.Println("==================================================")

	// 0. Automatically detect and apply router system timezone
	tz.DetectAndApplyTimezone()
	zoneName, offset := tz.GetTimezoneInfo()
	log.Printf("[Main] System timezone: %s (%s, offset: %d min, local time: %s)", 
		tz.GetCurrentZonename(), zoneName, offset/60, tz.Now().Format("2006-01-02 15:04:05 MST"))

	// 1. Initialize configuration
	cfgStore := config.NewConfigStore(*configPath)

	// 2. Initialize DPI engine
	dpiMgr := dpi.NewDPIManager(*featurePath)
	if len(cfgStore.Data.CustomApps) > 0 || len(cfgStore.Data.CustomCategories) > 0 {
		dpiMgr.LoadCustomData(cfgStore.Data.CustomApps, cfgStore.Data.CustomCategories)
	}

	// 3. Initialize firewall manager
	fwMgr := firewall.NewFirewallManager()
	if err := fwMgr.Init(); err != nil {
		log.Printf("[Main] Warning: Firewall init returned: %v", err)
	}

	// 4. Initialize SafeDNS
	dnsMgr := safedns.NewSafeDNSManager("/tmp/dnsmasq.d/parentcontrol.conf")
	_ = dnsMgr.ApplyConfig(
		cfgStore.Data.Settings.EnforceSafeSearch,
		true,
		cfgStore.Data.Settings.CustomBlacklist,
		cfgStore.Data.Settings.CustomWhitelist,
	)

	// 5. Initialize device tracker
	devTracker := device.NewDeviceTracker()

	// 5.1 Initialize statistical usage tracker
	statsTracker := stats.NewStatsTracker(*statsPath, dpiMgr)

	// 6. Initialize policy and quota engine
	engine := quota.NewPolicyEngine(fwMgr, dpiMgr, devTracker)
	engine.SetStatsTracker(statsTracker)
	engine.UpdateSettings(cfgStore.Data.Settings)

	// Load existing members
	for _, m := range cfgStore.Data.Members {
		engine.SetMember(m)
	}

	// Start policy background evaluation loop
	engine.Start()

	// 7. Register signal handler (graceful shutdown and firewall cleanup)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("[Main] Caught signal %v, cleaning up and exiting...", sig)
		engine.Stop()
		_ = statsTracker.Save()
		fwMgr.Cleanup()
		dnsMgr.Cleanup()
		os.Exit(0)
	}()

	// 8. Start Cloudflare Worker cloud syncer
	syncer := cloud.NewSyncer(engine, devTracker, dpiMgr, cfgStore)
	syncer.Start(context.Background())

	// 9. Start Web console and API
	webPort := cfgStore.Data.Settings.WebPort
	if *port != 8088 {
		webPort = *port
	}
	if webPort == 0 {
		webPort = 8088
	}

	server := api.NewServer(engine, dpiMgr, fwMgr, dnsMgr, devTracker, cfgStore, web.StaticFS)
	server.SetStatsTracker(statsTracker)
	if err := server.Start(webPort); err != nil {
		log.Fatalf("[Main] HTTP server failed: %v", err)
	}
}
