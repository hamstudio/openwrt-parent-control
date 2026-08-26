package firewall

import (
	"log"
	"os/exec"
	"strings"
	"sync"
)

// FirewallManager 管理 iptables 规则与防火墙链
type FirewallManager struct {
	mu           sync.Mutex
	blockedMACs  map[string]bool
	blockDoHDoT  bool
	redirectDNS  bool
}

const (
	ChainForward = "PARENT_CONTROL_FWD"
	ChainNatPre  = "PARENT_CONTROL_NAT_PRE"
)

// NewFirewallManager 创建防火墙管理器
func NewFirewallManager() *FirewallManager {
	return &FirewallManager{
		blockedMACs: make(map[string]bool),
		blockDoHDoT: true,
		redirectDNS: true,
	}
}

// Init 初始化并挂载自定义 iptables 规则链
func (fm *FirewallManager) Init() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	log.Println("[Firewall] Initializing parent control iptables chains...")

	// 1. 创建并挂载 filter 表 FORWARD 链
	fm.ensureCustomChain("filter", ChainForward, "FORWARD")

	// 2. 创建并挂载 nat 表 PREROUTING 链
	fm.ensureCustomChain("nat", ChainNatPre, "PREROUTING")

	// 3. 应用基础安全规则（DNS 重定向 + DoH/DoT 阻断）
	fm.applyBaseRules()

	return nil
}

// ensureCustomChain 确保自定义链存在并挂载到主链顶部
func (fm *FirewallManager) ensureCustomChain(table, chain, parentChain string) {
	// 检查自定义链是否存在，不存在则创建
	_ = exec.Command("iptables", "-t", table, "-N", chain).Run()

	// 检查是否已经在 parentChain 中引用
	checkCmd := exec.Command("iptables", "-t", table, "-C", parentChain, "-j", chain)
	if err := checkCmd.Run(); err != nil {
		// 未挂载，将其插入到第一条
		_ = exec.Command("iptables", "-t", table, "-I", parentChain, "1", "-j", chain).Run()
	}
}

// applyBaseRules 应用防绕过规则
func (fm *FirewallManager) applyBaseRules() {
	// 清理链内规则
	_ = exec.Command("iptables", "-t", "nat", "-F", ChainNatPre).Run()
	_ = exec.Command("iptables", "-t", "filter", "-F", ChainForward).Run()

	if fm.redirectDNS {
		// 强制将所有发往外部的 DNS (UDP/TCP 53) 重定向到本地 53 端口
		_ = exec.Command("iptables", "-t", "nat", "-A", ChainNatPre, "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "53").Run()
		_ = exec.Command("iptables", "-t", "nat", "-A", ChainNatPre, "-p", "tcp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "53").Run()
	}

	if fm.blockDoHDoT {
		// 阻断 DoT (853)
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainForward, "-p", "tcp", "--dport", "853", "-j", "REJECT").Run()
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainForward, "-p", "udp", "--dport", "853", "-j", "REJECT").Run()

		// 阻断主流知名公共 DoH 节点（防止私自加密 DNS 绕过）
		dohIPs := []string{
			"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4",
			"9.9.9.9", "149.112.112.112", "208.67.222.222", "208.67.220.220",
		}
		for _, ip := range dohIPs {
			_ = exec.Command("iptables", "-t", "filter", "-A", ChainForward, "-d", ip, "-p", "tcp", "--dport", "443", "-j", "REJECT").Run()
		}
	}
}

// SyncBlockedMACs 同步当前需要完全切断网络的 MAC 列表
func (fm *FirewallManager) SyncBlockedMACs(macs []string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// 清理现有的阻断规则并重新构建
	fm.applyBaseRules()

	fm.blockedMACs = make(map[string]bool)
	for _, mac := range macs {
		mac = strings.ToUpper(strings.TrimSpace(mac))
		if mac == "" {
			continue
		}
		fm.blockedMACs[mac] = true

		// 针对该 MAC 添加 DROP 规则
		cmd := exec.Command("iptables", "-t", "filter", "-A", ChainForward, "-m", "mac", "--mac-source", mac, "-j", "DROP")
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("[Firewall] Failed to block MAC %s: %v, output: %s", mac, err, string(out))
		}
	}

	log.Printf("[Firewall] Synced %d blocked MACs in iptables filter forward chain.", len(fm.blockedMACs))
	return nil
}

// Cleanup 清理所有 parent control 规则
func (fm *FirewallManager) Cleanup() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// 从主链中移除引用
	_ = exec.Command("iptables", "-t", "filter", "-D", "FORWARD", "-j", ChainForward).Run()
	_ = exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-j", ChainNatPre).Run()

	// 清空并删除自定义链
	_ = exec.Command("iptables", "-t", "filter", "-F", ChainForward).Run()
	_ = exec.Command("iptables", "-t", "filter", "-X", ChainForward).Run()

	_ = exec.Command("iptables", "-t", "nat", "-F", ChainNatPre).Run()
	_ = exec.Command("iptables", "-t", "nat", "-X", ChainNatPre).Run()

	log.Println("[Firewall] Cleaned up all iptables parent control chains.")
}
