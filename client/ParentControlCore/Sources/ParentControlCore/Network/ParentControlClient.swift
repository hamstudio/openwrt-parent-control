import Foundation

public enum ParentControlError: LocalizedError, Sendable {
    case invalidURL
    case networkError(String)
    case serverError(Int)
    case decodingError(String)
    case actionFailed(String)

    public var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "无效的路由器服务器地址"
        case .networkError(let msg):
            return "网络连接失败: \(msg)"
        case .serverError(let code):
            return "路由器返回错误状态码: \(code)"
        case .decodingError(let msg):
            return "数据解析失败: \(msg)"
        case .actionFailed(let msg):
            return "操作失败: \(msg)"
        }
    }
}

public actor ParentControlClient {
    private var baseURL: URL
    private let session: URLSession

    public init(baseURLString: String = "http://192.168.0.110:8088", session: URLSession = .shared) {
        self.baseURL = URL(string: baseURLString) ?? URL(string: "http://192.168.0.110:8088")!
        self.session = session
    }

    public func updateBaseURL(_ newURLString: String) {
        if let url = URL(string: newURLString) {
            self.baseURL = url
        }
    }

    public func currentBaseURL() -> String {
        return baseURL.absoluteString
    }

    // MARK: - Status
    public func fetchStatus() async throws -> SystemStatus {
        return try await get(endpoint: "/api/status")
    }

    // MARK: - Devices
    public func fetchDevices() async throws -> [Device] {
        return try await get(endpoint: "/api/devices")
    }

    // MARK: - App Categories & Apps CRUD
    public func fetchAppCategories() async throws -> [AppCategory] {
        return try await get(endpoint: "/api/apps")
    }

    public func createApp(_ app: AppInfo) async throws -> AppInfo {
        return try await post(endpoint: "/api/apps", body: app)
    }

    public func updateApp(id: Int, app: AppInfo) async throws -> AppInfo {
        return try await post(endpoint: "/api/apps/\(id)", body: app)
    }

    public func deleteApp(id: Int) async throws {
        let endpoint = "/api/apps/\(id)"
        guard let url = URL(string: endpoint, relativeTo: baseURL) else {
            throw ParentControlError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        request.timeoutInterval = 5

        let (data, response) = try await session.data(for: request)
        guard let httpRes = response as? HTTPURLResponse, (200...299).contains(httpRes.statusCode) else {
            let code = (response as? HTTPURLResponse)?.statusCode ?? 500
            throw ParentControlError.serverError(code)
        }
        _ = data
    }

    public func createCategory(_ category: AppCategory) async throws -> AppCategory {
        return try await post(endpoint: "/api/categories", body: category)
    }

    public func deleteCategory(id: Int) async throws {
        let endpoint = "/api/categories/\(id)"
        guard let url = URL(string: endpoint, relativeTo: baseURL) else {
            throw ParentControlError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        request.timeoutInterval = 5

        let (data, response) = try await session.data(for: request)
        guard let httpRes = response as? HTTPURLResponse, (200...299).contains(httpRes.statusCode) else {
            let code = (response as? HTTPURLResponse)?.statusCode ?? 500
            throw ParentControlError.serverError(code)
        }
        _ = data
    }

    // MARK: - Members
    public func fetchMembers() async throws -> [Member] {
        return try await get(endpoint: "/api/members")
    }

    public func saveMember(_ member: Member) async throws -> Member {
        return try await post(endpoint: "/api/members", body: member)
    }

    public func deleteMember(id: String) async throws {
        let endpoint = "/api/members/\(id)"
        guard let url = URL(string: endpoint, relativeTo: baseURL) else {
            throw ParentControlError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = "DELETE"
        request.timeoutInterval = 5

        let (data, response) = try await session.data(for: request)
        guard let httpRes = response as? HTTPURLResponse, (200...299).contains(httpRes.statusCode) else {
            let code = (response as? HTTPURLResponse)?.statusCode ?? 500
            throw ParentControlError.serverError(code)
        }
        _ = data
    }

    public func lockMember(id: String) async throws {
        let endpoint = "/api/members/\(id)/lock"
        let _: [String: String] = try await postEmpty(endpoint: endpoint)
    }

    public func unlockMember(id: String) async throws {
        let endpoint = "/api/members/\(id)/unlock"
        let _: [String: String] = try await postEmpty(endpoint: endpoint)
    }

    public func bonusMember(id: String, minutes: Int = 30) async throws {
        let endpoint = "/api/members/\(id)/bonus?minutes=\(minutes)"
        let _: [String: String] = try await postEmpty(endpoint: endpoint)
    }

    // MARK: - Settings
    public func fetchSettings() async throws -> GlobalSettings {
        return try await get(endpoint: "/api/settings")
    }

    public func saveSettings(_ settings: GlobalSettings) async throws -> GlobalSettings {
        return try await post(endpoint: "/api/settings", body: settings)
    }

    // MARK: - Internal HTTP Helpers
    private func get<T: Decodable>(endpoint: String) async throws -> T {
        guard let url = URL(string: endpoint, relativeTo: baseURL) else {
            throw ParentControlError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.timeoutInterval = 5

        do {
            let (data, response) = try await session.data(for: request)
            guard let httpRes = response as? HTTPURLResponse, (200...299).contains(httpRes.statusCode) else {
                let code = (response as? HTTPURLResponse)?.statusCode ?? 500
                throw ParentControlError.serverError(code)
            }
            return try JSONDecoder().decode(T.self, from: data)
        } catch let decErr as DecodingError {
            throw ParentControlError.decodingError(decErr.localizedDescription)
        } catch {
            throw ParentControlError.networkError(error.localizedDescription)
        }
    }

    private func post<T: Decodable, B: Encodable>(endpoint: String, body: B) async throws -> T {
        guard let url = URL(string: endpoint, relativeTo: baseURL) else {
            throw ParentControlError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.timeoutInterval = 5

        do {
            request.httpBody = try JSONEncoder().encode(body)
            let (data, response) = try await session.data(for: request)
            guard let httpRes = response as? HTTPURLResponse, (200...299).contains(httpRes.statusCode) else {
                let code = (response as? HTTPURLResponse)?.statusCode ?? 500
                throw ParentControlError.serverError(code)
            }
            return try JSONDecoder().decode(T.self, from: data)
        } catch let decErr as DecodingError {
            throw ParentControlError.decodingError(decErr.localizedDescription)
        } catch {
            throw ParentControlError.networkError(error.localizedDescription)
        }
    }

    private func postEmpty<T: Decodable>(endpoint: String) async throws -> T {
        guard let url = URL(string: endpoint, relativeTo: baseURL) else {
            throw ParentControlError.invalidURL
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.timeoutInterval = 5

        do {
            let (data, response) = try await session.data(for: request)
            guard let httpRes = response as? HTTPURLResponse, (200...299).contains(httpRes.statusCode) else {
                let code = (response as? HTTPURLResponse)?.statusCode ?? 500
                throw ParentControlError.serverError(code)
            }
            return try JSONDecoder().decode(T.self, from: data)
        } catch let decErr as DecodingError {
            throw ParentControlError.decodingError(decErr.localizedDescription)
        } catch {
            throw ParentControlError.networkError(error.localizedDescription)
        }
    }
}
