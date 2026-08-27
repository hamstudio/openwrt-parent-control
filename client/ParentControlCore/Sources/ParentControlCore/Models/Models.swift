import Foundation

// MARK: - Device Model
public struct Device: Codable, Identifiable, Hashable, Sendable {
    public var id: String { mac }
    public let mac: String
    public let ip: String
    public let hostname: String
    public let customName: String?
    public let vendor: String
    public let online: boolValue
    public let memberId: String?
    public let txRate: UInt64
    public let rxRate: UInt64
    public let totalBytes: UInt64
    public let lastSeen: String

    public typealias boolValue = Bool

    public enum CodingKeys: String, CodingKey {
        case mac, ip, hostname, vendor, online
        case customName = "custom_name"
        case memberId = "member_id"
        case txRate = "tx_rate"
        case rxRate = "rx_rate"
        case totalBytes = "total_bytes"
        case lastSeen = "last_seen"
    }

    public init(
        mac: String,
        ip: String,
        hostname: String,
        customName: String? = nil,
        vendor: String,
        online: Bool,
        memberId: String? = nil,
        txRate: UInt64 = 0,
        rxRate: UInt64 = 0,
        totalBytes: UInt64 = 0,
        lastSeen: String = ""
    ) {
        self.mac = mac
        self.ip = ip
        self.hostname = hostname
        self.customName = customName
        self.vendor = vendor
        self.online = online
        self.memberId = memberId
        self.txRate = txRate
        self.rxRate = rxRate
        self.totalBytes = totalBytes
        self.lastSeen = lastSeen
    }

    public var displayName: String {
        if let custom = customName, !custom.isEmpty {
            return custom
        }
        return hostname.isEmpty ? "Unknown-Device" : hostname
    }

    public var formattedSpeed: String {
        if rxRate >= 1024 * 1024 {
            return String(format: "%.1f MB/s", Double(rxRate) / (1024.0 * 1024.0))
        } else {
            return "\(rxRate / 1024) KB/s"
        }
    }
}

// MARK: - TimeRange Model
public struct TimeRange: Codable, Hashable, Sendable {
    public var startTime: String // "HH:MM"
    public var endTime: String   // "HH:MM"

    public enum CodingKeys: String, CodingKey {
        case startTime = "start_time"
        case endTime = "end_time"
    }

    public init(startTime: String, endTime: String) {
        self.startTime = startTime
        self.endTime = endTime
    }
}

// MARK: - ScheduleRule Model
public struct ScheduleRule: Codable, Hashable, Sendable {
    public var enabled: Bool
    public var days: [Int] // 0=Sun, 1=Mon, ..., 6=Sat
    public var timeRanges: [TimeRange]
    public var action: String // "block" or "allow"

    public enum CodingKeys: String, CodingKey {
        case enabled, days, action
        case timeRanges = "time_ranges"
    }

    public init(
        enabled: Bool = true,
        days: [Int] = [0, 1, 2, 3, 4, 5, 6],
        timeRanges: [TimeRange] = [TimeRange(startTime: "21:30", endTime: "07:00")],
        action: String = "block"
    ) {
        self.enabled = enabled
        self.days = days
        self.timeRanges = timeRanges
        self.action = action
    }
}

// MARK: - Member Model
public struct Member: Codable, Identifiable, Hashable, Sendable {
    public var id: String
    public var name: String
    public var avatar: String // "boy", "girl", "student", "child"
    public var deviceMACs: [String]
    public var enabled: Bool
    public var isLocked: Bool
    public var bonusUntil: String?
    public var quotaMinutes: Int
    public var usedMinutes: Int
    public var lastActiveTime: String?
    public var schedule: ScheduleRule
    public var blockedAppIDs: [Int]
    public var safeSearch: Bool
    public var blockAdult: Bool

    public enum CodingKeys: String, CodingKey {
        case id, name, avatar, enabled, schedule
        case deviceMACs = "device_macs"
        case isLocked = "is_locked"
        case bonusUntil = "bonus_until"
        case quotaMinutes = "quota_minutes"
        case usedMinutes = "used_minutes"
        case lastActiveTime = "last_active_time"
        case blockedAppIDs = "blocked_app_ids"
        case safeSearch = "safe_search"
        case blockAdult = "block_adult"
    }

