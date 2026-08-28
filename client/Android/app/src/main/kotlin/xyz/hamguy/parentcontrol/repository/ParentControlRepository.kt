package xyz.hamguy.parentcontrol.repository

import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import xyz.hamguy.parentcontrol.model.Device
import xyz.hamguy.parentcontrol.model.GlobalSettings
import xyz.hamguy.parentcontrol.model.Member
import xyz.hamguy.parentcontrol.model.SystemStatus
import java.util.concurrent.TimeUnit

class ParentControlRepository(
    var baseUrl: String = "http://192.168.0.110:8088",
    var pinCode: String = ""
) {

    private val gson = Gson()
    private val httpClient = OkHttpClient.Builder()
        .connectTimeout(5, TimeUnit.SECONDS)
        .readTimeout(10, TimeUnit.SECONDS)
        .build()

    private val _status = MutableStateFlow<SystemStatus?>(null)
    val status: StateFlow<SystemStatus?> = _status.asStateFlow()

    private val _members = MutableStateFlow<List<Member>>(emptyList())
    val members: StateFlow<List<Member>> = _members.asStateFlow()

    private val _devices = MutableStateFlow<List<Device>>(emptyList())
    val devices: StateFlow<List<Device>> = _devices.asStateFlow()

    private val _settings = MutableStateFlow<GlobalSettings>(GlobalSettings())
    val settings: StateFlow<GlobalSettings> = _settings.asStateFlow()

    private val _isConnected = MutableStateFlow(false)
    val isConnected: StateFlow<Boolean> = _isConnected.asStateFlow()

    private val _needsPinAuth = MutableStateFlow(false)
    val needsPinAuth: StateFlow<Boolean> = _needsPinAuth.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    private fun newRequestBuilder(endpoint: String): Request.Builder {
        val cleanBase = baseUrl.trimEnd('/')
        val cleanEndpoint = if (endpoint.startsWith("/")) endpoint else "/$endpoint"
        val builder = Request.Builder()
            .url("$cleanBase$cleanEndpoint")
            .header("Accept", "application/json")

        if (pinCode.isNotEmpty()) {
            builder.header("X-Pin-Code", pinCode)
            builder.header("Authorization", "Bearer $pinCode")
        }
        return builder
    }

    suspend fun refreshAll() = withContext(Dispatchers.IO) {
        _isLoading.value = true
        try {
            fetchStatus()
            fetchMembers()
            fetchDevices()
        } finally {
            _isLoading.value = false
        }
    }

    suspend fun fetchStatus(): Result<SystemStatus> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/status").get().build()
            val response = httpClient.newCall(request).execute()
            val body = response.body?.string() ?: "{}"
            val s = gson.fromJson(body, SystemStatus::class.java)
            _status.value = s
            _isConnected.value = true
            if (s.pinRequired && pinCode.isEmpty()) {
                _needsPinAuth.value = true
            }
            Result.success(s)
        } catch (e: Exception) {
            _isConnected.value = false
            Result.failure(e)
        }
    }

    suspend fun fetchMembers(): Result<List<Member>> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/members").get().build()
            val response = httpClient.newCall(request).execute()
            if (response.code == 401) {
                _needsPinAuth.value = true
                return@withContext Result.failure(Exception("401 PIN code required"))
            }
            val body = response.body?.string() ?: "[]"
            val type = object : TypeToken<List<Member>>() {}.type
            val memberList: List<Member> = gson.fromJson(body, type) ?: emptyList()
            _members.value = memberList
            _needsPinAuth.value = false
            Result.success(memberList)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchDevices(): Result<List<Device>> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/devices").get().build()
            val response = httpClient.newCall(request).execute()
            if (response.code == 401) {
                _needsPinAuth.value = true
                return@withContext Result.failure(Exception("401 PIN code required"))
            }
            val body = response.body?.string() ?: "[]"
            val type = object : TypeToken<List<Device>>() {}.type
            val deviceList: List<Device> = gson.fromJson(body, type) ?: emptyList()
            _devices.value = deviceList
            Result.success(deviceList)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun lockMember(memberId: String, lock: Boolean = true): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val action = if (lock) "lock" else "unlock"
            val request = newRequestBuilder("/api/members/$memberId/$action")
                .post("{}".toRequestBody("application/json".toMediaType()))
                .build()
            val response = httpClient.newCall(request).execute()
            val ok = response.isSuccessful
            if (ok) fetchMembers()
            Result.success(ok)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun bonusMember(memberId: String, minutes: Int = 30): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/members/$memberId/bonus?minutes=$minutes")
                .post("{}".toRequestBody("application/json".toMediaType()))
                .build()
            val response = httpClient.newCall(request).execute()
            val ok = response.isSuccessful
            if (ok) fetchMembers()
            Result.success(ok)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun lockDevice(mac: String, lock: Boolean = true): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val action = if (lock) "lock" else "unlock"
            val request = newRequestBuilder("/api/devices/$mac/$action")
                .post("{}".toRequestBody("application/json".toMediaType()))
                .build()
            val response = httpClient.newCall(request).execute()
            val ok = response.isSuccessful
            if (ok) fetchDevices()
            Result.success(ok)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun assignDevice(mac: String, memberId: String?): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val payload = mapOf("member_id" to (memberId ?: ""))
            val jsonBody = gson.toJson(payload)
            val request = newRequestBuilder("/api/devices/$mac/assign")
                .post(jsonBody.toRequestBody("application/json".toMediaType()))
                .build()
            val response = httpClient.newCall(request).execute()
            val ok = response.isSuccessful
            if (ok) {
                fetchMembers()
                fetchDevices()
            }
            Result.success(ok)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
