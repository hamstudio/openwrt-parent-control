package device

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"parentcontrol/internal/firewall"
	"parentcontrol/internal/models"
)

// DeviceTracker manages local network device discovery and traffic monitoring
type DeviceTracker struct {
	mu          sync.RWMutex
	devices     map[string]*models.Device // key: MAC
	fw          *firewall.FirewallManager
	prevRxBytes map[string]uint64
	prevTxBytes map[string]uint64
	lastSample  time.Time
}

// NewDeviceTracker creates a new DeviceTracker instance
func NewDeviceTracker() *DeviceTracker {
	dt := &DeviceTracker{
		devices:     make(map[string]*models.Device),
		prevRxBytes: make(map[string]uint64),
		prevTxBytes: make(map[string]uint64),
		lastSample:  time.Now(),
	}
	return dt
}

// SetFirewall attaches the firewall manager to enable kernel-level accurate traffic accounting
func (dt *DeviceTracker) SetFirewall(fw *firewall.FirewallManager) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.fw = fw
}

// ScanDevices scans all devices connected to the local network
func (dt *DeviceTracker) ScanDevices() []*models.Device {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	now := time.Now()

	// 1. Read /tmp/dhcp.leases
	// Format: 1724734800 00:11:22:33:44:55 192.168.0.150 iPhone-14 01:00:11:22:33:44:55
	if file, err := os.Open("/tmp/dhcp.leases"); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 4 {
				mac := strings.ToUpper(fields[1])
				ip := fields[2]
				hostname := fields[3]
				if hostname == "*" {
					hostname = "Unknown-Device"
				}

				dev, exists := dt.devices[mac]
				if !exists {
					dev = &models.Device{
						MAC:      mac,
						Vendor:   getVendorByMAC(mac),
						LastSeen: now,
					}
					dt.devices[mac] = dev
				}
				dev.IP = ip
				dev.Hostname = hostname
				dev.Online = true
				dev.LastSeen = now
			}
		}
		file.Close()
	}

	// 2. Read /proc/net/arp to supplement ARP table devices
	// Format: IP address HW type Flags HW address Mask Device
	if file, err := os.Open("/proc/net/arp"); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 4 && fields[0] != "IP" {
				ip := fields[0]
				flags := fields[2]
				mac := strings.ToUpper(fields[3])

				if mac == "00:00:00:00:00:00" || flags == "0x0" {
					continue
				}

				dev, exists := dt.devices[mac]
				if !exists {
					dev = &models.Device{
						MAC:      mac,
						Vendor:   getVendorByMAC(mac),
						Hostname: "Device-" + strings.ReplaceAll(ip, ".", "-"),
						LastSeen: now,
					}
					dt.devices[mac] = dev
				}
				dev.IP = ip
				dev.Online = true
				dev.LastSeen = now
			}
		}
		file.Close()
	}

	// 3. Calculate active traffic (count bytes from conntrack)
	dt.updateTrafficRates(now)

	// Return as slice
	list := make([]*models.Device, 0, len(dt.devices))
	for _, dev := range dt.devices {
		// If inactive for more than 10 minutes with no traffic, mark offline
		if now.Sub(dev.LastSeen) > 10*time.Minute {
			dev.Online = false
		}
		list = append(list, dev)
	}

	return list
}

