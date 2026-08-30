package xyz.hamguy.parentcontrol.repository

import android.content.Context
import android.content.SharedPreferences
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import xyz.hamguy.parentcontrol.model.*
import java.util.concurrent.TimeUnit

class ParentControlRepository(
    private val context: Context? = null
) {
    private val prefs: SharedPreferences? = context?.getSharedPreferences("parent_control_prefs", Context.MODE_PRIVATE)

    var baseUrl: String = prefs?.getString("server_url", "http://192.168.0.110:8088") ?: "http://192.168.0.110:8088"
        set(value) {
            field = value
            prefs?.edit()?.putString("server_url", value)?.apply()
        }

    var pinCode: String = prefs?.getString("pin_code", "") ?: ""
        set(value) {
            field = value
            prefs?.edit()?.putString("pin_code", value)?.apply()
        }

    var appLanguage: String = prefs?.getString("app_lang", "auto") ?: "auto"
        set(value) {
            field = value
            prefs?.edit()?.putString("app_lang", value)?.apply()
        }

    private val gson = Gson()
    private val httpClient = OkHttpClient.Builder()
        .connectTimeout(3, TimeUnit.SECONDS)
        .readTimeout(5, TimeUnit.SECONDS)
        .writeTimeout(5, TimeUnit.SECONDS)
        .build()

    private val _status = MutableStateFlow<SystemStatus?>(null)
    val status: StateFlow<SystemStatus?> = _status.asStateFlow()

    private val _members = MutableStateFlow<List<Member>>(emptyList())
    val members: StateFlow<List<Member>> = _members.asStateFlow()

    private val _devices = MutableStateFlow<List<Device>>(emptyList())
    val devices: StateFlow<List<Device>> = _devices.asStateFlow()

    private val _categories = MutableStateFlow<List<AppCategory>>(emptyList())
    val categories: StateFlow<List<AppCategory>> = _categories.asStateFlow()

    private val _settings = MutableStateFlow(GlobalSettings())
    val settings: StateFlow<GlobalSettings> = _settings.asStateFlow()

    private val _isConnected = MutableStateFlow(false)
    val isConnected: StateFlow<Boolean> = _isConnected.asStateFlow()

    private val _needsPinAuth = MutableStateFlow(false)
    val needsPinAuth: StateFlow<Boolean> = _needsPinAuth.asStateFlow()

    private val _isLoading = MutableStateFlow(false)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    private val _errorMessage = MutableStateFlow<String?>(null)
    val errorMessage: StateFlow<String?> = _errorMessage.asStateFlow()

    private var pollJob: Job? = null
    private val repoScope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    init {
        startAutoRefresh()
    }

    fun startAutoRefresh() {
        pollJob?.cancel()
        pollJob = repoScope.launch {
            while (isActive) {
                if (_isConnected.value && !_needsPinAuth.value) {
                    refreshLightweight()
                }
                delay(4000)
            }
        }
    }

    fun stopAutoRefresh() {
        pollJob?.cancel()
        pollJob = null
    }

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
        _errorMessage.value = null
        try {
            val statusResult = fetchStatus()
            if (statusResult.isSuccess) {
                val s = statusResult.getOrNull()
                if (s?.pinRequired == true && pinCode.isEmpty()) {
                    _needsPinAuth.value = true
                    _errorMessage.value = "PIN protection is enabled. Enter PIN to load data."
                    return@withContext
                }
                supervisorScope {
                    val mJob = launch { fetchMembers() }
                    val dJob = launch { fetchDevices() }
                    val cJob = launch { fetchAppCategories() }
                    val sJob = launch { fetchSettings() }
                    mJob.join()
                    dJob.join()
                    cJob.join()
                    sJob.join()
                }
            }
        } catch (e: Exception) {
            _errorMessage.value = e.localizedMessage
        } finally {
            _isLoading.value = false
        }
    }

    suspend fun refreshLightweight() = withContext(Dispatchers.IO) {
        try {
            fetchStatus()
            fetchMembers()
            fetchDevices()
        } catch (_: Exception) {}
    }

    suspend fun fetchStatus(): Result<SystemStatus> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/status").get().build()
            httpClient.newCall(request).execute().use { response ->
                val body = response.body?.string() ?: "{}"
                val s = gson.fromJson(body, SystemStatus::class.java)
                _status.value = s
                _isConnected.value = true
                if (s.pinRequired && pinCode.isEmpty()) {
                    _needsPinAuth.value = true
                }
                Result.success(s)
            }
        } catch (e: Exception) {
            _isConnected.value = false
            _errorMessage.value = e.localizedMessage
            Result.failure(e)
        }
    }

    suspend fun fetchMembers(): Result<List<Member>> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/members").get().build()
            httpClient.newCall(request).execute().use { response ->
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
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchDevices(): Result<List<Device>> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/devices").get().build()
            httpClient.newCall(request).execute().use { response ->
                if (response.code == 401) {
                    _needsPinAuth.value = true
                    return@withContext Result.failure(Exception("401 PIN code required"))
                }
                val body = response.body?.string() ?: "[]"
                val type = object : TypeToken<List<Device>>() {}.type
                val deviceList: List<Device> = gson.fromJson(body, type) ?: emptyList()
                _devices.value = deviceList
                Result.success(deviceList)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchAppCategories(): Result<List<AppCategory>> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/categories").get().build()
            httpClient.newCall(request).execute().use { response ->
                val body = response.body?.string() ?: "[]"
                val type = object : TypeToken<List<AppCategory>>() {}.type
                val catList: List<AppCategory> = gson.fromJson(body, type) ?: emptyList()
                _categories.value = catList
                Result.success(catList)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchSettings(): Result<GlobalSettings> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/settings").get().build()
            httpClient.newCall(request).execute().use { response ->
                if (response.code == 401) {
                    _needsPinAuth.value = true
                    return@withContext Result.failure(Exception("401 PIN required"))
                }
                val body = response.body?.string() ?: "{}"
                val set = gson.fromJson(body, GlobalSettings::class.java) ?: GlobalSettings()
                _settings.value = set
                Result.success(set)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun saveSettings(newSettings: GlobalSettings): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val jsonBody = gson.toJson(newSettings)
            val request = newRequestBuilder("/api/settings")
                .post(jsonBody.toRequestBody("application/json".toMediaType()))
                .build()
            httpClient.newCall(request).execute().use { response ->
                val ok = response.isSuccessful
                if (ok) {
                    _settings.value = newSettings
                }
                Result.success(ok)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun saveMember(member: Member): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val jsonBody = gson.toJson(member)
            val request = newRequestBuilder("/api/members")
                .post(jsonBody.toRequestBody("application/json".toMediaType()))
                .build()
            httpClient.newCall(request).execute().use { response ->
                val ok = response.isSuccessful
                if (ok) {
                    fetchMembers()
                    fetchDevices()
                }
                Result.success(ok)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteMember(memberId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/members/$memberId")
                .delete()
                .build()
            httpClient.newCall(request).execute().use { response ->
                val ok = response.isSuccessful
                if (ok) {
                    fetchMembers()
                    fetchDevices()
                }
                Result.success(ok)
            }
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
            httpClient.newCall(request).execute().use { response ->
                val ok = response.isSuccessful
                if (ok) fetchMembers()
                Result.success(ok)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun bonusMember(memberId: String, minutes: Int = 30): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val request = newRequestBuilder("/api/members/$memberId/bonus?minutes=$minutes")
                .post("{}".toRequestBody("application/json".toMediaType()))
                .build()
            httpClient.newCall(request).execute().use { response ->
                val ok = response.isSuccessful
                if (ok) fetchMembers()
                Result.success(ok)
            }
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
            httpClient.newCall(request).execute().use { response ->
                val ok = response.isSuccessful
                if (ok) fetchDevices()
                Result.success(ok)
            }
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
            httpClient.newCall(request).execute().use { response ->
                val ok = response.isSuccessful
                if (ok) {
                    fetchMembers()
                    fetchDevices()
                }
                Result.success(ok)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun autoDiscover(): String? = withContext(Dispatchers.IO) {
        val candidates = listOf(
            "http://192.168.0.110:8088",
            "http://192.168.1.1:8088",
            "http://192.168.0.1:8088",
            "http://192.168.31.1:8088",
            "http://10.0.0.1:8088",
            "http://192.168.2.1:8088"
        )
        for (candidate in candidates) {
            try {
                val req = Request.Builder()
                    .url("$candidate/api/status")
                    .build()
                val isFound = httpClient.newCall(req).execute().use { res ->
                    if (res.isSuccessful) {
                        baseUrl = candidate
                        true
                    } else false
                }
                if (isFound) {
                    refreshAll()
                    return@withContext candidate
                }
            } catch (_: Exception) {}
        }
        null
    }
}
