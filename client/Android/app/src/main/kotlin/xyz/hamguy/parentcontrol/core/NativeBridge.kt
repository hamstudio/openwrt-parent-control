package xyz.hamguy.parentcontrol.core

/**
 * JNI Cross-platform interface: loads libParentControlBridge.so compiled from Swift/C
 */
object NativeBridge {
    init {
        try {
            System.loadLibrary("ParentControlBridge")
        } catch (e: UnsatisfiedLinkError) {
            // Fallback in pure Java environments without native shared libraries
            println("ParentControlBridge native library not loaded: ${e.message}")
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
