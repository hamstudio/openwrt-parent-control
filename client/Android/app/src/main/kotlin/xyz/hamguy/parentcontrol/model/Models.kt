package xyz.hamguy.parentcontrol.model

import com.google.gson.annotations.SerializedName

data class SystemStatus(
    @SerializedName("running") val running: Boolean = true,
    @SerializedName("uptime_seconds") val uptimeSeconds: Long = 0,
    @SerializedName("total_devices") val totalDevices: Int = 0,
    @SerializedName("active_devices") val activeDevices: Int = 0,
    @SerializedName("managed_members") val managedMembers: Int = 0,
    @SerializedName("kernel_dpi_ready") val kernelDpiReady: Boolean = true,
    @SerializedName("app_count") val appCount: Int = 0,
    @SerializedName("pin_required") val pinRequired: Boolean = false
)

data class Member(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("avatar") val avatar: String = "boy",
    @SerializedName("device_macs") val deviceMacs: List<String> = emptyList(),
    @SerializedName("enabled") val enabled: Boolean = true,
    @SerializedName("is_locked") val isLocked: Boolean = false,
    @SerializedName("bonus_until") val bonusUntil: String? = null,
    @SerializedName("quota_minutes") val quotaMinutes: Int = 120,
    @SerializedName("used_minutes") val usedMinutes: Int = 0,
    @SerializedName("blocked_app_ids") val blockedAppIds: List<Int> = emptyList()
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
    @SerializedName("vendor") val vendor: String = "Generic",
    @SerializedName("online") val online: Boolean = true,
    @SerializedName("member_id") val memberId: String? = null,
    @SerializedName("is_locked") val isLocked: Boolean = false,
    @SerializedName("tx_rate") val txRate: Long = 0,
    @SerializedName("rx_rate") val rxRate: Long = 0,
    @SerializedName("total_bytes") val totalBytes: Long = 0,
    @SerializedName("last_seen") val lastSeen: Long = 0
) {
    val displayName: String
        get() = hostname.ifEmpty { ip.ifEmpty { mac } }
}

data class AppCategory(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("icon") val icon: String = "📱",
    @SerializedName("app_count") val appCount: Int = 0
)

data class GlobalSettings(
    @SerializedName("enabled") val enabled: Boolean = true,
    @SerializedName("enforce_safesearch") val enforceSafeSearch: Boolean = true,
    @SerializedName("block_doh_dot") val blockDoHDoT: Boolean = true,
    @SerializedName("isolate_new_devices") val isolateNewDevices: Boolean = false,
    @SerializedName("pin_code") val pinCode: String = ""
)
