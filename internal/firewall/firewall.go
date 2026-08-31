package firewall

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// FirewallManager manages iptables rules across mangle, filter, and nat tables
type FirewallManager struct {
	mu          sync.Mutex
	blockedMACs map[string]bool
	blockDoHDoT bool
	redirectDNS bool
}

const (
	ChainManglePre  = "PARENT_CONTROL_MANGLE_PRE"
	ChainFilterFwd  = "PARENT_CONTROL_FWD"
	ChainFilterIn   = "PARENT_CONTROL_INPUT"
	ChainFilterAcct = "PARENT_CONTROL_ACCT"
	ChainNatPre     = "PARENT_CONTROL_NAT_PRE"
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

	log.Println("[Firewall] Initializing multi-layer parent control iptables chains...")

	// 1. Filter ACCT: Low-overhead monotonic byte accounting for LAN devices
	fm.ensureCustomChain("filter", ChainFilterAcct, "FORWARD")

	// 2. Mangle PREROUTING: Intercepts before OpenClash, Passwall, or any TProxy/NAT
	fm.ensureCustomChain("mangle", ChainManglePre, "PREROUTING")

	// 3. Filter INPUT: Blocks traffic targeting router ports
	fm.ensureCustomChain("filter", ChainFilterIn, "INPUT")

	// 4. Filter FORWARD: Standard forwarding block & DoH/DoT intercept
	fm.ensureCustomChain("filter", ChainFilterFwd, "FORWARD")

	// 5. Nat PREROUTING: Port 53 DNS redirection
	fm.ensureCustomChain("nat", ChainNatPre, "PREROUTING")

	// 6. Apply baseline security rules
	fm.applyBaseRules()

	return nil
}

// ensureCustomChain ensures the custom chain exists and is inserted at position 1
func (fm *FirewallManager) ensureCustomChain(table, chain, parentChain string) {
	// 1. Create chain if not existing
	_ = exec.Command("iptables", "-t", table, "-N", chain).Run()

	// 2. Remove all old references to prevent duplicates
	for {
		err := exec.Command("iptables", "-t", table, "-D", parentChain, "-j", chain).Run()
		if err != nil {
			break
		}
	}

	// 3. Insert at position 1 (highest priority before other services like OpenClash)
	_ = exec.Command("iptables", "-t", table, "-I", parentChain, "1", "-j", chain).Run()
}

// applyBaseRules applies anti-bypass firewall rules
func (fm *FirewallManager) applyBaseRules() {
	// Flush rules within custom chains
	_ = exec.Command("iptables", "-t", "mangle", "-F", ChainManglePre).Run()
	_ = exec.Command("iptables", "-t", "filter", "-F", ChainFilterIn).Run()
	_ = exec.Command("iptables", "-t", "filter", "-F", ChainFilterFwd).Run()
	_ = exec.Command("iptables", "-t", "nat", "-F", ChainNatPre).Run()

	if fm.redirectDNS {
		// Force all outbound DNS (UDP/TCP 53) redirected to local port 53
		_ = exec.Command("iptables", "-t", "nat", "-A", ChainNatPre, "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "53").Run()
		_ = exec.Command("iptables", "-t", "nat", "-A", ChainNatPre, "-p", "tcp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "53").Run()
	}

	if fm.blockDoHDoT {
		// Block DoT (port 853)
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainFilterFwd, "-p", "tcp", "--dport", "853", "-j", "REJECT").Run()
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainFilterFwd, "-p", "udp", "--dport", "853", "-j", "REJECT").Run()

		// Block major public DoH server IPs
		dohIPs := []string{
			"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4",
			"9.9.9.9", "149.112.112.112", "208.67.222.222", "208.67.220.220",
		}
		for _, ip := range dohIPs {
			_ = exec.Command("iptables", "-t", "filter", "-A", ChainFilterFwd, "-d", ip, "-p", "tcp", "--dport", "443", "-j", "REJECT").Run()
		}
	}
}

