package com.parentcontrol.core

/**
 * JNI 跨平台接口层：直接加载 Swift 编译生成的 libParentControlBridge.so
 */
object NativeBridge {
    init {
        try {
            System.loadLibrary("ParentControlBridge")
        } catch (e: UnsatisfiedLinkError) {
            println("ParentControlBridge library not loaded: ${e.message}")
        }
    }

    @JvmStatic
    external fun parentcontrol_create_client(url: String): Long

    @JvmStatic
    external fun parentcontrol_destroy_client(clientPtr: Long)

    @JvmStatic
    external fun parentcontrol_fetch_status_json(
        clientPtr: Long,
        callback: (json: String?, error: String?) -> Unit
    )

    @JvmStatic
    external fun parentcontrol_lock_member(
        clientPtr: Long,
        memberId: String,
        callback: (success: Boolean, error: String?) -> Unit
    )
}
