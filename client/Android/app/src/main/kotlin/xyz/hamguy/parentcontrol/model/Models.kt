package xyz.hamguy.parentcontrol.model

import com.google.gson.annotations.SerializedName

data class SystemStatus(
    @SerializedName("running") val running: Boolean = true,
    @SerializedName("uptime_seconds") val uptimeSeconds: Long = 0,
    @SerializedName("total_devices") val totalDevices: Int = 0,
    @SerializedName("active_devices") val activeDevices: Int = 0,
    @SerializedName("managed_members") val managedMembers: Int = 0,
    @SerializedName("kernel_dpi_ready") val kernelDpiReady: Boolean = true,
    @SerializedName("app_count") val appCount: Int = 0
)

data class Member(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("avatar") val avatar: String = "👤",
    @SerializedName("mac_addresses") val macAddresses: List<String> = emptyList(),
    @SerializedName("ip_addresses") val ipAddresses: List<String> = emptyList(),
    @SerializedName("is_locked") val isLocked: Boolean = false,
    @SerializedName("today_usage_minutes") val todayUsageMinutes: Int = 0,
    @SerializedName("daily_time_limit_minutes") val dailyTimeLimitMinutes: Int = 120,
    @SerializedName("blocked_categories") val blockedCategories: List<String> = emptyList()
)

data class Device(
    @SerializedName("ip") val ip: String,
    @SerializedName("mac") val mac: String,
    @SerializedName("hostname") val hostname: String = "",
    @SerializedName("member_id") val memberId: String? = null,
    @SerializedName("is_online") val isOnline: Boolean = true,
    @SerializedName("tx_bytes") val txBytes: Long = 0,
    @SerializedName("rx_bytes") val rxBytes: Long = 0
)

data class AppCategory(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("icon") val icon: String = "📱",
    @SerializedName("app_count") val appCount: Int = 0
)
