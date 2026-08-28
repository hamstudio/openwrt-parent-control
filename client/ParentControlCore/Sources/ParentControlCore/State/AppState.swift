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

    @Published public var pinCode: String {
        didSet {
            UserDefaults.standard.set(pinCode, forKey: "parentcontrol_pin_code")
            Task {
                await client.setPinCode(pinCode.isEmpty ? nil : pinCode)
                await refreshAll()
            }
        }
    }

    @Published public var appLanguage: String {
        didSet {
            I18n.shared.setLocale(appLanguage)
        }
    }

    @Published public var isConnected: Bool = false
    @Published public var isRefreshing: Bool = false
    @Published public var needsPinAuth: Bool = false
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
        let savedPin = UserDefaults.standard.string(forKey: "parentcontrol_pin_code") ?? ""
        let savedLang = UserDefaults.standard.string(forKey: "parentcontrol_app_lang") ?? "auto"
        self.serverURL = savedURL
        self.pinCode = savedPin
        self.appLanguage = savedLang
        self.client = ParentControlClient(baseURLString: savedURL, pinCode: savedPin.isEmpty ? nil : savedPin)
        I18n.shared.setLocale(savedLang)
    }

    // MARK: - Lifecycle
    public func startAutoRefresh() {
        Task {
            await refreshAll()
            // Auto discover router if default address fails
            if !self.isConnected {
                await autoDiscover()
            }
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

    public func setPin(_ newPin: String) {
        self.pinCode = newPin
    }

    // MARK: - Actions
    public func refreshAll() async {
        isRefreshing = true
        errorMessage = nil

        do {
            // 1. Fetch public status endpoint
            let currentStatus = try await client.fetchStatus()
            self.status = currentStatus
            self.isConnected = true

            // 2. Check if PIN authentication is required
            if currentStatus.pinRequired == true && self.pinCode.isEmpty {
                self.needsPinAuth = true
                self.errorMessage = "Router PIN protection is enabled. Please enter PIN to load data"
                self.isRefreshing = false
                return
            }

            // 3. Concurrently fetch protected endpoints
            async let m = client.fetchMembers()
            async let d = client.fetchDevices()
            async let c = client.fetchAppCategories()
            async let set = client.fetchSettings()

            let (resMembers, resDevices, resCategories, resSettings) = try await (m, d, c, set)

            self.members = resMembers
            self.devices = resDevices
            self.categories = resCategories
            self.settings = resSettings
            self.needsPinAuth = false
        } catch let err as ParentControlError {
            if case .serverError(let code) = err, code == 401 {
                self.needsPinAuth = true
                self.errorMessage = "Incorrect PIN or unauthorized (401), please try again"
            } else {
                self.isConnected = false
                self.errorMessage = err.localizedDescription
            }
        } catch {
            self.isConnected = false
            self.errorMessage = error.localizedDescription
        }

        isRefreshing = false
    }

    public func refreshLightweight() async {
        guard isConnected && !needsPinAuth else { return }

        do {
            async let s = client.fetchStatus()
            async let m = client.fetchMembers()
            async let d = client.fetchDevices()

            let (resStatus, resMembers, resDevices) = try await (s, m, d)

            self.status = resStatus
            self.members = resMembers
            self.devices = resDevices
            self.isConnected = true
        } catch let err as ParentControlError {
            if case .serverError(let code) = err, code == 401 {
                self.needsPinAuth = true
            }
        } catch {
            // Keep previous state to avoid UI flashing
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
            self.errorMessage = "Lock member failed: \(error.localizedDescription)"
        }
    }

    public func unlockMember(id: String) async {
        do {
            try await client.unlockMember(id: id)
            await refreshLightweight()
        } catch {
            self.errorMessage = "Unlock member failed: \(error.localizedDescription)"
        }
    }

    public func bonusMember(id: String, minutes: Int = 30) async {
        do {
            try await client.bonusMember(id: id, minutes: minutes)
            await refreshLightweight()
        } catch {
            self.errorMessage = "Grant bonus failed: \(error.localizedDescription)"
        }
    }

    public func lockDevice(mac: String) async {
        do {
            try await client.lockDevice(mac: mac)
            await refreshAll()
        } catch {
            self.errorMessage = "Block device failed: \(error.localizedDescription)"
        }
    }

    public func unlockDevice(mac: String) async {
        do {
            try await client.unlockDevice(mac: mac)
            await refreshAll()
        } catch {
            self.errorMessage = "Unblock device failed: \(error.localizedDescription)"
        }
    }

    public func assignDevice(mac: String, memberId: String?) async -> Bool {
        do {
            try await client.assignDevice(mac: mac, memberId: memberId)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "Assign device failed: \(error.localizedDescription)"
            return false
        }
    }

    public func saveMember(_ member: Member) async -> Bool {
        do {
            _ = try await client.saveMember(member)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "Save member failed: \(error.localizedDescription)"
            return false
        }
    }

    public func deleteMember(id: String) async {
        do {
            try await client.deleteMember(id: id)
            await refreshAll()
        } catch {
            self.errorMessage = "Delete member failed: \(error.localizedDescription)"
        }
    }

    public func saveSettings(_ newSettings: GlobalSettings) async -> Bool {
        do {
            self.settings = try await client.saveSettings(newSettings)
            return true
        } catch {
            self.errorMessage = "Save settings failed: \(error.localizedDescription)"
            return false
        }
    }

    public func createApp(_ app: AppInfo) async -> Bool {
        do {
            _ = try await client.createApp(app)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "Create app failed: \(error.localizedDescription)"
            return false
        }
    }

    public func updateApp(id: Int, app: AppInfo) async -> Bool {
        do {
            _ = try await client.updateApp(id: id, app: app)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "Update app failed: \(error.localizedDescription)"
            return false
        }
    }

    public func deleteApp(id: Int) async {
        do {
            try await client.deleteApp(id: id)
            await refreshAll()
        } catch {
            self.errorMessage = "Delete app failed: \(error.localizedDescription)"
        }
    }

    public func createCategory(_ cat: AppCategory) async -> Bool {
        do {
            _ = try await client.createCategory(cat)
            await refreshAll()
            return true
        } catch {
            self.errorMessage = "Create category failed: \(error.localizedDescription)"
            return false
        }
    }

    public func deleteCategory(id: Int) async {
        do {
            try await client.deleteCategory(id: id)
            await refreshAll()
        } catch {
            self.errorMessage = "Delete category failed: \(error.localizedDescription)"
        }
    }
}
