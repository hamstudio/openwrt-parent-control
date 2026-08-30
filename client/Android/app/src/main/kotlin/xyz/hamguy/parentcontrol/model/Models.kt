package xyz.hamguy.parentcontrol.model

import com.google.gson.annotations.SerializedName

data class TimeRange(
    @SerializedName("start_time") var startTime: String = "21:30",
    @SerializedName("end_time") var endTime: String = "07:00"
)

data class ScheduleRule(
    @SerializedName("enabled") var enabled: Boolean = true,
    @SerializedName("days") var days: List<Int> = listOf(0, 1, 2, 3, 4, 5, 6),
    @SerializedName("time_ranges") var timeRanges: List<TimeRange> = listOf(TimeRange("21:30", "07:00")),
    @SerializedName("action") var action: String = "block" // "block" or "allow"
)

data class Member(
    @SerializedName("id") var id: String = "m_${System.currentTimeMillis()}",
    @SerializedName("name") var name: String = "",
    @SerializedName("avatar") var avatar: String = "boy",
    @SerializedName("device_macs") var deviceMacs: List<String> = emptyList(),
    @SerializedName("enabled") var enabled: Boolean = true,
    @SerializedName("is_locked") var isLocked: Boolean = false,
    @SerializedName("bonus_until") var bonusUntil: String? = null,
    @SerializedName("quota_minutes") var quotaMinutes: Int = 120,
    @SerializedName("used_minutes") var usedMinutes: Int = 0,
    @SerializedName("last_active_time") var lastActiveTime: String? = null,
    @SerializedName("schedule") var schedule: ScheduleRule = ScheduleRule(),
    @SerializedName("blocked_app_ids") var blockedAppIds: List<Int> = emptyList(),
    @SerializedName("safe_search") var safeSearch: Boolean = true,
    @SerializedName("block_adult") var blockAdult: Boolean = true
) {
    val isBonusActive: Boolean
        get() = !bonusUntil.isNullOrEmpty()

    val isQuotaExceeded: Boolean
        get() = quotaMinutes in 1..usedMinutes

    val quotaProgress: Float
        get() = if (quotaMinutes <= 0) 0f else (usedMinutes.toFloat() / quotaMinutes.toFloat()).coerceIn(0f, 1f)
}

data class Device(
    @SerializedName("ip") val ip: String = "",
    @SerializedName("mac") val mac: String = "",
    @SerializedName("hostname") val hostname: String = "",
    @SerializedName("custom_name") val customName: String? = null,
    @SerializedName("vendor") val vendor: String = "Generic",
    @SerializedName("online") val online: Boolean = true,
    @SerializedName("member_id") val memberId: String? = null,
    @SerializedName("is_locked") val isLocked: Boolean = false,
    @SerializedName("tx_rate") val txRate: Long = 0,
    @SerializedName("rx_rate") val rxRate: Long = 0,
    @SerializedName("total_bytes") val totalBytes: Long = 0,
    @SerializedName("last_seen") val lastSeen: String? = null
) {
    val displayName: String
        get() = if (!customName.isNullOrEmpty()) customName else hostname.ifEmpty { ip.ifEmpty { mac } }
}

data class AppInfo(
    @SerializedName("id") val id: Int = 0,
    @SerializedName("name") val name: String = "",
    @SerializedName("class_id") val classId: Int = 0,
    @SerializedName("class_name") val className: String = "",
    @SerializedName("class_zh") val classZh: String = "",
    @SerializedName("description") val description: String? = null,
    @SerializedName("is_custom") val isCustom: Boolean? = true
)

data class AppCategory(
    @SerializedName("class_id") val classId: Int = 0,
    @SerializedName("class_name") val className: String = "",
    @SerializedName("class_zh") val classZh: String = "",
    @SerializedName("icon") val icon: String = "grid",
    @SerializedName("is_custom") val isCustom: Boolean? = true,
    @SerializedName("apps") val apps: List<AppInfo> = emptyList()
)

data class GlobalSettings(
    @SerializedName("enabled") var enabled: Boolean = true,
    @SerializedName("pin_code") var pinCode: String? = null,
    @SerializedName("cloud_sync_enabled") var cloudSyncEnabled: Boolean? = false,
    @SerializedName("cloud_worker_url") var cloudWorkerUrl: String? = null,
    @SerializedName("cloud_device_secret") var cloudDeviceSecret: String? = null,
    @SerializedName("enforce_safe_search") var enforceSafeSearch: Boolean = true,
    @SerializedName("block_doh_dot") var blockDoHDoT: Boolean = true,
    @SerializedName("isolate_new_devices") var isolateNewDevices: Boolean = false
)

data class SystemStatus(
    @SerializedName("running") val running: Boolean = true,
    @SerializedName("uptime_seconds") val uptimeSeconds: Long = 0,
    @SerializedName("total_devices") val totalDevices: Int = 0,
    @SerializedName("active_devices") val activeDevices: Int = 0,
    @SerializedName("managed_members") val managedMembers: Int = 0,
    @SerializedName("blocked_count_today") val blockedCountToday: Long = 0,
    @SerializedName("kernel_dpi_ready") val kernelDpiReady: Boolean = true,
    @SerializedName("app_count") val appCount: Int = 0,
    @SerializedName("pin_required") val pinRequired: Boolean = false,
    @SerializedName("server_time") val serverTime: String? = null
)
