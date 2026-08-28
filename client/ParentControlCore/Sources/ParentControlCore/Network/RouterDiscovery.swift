import Foundation
import Network
#if canImport(Darwin)
import Darwin
#endif

public actor RouterDiscovery {
    public static let shared = RouterDiscovery()

    private let defaultFallbackCandidates = [
        "http://192.168.0.110:8088",
        "http://192.168.0.1:8088",
        "http://192.168.1.1:8088",
        "http://192.168.31.1:8088",
        "http://192.168.50.1:8088",
        "http://192.168.2.1:8088",
        "http://10.0.0.1:8088"
    ]

    private let session: URLSession

    public init() {
        let config = URLSessionConfiguration.ephemeral
        config.waitsForConnectivity = false
        config.timeoutIntervalForRequest = 2.0
        config.timeoutIntervalForResource = 3.0
        self.session = URLSession(configuration: config)
    }

    /// Gets dynamically deduced candidate URL list from local Wi-Fi subnet
    public func getCandidateURLs() -> [String] {
        var candidates: [String] = []

        if let wifiIP = getLocalWiFiIP() {
            let parts = wifiIP.split(separator: ".")
            if parts.count == 4 {
                let subnet = "\(parts[0]).\(parts[1]).\(parts[2])"
                // Add deduced common gateway and default server addresses
                candidates.append("http://\(subnet).1:8088")
                candidates.append("http://\(subnet).110:8088")
                candidates.append("http://\(subnet).254:8088")
                candidates.append("http://\(subnet).2:8088")
            }
        }

        for fallback in defaultFallbackCandidates {
            if !candidates.contains(fallback) {
                candidates.append(fallback)
            }
        }

        return candidates
    }

    /// Concurrently scans common gateway addresses and returns the first responsive router address
    public func discoverRouter() async -> String? {
        let candidateList = getCandidateURLs()

        return await withTaskGroup(of: String?.self) { group in
            for candidate in candidateList {
                group.addTask {
                    if await self.probe(urlStr: candidate) {
                        return candidate
                    }
                    return nil
                }
            }

            for await result in group {
                if let found = result {
                    group.cancelAll()
                    return found
                }
            }
            return nil
        }
    }

    private func probe(urlStr: String) async -> Bool {
        guard let url = URL(string: "\(urlStr)/api/status") else { return false }
        var req = URLRequest(url: url)
        req.timeoutInterval = 2.0

        do {
            let (data, response) = try await session.data(for: req)
            guard let http = response as? HTTPURLResponse, http.statusCode == 200 else { return false }
            let status = try JSONDecoder().decode(SystemStatus.self, from: data)
            return status.running
        } catch {
            return false
        }
    }

    /// Get local Wi-Fi IPv4 address
    private func getLocalWiFiIP() -> String? {
        #if canImport(Darwin)
        var address: String?
        var ifaddr: UnsafeMutablePointer<ifaddrs>?
        guard getifaddrs(&ifaddr) == 0, let firstAddr = ifaddr else { return nil }
        defer { freeifaddrs(ifaddr) }

        for ptr in sequence(first: firstAddr, next: { $0.pointee.ifa_next }) {
            let interface = ptr.pointee
            let addrFamily = interface.ifa_addr.pointee.sa_family
            if addrFamily == UInt8(AF_INET) {
                let name = String(cString: interface.ifa_name)
                if name == "en0" || name == "en1" {
                    var hostname = [CChar](repeating: 0, count: Int(NI_MAXHOST))
                    getnameinfo(interface.ifa_addr, socklen_t(interface.ifa_addr.pointee.sa_len),
                                &hostname, socklen_t(hostname.count),
                                nil, socklen_t(0), NI_NUMERICHOST)
                    address = String(cString: hostname)
                    break
                }
            }
        }
        return address
        #else
        return nil
        #endif
    }
}
