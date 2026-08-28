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
import xyz.hamguy.parentcontrol.core.NativeBridge
import xyz.hamguy.parentcontrol.model.Device
import xyz.hamguy.parentcontrol.model.Member
import xyz.hamguy.parentcontrol.model.SystemStatus
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume
import kotlin.coroutines.suspendCoroutine

class ParentControlRepository(var baseUrl: String = "http://192.168.0.110:8088") {

    private val gson = Gson()
    private val httpClient = OkHttpClient.Builder()
        .connectTimeout(5, TimeUnit.SECONDS)
        .readTimeout(10, TimeUnit.SECONDS)
        .build()

    private var nativeClientPtr: Long = 0

    private val _status = MutableStateFlow<SystemStatus?>(null)
    val status: StateFlow<SystemStatus?> = _status.asStateFlow()

    private val _members = MutableStateFlow<List<Member>>(emptyList())
    val members: StateFlow<List<Member>> = _members.asStateFlow()

    private val _devices = MutableStateFlow<List<Device>>(emptyList())
    val devices: StateFlow<List<Device>> = _devices.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    init {
        try {
            nativeClientPtr = NativeBridge.parentcontrol_create_client(baseUrl)
        } catch (e: Throwable) {
            nativeClientPtr = 0
        }
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
        if (nativeClientPtr != 0L) {
            try {
                val jsonResult = suspendCoroutine<String?> { cont ->
                    NativeBridge.parentcontrol_fetch_status_json(nativeClientPtr) { json, _ ->
                        cont.resume(json)
                    }
                }
                if (jsonResult != null) {
                    val status = gson.fromJson(jsonResult, SystemStatus::class.java)
                    _status.value = status
                    return@withContext Result.success(status)
                }
            } catch (ignored: Throwable) {}
        }

        // Fallback to HTTP REST
        try {
            val request = Request.Builder()
                .url("$baseUrl/api/v1/status")
                .get()
                .build()
            val response = httpClient.newCall(request).execute()
            val body = response.body?.string() ?: "{}"
            val status = gson.fromJson(body, SystemStatus::class.java)
            _status.value = status
            Result.success(status)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchMembers(): Result<List<Member>> = withContext(Dispatchers.IO) {
        try {
            val request = Request.Builder()
                .url("$baseUrl/api/v1/members")
                .get()
                .build()
            val response = httpClient.newCall(request).execute()
            val body = response.body?.string() ?: "[]"
            val type = object : TypeToken<List<Member>>() {}.type
            val memberList: List<Member> = gson.fromJson(body, type) ?: emptyList()
            _members.value = memberList
            Result.success(memberList)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchDevices(): Result<List<Device>> = withContext(Dispatchers.IO) {
        try {
            val request = Request.Builder()
                .url("$baseUrl/api/v1/devices")
                .get()
                .build()
            val response = httpClient.newCall(request).execute()
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
        if (nativeClientPtr != 0L) {
            try {
                val success = suspendCoroutine<Boolean> { cont ->
                    NativeBridge.parentcontrol_lock_member(nativeClientPtr, memberId) { ok, _ ->
                        cont.resume(ok)
                    }
                }
                if (success) {
                    fetchMembers()
                    return@withContext Result.success(true)
                }
            } catch (ignored: Throwable) {}
        }

        // Fallback HTTP
        try {
            val action = if (lock) "lock" else "unlock"
            val request = Request.Builder()
                .url("$baseUrl/api/v1/members/$memberId/$action")
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

    fun close() {
        if (nativeClientPtr != 0L) {
            try {
                NativeBridge.parentcontrol_destroy_client(nativeClientPtr)
            } catch (ignored: Throwable) {}
            nativeClientPtr = 0
        }
    }
}
