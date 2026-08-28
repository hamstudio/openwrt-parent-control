package device

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"parentcontrol/internal/models"
)

// DeviceTracker manages local network device discovery and traffic monitoring
type DeviceTracker struct {
	mu         sync.RWMutex
	devices    map[string]*models.Device // key: MAC
	prevBytes  map[string]uint64
	lastSample time.Time
}

// NewDeviceTracker creates a new DeviceTracker instance
func NewDeviceTracker() *DeviceTracker {
	dt := &DeviceTracker{
		devices:    make(map[string]*models.Device),
		prevBytes:  make(map[string]uint64),
		lastSample: time.Now(),
	}
	return dt
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

	currentBytes := make(map[string]uint64)

	// Try reading /proc/net/nf_conntrack or /proc/net/ip_conntrack
	conntrackPath := "/proc/net/nf_conntrack"
	if _, err := os.Stat(conntrackPath); os.IsNotExist(err) {
		conntrackPath = "/proc/net/ip_conntrack"
	}

	if file, err := os.Open(conntrackPath); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			// Look for bytes= statistic: src=192.168.0.150 dst=... bytes=12345
			if strings.Contains(line, "bytes=") {
				fields := strings.Fields(line)
				var srcIP string
				var bytesVal uint64
				for _, f := range fields {
					if strings.HasPrefix(f, "src=192.168.") {
						srcIP = strings.TrimPrefix(f, "src=")
					} else if strings.HasPrefix(f, "bytes=") && srcIP != "" {
						b, _ := strconv.ParseUint(strings.TrimPrefix(f, "bytes="), 10, 64)
						bytesVal += b
					}
				}
				if srcIP != "" {
					// Map IP to MAC
					for _, d := range dt.devices {
						if d.IP == srcIP {
							currentBytes[d.MAC] += bytesVal
							break
						}
					}
				}
			}
		}
		file.Close()
	}

	for mac, d := range dt.devices {
		curr := currentBytes[mac]
		prev := dt.prevBytes[mac]
		if curr >= prev && prev > 0 {
			diff := curr - prev
			d.RxRate = uint64(float64(diff) / elapsed)
			d.TotalBytes += diff
		}
		dt.prevBytes[mac] = curr
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
