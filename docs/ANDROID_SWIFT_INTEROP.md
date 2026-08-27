# Android 端复用 Swift 核心逻辑架构指南 (Android Swift Interop)

[English](ANDROID_SWIFT_INTEROP.md) | [简体中文](ANDROID_SWIFT_INTEROP_zh.md)

本项目采用 **Shared Swift Core (通用 Swift 业务层)** 架构：核心数据模型、网络请求、路由器自动探测、配额倒计时与加时算法全部由 `ParentControlCore` 统一实现，iOS 直接使用，Android 通过 **C-FFI / JNI** 无缝复用。

---

## 1. 架构示意图

```mermaid
flowchart TD
    subgraph Shared_Core [通用 Swift 核心层 (ParentControlCore)]
        Models[Models.swift 强类型数据模型]
        Network[ParentControlClient.swift 异步网络引擎]
        Discovery[RouterDiscovery.swift 路由器自动探测]
        Bridge[ParentControlBridge.swift C-FFI / JNI 导出接口]
    end

    subgraph iOS_App [iOS 原生客户端]
        SwiftUI[SwiftUI 界面 (Dashboard / Devices / Editor)]
        AppState[AppState.swift 响应式状态流]
    end

    subgraph Android_App [Android 原生客户端]
        JNI[NativeBridge.kt JNI 动态链接库 libParentControlBridge.so]
        KotlinRepo[ParentControlRepository.kt Kotlin 协程 / Flow 封装]
        Compose[Jetpack Compose 响应式界面]
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

## 2. 编译 Swift 为 Android .so 动态链接库

通过 Swift Android Toolchain（或 Docker Android NDK Swift 构建镜像），可以轻松将 `ParentControlCore` 编译为 `arm64-v8a` 和 `x86_64` 的 Linux 动态链接库：

```bash
# 交叉编译到 Android ARM64
swift build \
  --destination destination-aarch64-linux-android.json \
  -c release \
  --product ParentControlBridge
```

编译输出的 `libParentControlBridge.so` 直接放入 Android 工程的 `app/src/main/jniLibs/arm64-v8a/` 目录下。

---

## 3. Kotlin JNI 调用与 Coroutine 封装

### 3.1 JNI 接口层 (`NativeBridge.kt`)
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

### 3.2 Kotlin 协程层 (`ParentControlRepository.kt`)
通过 `suspendCoroutine` 将 Swift 异步回调转换为 Kotlin 标准协程：

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