// updateTrafficRates calculates current transfer rates for each device
func (dt *DeviceTracker) updateTrafficRates(now time.Time) {
	elapsed := now.Sub(dt.lastSample).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	currentRx := make(map[string]uint64)
	currentTx := make(map[string]uint64)

	// Collect active LAN IPs for accounting sync
	activeIPs := make([]string, 0, len(dt.devices))
	for _, d := range dt.devices {
		if d.IP != "" {
			activeIPs = append(activeIPs, d.IP)
		}
	}

	// 1. Try reading from kernel iptables PARENT_CONTROL_ACCT chain (strictly monotonic, most accurate)
	usedIPTables := false
	if dt.fw != nil {
		acctMap := dt.fw.ReadAccountingBytes()
		if len(acctMap) > 0 {
			usedIPTables = true
			for _, d := range dt.devices {
				if d.IP != "" {
					if stat, ok := acctMap[d.IP]; ok {
						currentRx[d.MAC] = stat.RxBytes
						currentTx[d.MAC] = stat.TxBytes
					}
				}
			}
		}
		// Periodically ensure accounting rules are present for all active LAN IPs
		dt.fw.SyncAccountingIPs(activeIPs)
	}

	// 2. Fallback to conntrack if iptables accounting not available
	if !usedIPTables {
		ipToDev := make(map[string]*models.Device)
		for _, d := range dt.devices {
			if d.IP != "" {
				ipToDev[d.IP] = d
			}
		}

		conntrackPath := "/proc/net/nf_conntrack"
		if _, err := os.Stat(conntrackPath); os.IsNotExist(err) {
			conntrackPath = "/proc/net/ip_conntrack"
		}

		if file, err := os.Open(conntrackPath); err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.Contains(line, "bytes=") {
					continue
				}

				fields := strings.Fields(line)
				var srcList, dstList []string
				var bytesList []uint64

				for _, f := range fields {
					if strings.HasPrefix(f, "src=") {
						srcList = append(srcList, strings.TrimPrefix(f, "src="))
					} else if strings.HasPrefix(f, "dst=") {
						dstList = append(dstList, strings.TrimPrefix(f, "dst="))
					} else if strings.HasPrefix(f, "bytes=") {
						if b, err := strconv.ParseUint(strings.TrimPrefix(f, "bytes="), 10, 64); err == nil {
							bytesList = append(bytesList, b)
						}
					}
				}

				if len(srcList) >= 2 && len(bytesList) >= 2 {
					origSrc := srcList[0]
					origBytes := bytesList[0]
					replyBytes := bytesList[1]

					if dev, ok := ipToDev[origSrc]; ok {
						currentTx[dev.MAC] += origBytes
						currentRx[dev.MAC] += replyBytes
					} else if len(dstList) >= 2 {
						replyDst := dstList[1]
						if dev, ok := ipToDev[replyDst]; ok {
							currentTx[dev.MAC] += origBytes
							currentRx[dev.MAC] += replyBytes
						}
					}
				}
			}
			file.Close()
		}
	}

	for mac, d := range dt.devices {
		rx := currentRx[mac]
		tx := currentTx[mac]

		prevRx := dt.prevRxBytes[mac]
		prevTx := dt.prevTxBytes[mac]

		if rx >= prevRx && prevRx > 0 {
			rxDiff := rx - prevRx
			d.RxRate = uint64(float64(rxDiff) / elapsed)
			d.TotalBytes += rxDiff
		} else {
			d.RxRate = 0
		}

		if tx >= prevTx && prevTx > 0 {
			txDiff := tx - prevTx
			d.TxRate = uint64(float64(txDiff) / elapsed)
			d.TotalBytes += txDiff
		} else {
			d.TxRate = 0
		}

		dt.prevRxBytes[mac] = rx
		dt.prevTxBytes[mac] = tx
	}

	dt.lastSample = now
}

// GetDevice returns a single device by MAC address
func (dt *DeviceTracker) GetDevice(mac string) *models.Device {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.devices[strings.ToUpper(mac)]
}

// getVendorByMAC matches common MAC OUI vendor prefixes
func getVendorByMAC(mac string) string {
	clean := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(mac, ":", ""), "-", ""))
	if len(clean) < 6 {
		return "Unknown"
	}
	prefix := clean[:6]

	vendors := map[string]string{
		"F01898": "Apple",
		"ACBC32": "Apple",
		"DC2B61": "Apple",
		"BC9FE4": "Apple",
		"38F9D3": "Apple",
		"9801A7": "Apple",
		"68DBCA": "Huawei",
		"707BB8": "Huawei",
		"00E0FC": "Huawei",
		"E40ECD": "Huawei",
		"50D2F5": "Xiaomi",
		"7C49EB": "Xiaomi",
		"286C07": "Xiaomi",
		"ACF7F3": "Xiaomi",
		"64B5C6": "Samsung",
		"B0C554": "Samsung",
		"D0176A": "Samsung",
		"00095B": "Netgear",
		"001A70": "Cisco",
		"00248C": "ASUS",
		"000E08": "Sony",
		"709E29": "Sony (PlayStation)",
		"9CCB78": "Nintendo (Switch)",
		"70480F": "Nintendo",
		"28FFA7": "Microsoft (Xbox)",
	}

	if v, ok := vendors[prefix]; ok {
		return v
	}
	return "Standard Device"
}

func init() {
	log.Println("[Device] Device tracker module initialized.")
}