// SyncBlockedMACs synchronizes the list of MAC addresses that should be completely blocked
func (fm *FirewallManager) SyncBlockedMACs(macs []string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Ensure top-level insertion before rebuilding rules
	fm.ensureCustomChain("mangle", ChainManglePre, "PREROUTING")
	fm.ensureCustomChain("filter", ChainFilterIn, "INPUT")
	fm.ensureCustomChain("filter", ChainFilterFwd, "FORWARD")

	// Flush existing block rules and rebuild baseline
	fm.applyBaseRules()

	fm.blockedMACs = make(map[string]bool)
	for _, mac := range macs {
		mac = strings.ToUpper(strings.TrimSpace(mac))
		if mac == "" {
			continue
		}
		fm.blockedMACs[mac] = true

		// 1. Mangle PREROUTING: Drop at the very first ingress stage (bypasses OpenClash/Passwall/TProxy)
		_ = exec.Command("iptables", "-t", "mangle", "-A", ChainManglePre, "-m", "mac", "--mac-source", mac, "-j", "DROP").Run()

		// 2. Filter INPUT: Prevent local router access (except DHCP request/ack)
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainFilterIn, "-m", "mac", "--mac-source", mac, "-p", "udp", "--dport", "67:68", "--sport", "67:68", "-j", "ACCEPT").Run()
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainFilterIn, "-m", "mac", "--mac-source", mac, "-j", "DROP").Run()

		// 3. Filter FORWARD: Fallback drop
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainFilterFwd, "-m", "mac", "--mac-source", mac, "-j", "DROP").Run()
	}

	log.Printf("[Firewall] Synced %d blocked MACs across mangle/filter tables.", len(fm.blockedMACs))
	return nil
}

type IPByteStats struct {
	RxBytes uint64
	TxBytes uint64
}

// SyncAccountingIPs ensures accounting rules exist for managed or active LAN IPs
func (fm *FirewallManager) SyncAccountingIPs(ips []string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Ensure ChainFilterAcct exists and is inserted at FORWARD
	fm.ensureCustomChain("filter", ChainFilterAcct, "FORWARD")

	_ = exec.Command("iptables", "-t", "filter", "-F", ChainFilterAcct).Run()
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" || !strings.HasPrefix(ip, "192.168.") && !strings.HasPrefix(ip, "10.") && !strings.HasPrefix(ip, "172.") {
			continue
		}
		// Inbound (Download / Rx for LAN device)
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainFilterAcct, "-d", ip, "-j", "RETURN").Run()
		// Outbound (Upload / Tx for LAN device)
		_ = exec.Command("iptables", "-t", "filter", "-A", ChainFilterAcct, "-s", ip, "-j", "RETURN").Run()
	}
}

// ReadAccountingBytes parses byte counters from PARENT_CONTROL_ACCT iptables chain
func (fm *FirewallManager) ReadAccountingBytes() map[string]IPByteStats {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	out, err := exec.Command("iptables", "-t", "filter", "-L", ChainFilterAcct, "-v", "-n", "-x").Output()
	if err != nil {
		return nil
	}

	result := make(map[string]IPByteStats)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		// Expected format: pkts bytes target prot opt in out source destination
		if len(fields) < 9 {
			continue
		}
		bytesVal, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		src := fields[7]
		dst := fields[8]

		if strings.HasPrefix(dst, "192.168.") || strings.HasPrefix(dst, "10.") || strings.HasPrefix(dst, "172.") {
			st := result[dst]
			st.RxBytes += bytesVal
			result[dst] = st
		} else if strings.HasPrefix(src, "192.168.") || strings.HasPrefix(src, "10.") || strings.HasPrefix(src, "172.") {
			st := result[src]
			st.TxBytes += bytesVal
			result[src] = st
		}
	}

	return result
}

// Cleanup removes all parent control firewall rules and chains
func (fm *FirewallManager) Cleanup() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Remove references from main chains
	_ = exec.Command("iptables", "-t", "mangle", "-D", "PREROUTING", "-j", ChainManglePre).Run()
	_ = exec.Command("iptables", "-t", "filter", "-D", "INPUT", "-j", ChainFilterIn).Run()
	_ = exec.Command("iptables", "-t", "filter", "-D", "FORWARD", "-j", ChainFilterFwd).Run()
	_ = exec.Command("iptables", "-t", "filter", "-D", "FORWARD", "-j", ChainFilterAcct).Run()
	_ = exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-j", ChainNatPre).Run()

	// Flush and delete custom chains
	_ = exec.Command("iptables", "-t", "mangle", "-F", ChainManglePre).Run()
	_ = exec.Command("iptables", "-t", "mangle", "-X", ChainManglePre).Run()

	_ = exec.Command("iptables", "-t", "filter", "-F", ChainFilterIn).Run()
	_ = exec.Command("iptables", "-t", "filter", "-X", ChainFilterIn).Run()

	_ = exec.Command("iptables", "-t", "filter", "-F", ChainFilterFwd).Run()
	_ = exec.Command("iptables", "-t", "filter", "-X", ChainFilterFwd).Run()

	_ = exec.Command("iptables", "-t", "filter", "-F", ChainFilterAcct).Run()
	_ = exec.Command("iptables", "-t", "filter", "-X", ChainFilterAcct).Run()

	_ = exec.Command("iptables", "-t", "nat", "-F", ChainNatPre).Run()
	_ = exec.Command("iptables", "-t", "nat", "-X", ChainNatPre).Run()

	log.Println("[Firewall] Cleaned up all iptables parent control chains.")
}
