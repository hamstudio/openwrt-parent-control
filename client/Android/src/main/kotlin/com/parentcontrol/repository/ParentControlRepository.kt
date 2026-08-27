package com.parentcontrol.repository

import com.parentcontrol.core.NativeBridge
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import kotlin.coroutines.resume
import kotlin.coroutines.suspendCoroutine

data class SystemStatus(
    val running: Boolean,
    val uptimeSeconds: Long,
    val totalDevices: Int,
    val activeDevices: Int,
    val managedMembers: Int,
    val kernelDpiReady: Boolean,
    val appCount: Int
)

class ParentControlRepository(baseUrl: String = "http://192.168.0.110:8088") {
    private var clientPtr: Long = 0

    private val _statusFlow = MutableStateFlow<SystemStatus?>(null)
    val statusFlow: StateFlow<SystemStatus?> = _statusFlow.asStateFlow()

    init {
        clientPtr = NativeBridge.parentcontrol_create_client(baseUrl)
    }

    suspend fun fetchStatus(): Result<String> = withContext(Dispatchers.IO) {
        suspendCoroutine { continuation ->
            NativeBridge.parentcontrol_fetch_status_json(clientPtr) { json, error ->
                if (json != null) {
                    continuation.resume(Result.success(json))
                } else {
                    continuation.resume(Result.failure(Exception(error ?: "Unknown error")))
                }
            }
        }
    }

    suspend fun lockMember(memberId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        suspendCoroutine { continuation ->
            NativeBridge.parentcontrol_lock_member(clientPtr, memberId) { success, error ->
                if (success) {
                    continuation.resume(Result.success(true))
                } else {
                    continuation.resume(Result.failure(Exception(error ?: "Failed to lock member")))
                }
            }
        }
    }

    fun close() {
        if (clientPtr != 0L) {
            NativeBridge.parentcontrol_destroy_client(clientPtr)
            clientPtr = 0
        }
    }
}
