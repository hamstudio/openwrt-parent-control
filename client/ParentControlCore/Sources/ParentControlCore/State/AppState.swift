import Foundation
import Combine

@MainActor
public final class AppState: ObservableObject {
    public static let shared = AppState()

    // MARK: - Published State
    @Published public var serverURL: String {
        didSet {
            UserDefaults.standard.set(serverURL, forKey: "parentcontrol_server_url")
            Task {
                await client.updateBaseURL(serverURL)
                await refreshAll()
            }
        }
    }

    @Published public var isConnected: Bool = false
    @Published public var isRefreshing: Bool = false
    @Published public var status: SystemStatus?
    @Published public var members: [Member] = []
    @Published public var devices: [Device] = []
    @Published public var categories: [AppCategory] = []
    @Published public var settings: GlobalSettings = GlobalSettings()
    @Published public var errorMessage: String?

    public let client: ParentControlClient
    private var timer: Timer?

    public init(initialURL: String = "http://192.168.0.110:8088") {
        let savedURL = UserDefaults.standard.string(forKey: "parentcontrol_server_url") ?? initialURL
        self.serverURL = savedURL
        self.client = ParentControlClient(baseURLString: savedURL)
    }

    // MARK: - Lifecycle
    public func startAutoRefresh() {
        Task {
            await refreshAll()
        }
        timer?.invalidate()
        timer = Timer.scheduledTimer(withTimeInterval: 5.0, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                await self?.refreshLightweight()
            }
        }
    }

    public func stopAutoRefresh() {
        timer?.invalidate()
        timer = nil
    }

    // MARK: - Actions
    public func refreshAll() async {
        isRefreshing = true
        errorMessage = nil

        do {
            async let s = client.fetchStatus()
            async let m = client.fetchMembers()
            async let d = client.fetchDevices()
            async let c = client.fetchAppCategories()
            async let set = client.fetchSettings()

            let (resStatus, resMembers, resDevices, resCategories, resSettings) = try await (s, m, d, c, set)

            self.status = resStatus
            self.members = resMembers
            self.devices = resDevices
            self.categories = resCategories
            self.settings = resSettings
            self.isConnected = true
        } catch {
            self.isConnected = false
            self.errorMessage = error.localizedDescription
        }

        isRefreshing = false
    }

    public func refreshLightweight() async {
        do {
            async let s = client.fetchStatus()
            async let m = client.fetchMembers()
            async let d = client.fetchDevices()

            let (resStatus, resMembers, resDevices) = try await (s, m, d)

            self.status = resStatus
            self.members = resMembers
            self.devices = resDevices
            self.isConnected = true
        } catch {
            self.isConnected = false
        }
    }

    public func autoDiscover() async {
        isRefreshing = true
        if let found = await RouterDiscovery.shared.discoverRouter() {
            self.serverURL = found
        }
        await refreshAll()
    }

    public func lockMember(id: String) async {
        do {
            try await client.lockMember(id: id)
            await refreshLightweight()
        } catch {
            self.errorMessage = "锁定失败: \(error.localizedDescription)"
        }
    }

    public func unlockMember(id: String) async {
        do {
            try await client.unlockMember(id: id)
            await refreshLightweight()
        } catch {
            self.errorMessage = "解锁失败: \(error.localizedDescription)"
        }
    }

    public func bonusMember(id: String, minutes: Int = 30) async {
        do {
            try await client.bonusMember(id: id, minutes: minutes)
            await refreshLightweight()
        } catch {
            self.errorMessage = "加时失败: \(error.localizedDescription)"
        }
    }

    public func saveMember(_ member: Member) async -> Bool {
        do {
            _ = try await client.saveMember(member)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "保存失败: \(error.localizedDescription)"
            return false
        }
    }

    public func deleteMember(id: String) async {
        do {
            try await client.deleteMember(id: id)
            await refreshAll()
        } catch {
            self.errorMessage = "删除失败: \(error.localizedDescription)"
        }
    }

    public func saveSettings(_ newSettings: GlobalSettings) async -> Bool {
        do {
            self.settings = try await client.saveSettings(newSettings)
            return true
        } catch {
            self.errorMessage = "保存设置失败: \(error.localizedDescription)"
            return false
        }
    }

    public func createApp(_ app: AppInfo) async -> Bool {
        do {
            _ = try await client.createApp(app)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "创建应用失败: \(error.localizedDescription)"
            return false
        }
    }

    public func updateApp(id: Int, app: AppInfo) async -> Bool {
        do {
            _ = try await client.updateApp(id: id, app: app)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "更新应用失败: \(error.localizedDescription)"
            return false
        }
    }

    public func deleteApp(id: Int) async {
        do {
            try await client.deleteApp(id: id)
            await refreshAll()
        } catch {
            self.errorMessage = "删除应用失败: \(error.localizedDescription)"
        }
    }

    public func createCategory(_ cat: AppCategory) async -> Bool {
        do {
            _ = try await client.createCategory(cat)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "创建分类失败: \(error.localizedDescription)"
            return false
        }
    }

    public func deleteCategory(id: Int) async {
        do {
            try await client.deleteCategory(id: id)
            await refreshAll()
        } catch {
            self.errorMessage = "删除分类失败: \(error.localizedDescription)"
        }
    }
}
