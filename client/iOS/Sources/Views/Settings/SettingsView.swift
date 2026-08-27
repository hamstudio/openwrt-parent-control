import SwiftUI
import ParentControlCore

struct SettingsView: View {
    @EnvironmentObject var appState: AppState
    @State private var serverAddress: String = ""
    @State private var globalEnabled: Bool = true
    @State private var safeSearch: Bool = true
    @State private var blockDoH: Bool = true
    @State private var isolateNew: Bool = false
    @State private var isDiscovering: Bool = false
    @State private var isSaving: Bool = false

    var body: some View {
        NavigationStack {
            Form {
                // 路由器直连配置
                Section("路由器连接设置") {
                    HStack {
                        #if os(iOS)
                        TextField("http://192.168.0.110:8088", text: $serverAddress)
                            .keyboardType(.URL)
                            .textInputAutocapitalization(.never)
                            .disableAutocorrection(true)
                        #else
                        TextField("http://192.168.0.110:8088", text: $serverAddress)
                            .disableAutocorrection(true)
                        #endif

                        Button("保存") {
                            appState.serverURL = serverAddress
                            HapticManager.notification(.success)
                        }
                        .bold()
                        .foregroundColor(.guardianGreen)
                    }

                    Button {
                        isDiscovering = true
                        HapticManager.impact(.medium)
                        Task {
                            await appState.autoDiscover()
                            serverAddress = appState.serverURL
                            isDiscovering = false
                        }
                    } label: {
                        HStack {
                            Image(systemName: "antenna.radiowaves.left.and.right")
                            Text("局域网自动探测路由器")
                            if isDiscovering {
                                Spacer()
                                ProgressView()
                            }
                        }
                        .foregroundColor(.guardianGreen)
                    }
                }

                // 全局安全策略
                Section("全局健康守护与防绕过策略") {
                    Toggle("家长控制系统总开关", isOn: $globalEnabled)
                        .tint(.guardianGreen)
                    Toggle("强制 SafeSearch 安全搜索", isOn: $safeSearch)
                        .tint(.guardianGreen)
                    Toggle("阻断公共 DoH/DoT (防改 DNS 绕过)", isOn: $blockDoH)
                        .tint(.guardianGreen)
                    Toggle("新设备隔离审批 (防 MAC 随机化)", isOn: $isolateNew)
                        .tint(.guardianGreen)

                    Button("应用并下发全局健康守护设置") {
                        saveSettings()
                    }
                    .bold()
                    .foregroundColor(.guardianGreen)
                    .frame(maxWidth: .infinity)
                }

                // 系统与运行状态
                Section("系统信息") {
                    LabeledContent("客户端版本", value: "1.0.0 (Native)")
                    LabeledContent("DPI 引擎", value: appState.status?.kernelDpiReady == true ? "kmod-oaf (正常)" : "未激活")
                    LabeledContent("系统运行时间", value: "\(appState.status?.uptimeSeconds ?? 0) 秒")
                    LabeledContent("已探测设备总数", value: "\(appState.status?.totalDevices ?? 0) 台")
                }
            }
            .navigationTitle("安全设置")
            .onAppear {
                serverAddress = appState.serverURL
                globalEnabled = appState.settings.enabled
                safeSearch = appState.settings.enforceSafeSearch
                blockDoH = appState.settings.blockDoHDoT
                isolateNew = appState.settings.isolateNewDevices
            }
        }
    }

    private func saveSettings() {
        isSaving = true
        let newSettings = GlobalSettings(
            enabled: globalEnabled,
            enforceSafeSearch: safeSearch,
            blockDoHDoT: blockDoH,
            isolateNewDevices: isolateNew
        )

        Task {
            let success = await appState.saveSettings(newSettings)
            if success {
                HapticManager.notification(.success)
            }
            isSaving = false
        }
    }
}
