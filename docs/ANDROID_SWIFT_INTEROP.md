# Android Swift Interoperability Architecture Guide (Shared Swift Core)

[English](ANDROID_SWIFT_INTEROP.md) | [简体中文](ANDROID_SWIFT_INTEROP_zh.md)

This project adopts a **Shared Swift Core** architecture: core data models, network client, router auto-discovery, quota countdowns, and bonus algorithms are all unified in `ParentControlCore`. The iOS client uses it directly, while Android reuses it seamlessly via **C-FFI / JNI**.

---

## 1. Architecture Diagram

```mermaid
flowchart TD
    subgraph Shared_Core [Shared Swift Core (ParentControlCore)]
        Models[Models.swift Strongly-typed Models]
        Network[ParentControlClient.swift Async Networking]
        Discovery[RouterDiscovery.swift Router Auto-Discovery]
        Bridge[ParentControlBridge.swift C-FFI / JNI Export]
    end

    subgraph iOS_App [iOS Native App]
        SwiftUI[SwiftUI Views (Dashboard / Devices / Editor)]
        AppState[AppState.swift Reactive State Stream]
    end

    subgraph Android_App [Android Native App]
        JNI[NativeBridge.kt JNI Shared Lib libParentControlBridge.so]
        KotlinRepo[ParentControlRepository.kt Kotlin Coroutines / Flow]
        Compose[Jetpack Compose UI]
    end

    Models --> AppState
    Network --> AppState
    AppState --> SwiftUI

    Models --> Bridge
    Network --> Bridge
    Bridge --> JNI
    JNI --> KotlinRepo
    KotlinRepo --> Compose
```

---

## 2. Compiling Swift to Android .so Dynamic Shared Libraries

Using the Swift Android Toolchain (or Docker Android NDK Swift builder image), `ParentControlCore` can be compiled into Linux dynamic shared libraries for `arm64-v8a` and `x86_64`:

```bash
# Cross-compile to Android ARM64
swift build \
  --destination destination-aarch64-linux-android.json \
  -c release \
  --product ParentControlBridge
```

The compiled `libParentControlBridge.so` is placed into the Android project's `app/src/main/jniLibs/arm64-v8a/` directory.

---

## 3. Kotlin JNI Invocation & Coroutine Wrapping

### 3.1 JNI Interface Layer (`NativeBridge.kt`)
```kotlin
package com.parentcontrol.core

object NativeBridge {
    init {
        System.loadLibrary("ParentControlBridge")
    }

    external fun createClient(url: String): Long
    external fun destroyClient(clientPtr: Long)
    external fun fetchStatusJson(clientPtr: Long, callback: (json: String?, error: String?) -> Unit)
    external fun lockMember(clientPtr: Long, memberId: String, callback: (success: Boolean, error: String?) -> Unit)
}
```

### 3.2 Kotlin Coroutine Layer (`ParentControlRepository.kt`)
Using `suspendCoroutine` to wrap Swift async callbacks into standard Kotlin coroutines:

```kotlin
suspend fun fetchStatus(): Result<SystemStatus> = suspendCoroutine { continuation ->
    NativeBridge.fetchStatusJson(clientPtr) { json, error ->
        if (json != null) {
            val status = Json.decodeFromString<SystemStatus>(json)
            continuation.resume(Result.success(status))
        } else {
            continuation.resume(Result.failure(Exception(error ?: "Unknown error")))
        }
    }
}
```