    public init(
        id: String = "m_\(Int(Date().timeIntervalSince1970 * 1000))",
        name: String,
        avatar: String = "boy",
        deviceMACs: [String] = [],
        enabled: Bool = true,
        isLocked: Bool = false,
        bonusUntil: String? = nil,
        quotaMinutes: Int = 120,
        usedMinutes: Int = 0,
        lastActiveTime: String? = nil,
        schedule: ScheduleRule = ScheduleRule(),
        blockedAppIDs: [Int] = [],
        safeSearch: Bool = true,
        blockAdult: Bool = true
    ) {
        self.id = id
        self.name = name
        self.avatar = avatar
        self.deviceMACs = deviceMACs
        self.enabled = enabled
        self.isLocked = isLocked
        self.bonusUntil = bonusUntil
        self.quotaMinutes = quotaMinutes
        self.usedMinutes = usedMinutes
        self.lastActiveTime = lastActiveTime
        self.schedule = schedule
        self.blockedAppIDs = blockedAppIDs
        self.safeSearch = safeSearch
        self.blockAdult = blockAdult
    }

    public var quotaProgress: Double {
        guard quotaMinutes > 0 else { return 0 }
        return min(1.0, Double(usedMinutes) / Double(quotaMinutes))
    }

    public var isQuotaExceeded: Bool {
        return quotaMinutes > 0 && usedMinutes >= quotaMinutes
    }

    public var isBonusActive: Bool {
        guard let bonusStr = bonusUntil, !bonusStr.isEmpty else { return false }
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: bonusStr) ?? ISO8601DateFormatter().date(from: bonusStr) {
            return date > Date()
        }
        return false
    }
}

// MARK: - AppInfo & Category
public struct AppInfo: Codable, Identifiable, Hashable, Sendable {
    public let id: Int
    public let name: String
    public let classId: Int
    public let className: String
    public let classZh: String
    public let description: String?
    public let isCustom: Bool?

    public enum CodingKeys: String, CodingKey {
        case id, name, description
        case classId = "class_id"
        case className = "class_name"
        case classZh = "class_zh"
        case isCustom = "is_custom"
    }

    public init(id: Int = 0, name: String, classId: Int, className: String = "", classZh: String = "", description: String? = nil, isCustom: Bool? = true) {
        self.id = id
        self.name = name
        self.classId = classId
        self.className = className
        self.classZh = classZh
        self.description = description
        self.isCustom = isCustom
    }
}

public struct AppCategory: Codable, Identifiable, Hashable, Sendable {
    public var id: Int { classId }
    public let classId: Int
    public let className: String
    public let classZh: String
    public let icon: String
    public let isCustom: Bool?
    public let apps: [AppInfo]

    public enum CodingKeys: String, CodingKey {
        case icon, apps
        case classId = "class_id"
        case className = "class_name"
        case classZh = "class_zh"
        case isCustom = "is_custom"
    }

    public init(classId: Int = 0, className: String = "", classZh: String, icon: String = "grid", isCustom: Bool? = true, apps: [AppInfo] = []) {
        self.classId = classId
        self.className = className
        self.classZh = classZh
        self.icon = icon
        self.isCustom = isCustom
        self.apps = apps
    }
}

// MARK: - GlobalSettings
public struct GlobalSettings: Codable, Hashable, Sendable {
    public var enabled: Bool
    public var enforceSafeSearch: Bool
    public var blockDoHDoT: Bool
    public var isolateNewDevices: Bool

    public enum CodingKeys: String, CodingKey {
        case enabled
        case enforceSafeSearch = "enforce_safe_search"
        case blockDoHDoT = "block_doh_dot"
        case isolateNewDevices = "isolate_new_devices"
    }

    public init(
        enabled: Bool = true,
        enforceSafeSearch: Bool = true,
        blockDoHDoT: Bool = true,
        isolateNewDevices: Bool = false
    ) {
        self.enabled = enabled
        self.enforceSafeSearch = enforceSafeSearch
        self.blockDoHDoT = blockDoHDoT
        self.isolateNewDevices = isolateNewDevices
    }
}

// MARK: - SystemStatus
public struct SystemStatus: Codable, Hashable, Sendable {
    public let running: Bool
    public let uptimeSeconds: Int64
    public let totalDevices: Int
    public let activeDevices: Int
    public let managedMembers: Int
    public let blockedCountToday: Int64
    public let kernelDpiReady: Bool
    public let appCount: Int
    public let serverTime: String

    public enum CodingKeys: String, CodingKey {
        case running
        case uptimeSeconds = "uptime_seconds"
        case totalDevices = "total_devices"
        case activeDevices = "active_devices"
        case managedMembers = "managed_members"
        case blockedCountToday = "blocked_count_today"
        case kernelDpiReady = "kernel_dpi_ready"
        case appCount = "app_count"
        case serverTime = "server_time"
    }
}
