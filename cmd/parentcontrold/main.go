package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"parentcontrol/internal/api"
	"parentcontrol/internal/config"
	"parentcontrol/internal/device"
	"parentcontrol/internal/dpi"
	"parentcontrol/internal/firewall"
	"parentcontrol/internal/quota"
	"parentcontrol/internal/safedns"
	"parentcontrol/web"
)

func main() {
	configPath := flag.String("config", "/etc/parentcontrol/config.json", "Path to config file")
	featurePath := flag.String("feature", "/etc/appfilter/feature_cn.cfg", "Path to OAF feature file")
	port := flag.Int("port", 8088, "HTTP Web/API port")
	flag.Parse()

	log.Println("==================================================")
	log.Println("  ParentControl Daemon (Go + kmod-oaf DPI Engine)")
	log.Println("==================================================")

	// 1. 初始化配置
	cfgStore := config.NewConfigStore(*configPath)

	// 2. 初始化 DPI 引擎
	dpiMgr := dpi.NewDPIManager(*featurePath)
	if len(cfgStore.Data.CustomApps) > 0 || len(cfgStore.Data.CustomCategories) > 0 {
		dpiMgr.LoadCustomData(cfgStore.Data.CustomApps, cfgStore.Data.CustomCategories)
	}

	// 3. 初始化防火墙管理器
	fwMgr := firewall.NewFirewallManager()
	if err := fwMgr.Init(); err != nil {
		log.Printf("[Main] Warning: Firewall init returned: %v", err)
	}

	// 4. 初始化 SafeDNS
	dnsMgr := safedns.NewSafeDNSManager("/tmp/dnsmasq.d/parentcontrol.conf")
	_ = dnsMgr.ApplyConfig(
		cfgStore.Data.Settings.EnforceSafeSearch,
		true,
		cfgStore.Data.Settings.CustomBlacklist,
		cfgStore.Data.Settings.CustomWhitelist,
	)

	// 5. 初始化设备追踪器
	devTracker := device.NewDeviceTracker()

	// 6. 初始化策略与配额引擎
	engine := quota.NewPolicyEngine(fwMgr, dpiMgr, devTracker)
	engine.UpdateSettings(cfgStore.Data.Settings)

	// 载入已有成员
	for _, m := range cfgStore.Data.Members {
		engine.SetMember(m)
	}

	// 启动策略后台循环
	engine.Start()

	// 7. 注册信号处理 (优雅关机清理防火墙)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("[Main] Caught signal %v, cleaning up and exiting...", sig)
		engine.Stop()
		fwMgr.Cleanup()
		dnsMgr.Cleanup()
		os.Exit(0)
	}()

	// 8. 启动 Web 控制台与 API
	webPort := cfgStore.Data.Settings.WebPort
	if *port != 8088 {
		webPort = *port
	}
	if webPort == 0 {
		webPort = 8088
	}

	server := api.NewServer(engine, dpiMgr, fwMgr, dnsMgr, devTracker, cfgStore, web.StaticFS)
	if err := server.Start(webPort); err != nil {
		log.Fatalf("[Main] HTTP server failed: %v", err)
	}
}
