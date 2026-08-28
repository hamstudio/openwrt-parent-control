package firewall

import (
	"log"
	"os/exec"
	"strings"
	"sync"
)

// FirewallManager manages iptables rules and custom chains
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

// NewFirewallManager creates a new FirewallManager instance
func NewFirewallManager() *FirewallManager {
	return &FirewallManager{
		blockedMACs: make(map[string]bool),
		blockDoHDoT: true,
		redirectDNS: true,
	}
}

// Init initializes and mounts custom iptables rule chains
func (fm *FirewallManager) Init() error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	log.Println("[Firewall] Initializing parent control iptables chains...")

	// 1. Create and attach filter table FORWARD chain
	fm.ensureCustomChain("filter", ChainForward, "FORWARD")

	// 2. Create and attach nat table PREROUTING chain
	fm.ensureCustomChain("nat", ChainNatPre, "PREROUTING")

	// 3. Apply baseline security rules (DNS redirect + DoH/DoT blocking)
	fm.applyBaseRules()

	return nil
}

// ensureCustomChain ensures the custom chain exists and is inserted at the top of parent chain
func (fm *FirewallManager) ensureCustomChain(table, chain, parentChain string) {
	// Check if custom chain exists; create if not
	_ = exec.Command("iptables", "-t", table, "-N", chain).Run()

	// Check if already referenced in parentChain
	checkCmd := exec.Command("iptables", "-t", table, "-C", parentChain, "-j", chain)
	if err := checkCmd.Run(); err != nil {
		// Not attached, insert at position 1
		_ = exec.Command("iptables", "-t", table, "-I", parentChain, "1", "-j", chain).Run()
	}
}

// applyBaseRules applies anti-bypass firewall rules
func (fm *FirewallManager) applyBaseRules() {
	// Flush rules within custom chains
	_ = exec.Command("iptables", "-t", "nat", "-F", ChainNatPre).Run()
	_ = exec.Command("iptables", "-t", "filter", "-F", ChainForward).Run()

	if fm.redirectDNS {
		// Force all outbound DNS (UDP/TCP 53) redirected to local port 53
		_ = exec.Command("iptables", "-t", "nat", "-A", ChainNatPre, "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "53").Run()
		_ = exec.Command("iptables", "-t", "nat", "-A", ChainNatPre, "-p", "tcp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "53").Run()
	}

	if fm.blockDoHDoT {
		// Block DoT (port 853)
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainForward, "-p", "tcp", "--dport", "853", "-j", "REJECT").Run()
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainForward, "-p", "udp", "--dport", "853", "-j", "REJECT").Run()

		// Block major public DoH server IPs (prevents bypassing local DNS filtering)
		dohIPs := []string{
			"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4",
			"9.9.9.9", "149.112.112.112", "208.67.222.222", "208.67.220.220",
		}
		for _, ip := range dohIPs {
			_ = exec.Command("iptables", "-t", "filter", "-A", ChainForward, "-d", ip, "-p", "tcp", "--dport", "443", "-j", "REJECT").Run()
		}
	}
}

// SyncBlockedMACs synchronizes the list of MAC addresses that should be completely blocked
func (fm *FirewallManager) SyncBlockedMACs(macs []string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Flush existing block rules and rebuild baseline
	fm.applyBaseRules()

	fm.blockedMACs = make(map[string]bool)
	for _, mac := range macs {
		mac = strings.ToUpper(strings.TrimSpace(mac))
		if mac == "" {
			continue
		}
		fm.blockedMACs[mac] = true

		// Add DROP rule for this MAC
		cmd := exec.Command("iptables", "-t", "filter", "-A", ChainForward, "-m", "mac", "--mac-source", mac, "-j", "DROP")
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("[Firewall] Failed to block MAC %s: %v, output: %s", mac, err, string(out))
		}
	}

	log.Printf("[Firewall] Synced %d blocked MACs in iptables filter forward chain.", len(fm.blockedMACs))
	return nil
}

// Cleanup removes all parent control firewall rules and chains
func (fm *FirewallManager) Cleanup() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Remove references from main chains
	_ = exec.Command("iptables", "-t", "filter", "-D", "FORWARD", "-j", ChainForward).Run()
	_ = exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-j", ChainNatPre).Run()

	// Flush and delete custom chains
	_ = exec.Command("iptables", "-t", "filter", "-F", ChainForward).Run()
	_ = exec.Command("iptables", "-t", "filter", "-X", ChainForward).Run()

	_ = exec.Command("iptables", "-t", "nat", "-F", ChainNatPre).Run()
	_ = exec.Command("iptables", "-t", "nat", "-X", ChainNatPre).Run()

	log.Println("[Firewall] Cleaned up all iptables parent control chains.")
}
