import Foundation

public actor RouterDiscovery {
    public static let shared = RouterDiscovery()

    private let defaultCandidates = [
        "http://192.168.0.110:8088",
        "http://192.168.0.1:8088",
        "http://192.168.1.1:8088",
        "http://192.168.31.1:8088",
        "http://192.168.50.1:8088",
        "http://10.0.0.1:8088"
    ]

    public init() {}

    /// 并发扫描常用网关地址，返回首个正常响应的路由器地址
    public func discoverRouter() async -> String? {
        await withTaskGroup(of: String?.self) { group in
            for candidate in defaultCandidates {
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
        req.timeoutInterval = 1.5

        do {
            let (data, response) = try await URLSession.shared.data(for: req)
            guard let http = response as? HTTPURLResponse, http.statusCode == 200 else { return false }
            let status = try JSONDecoder().decode(SystemStatus.self, from: data)
            return status.running
        } catch {
            return false
        }
    }
}
